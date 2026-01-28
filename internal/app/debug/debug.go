// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package debug implements machine.DebugService.
package debug

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/leases"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/siderolabs/talos/internal/app/internal/ctrhelper"
	"github.com/siderolabs/talos/internal/app/machined/pkg/system/runner/containerd"
	"github.com/siderolabs/talos/internal/pkg/capability"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
)

// Service implements machine.DebugService.
type Service struct {
	machine.UnimplementedDebugServiceServer
}

// ContainerRun implements machine.DebugService.ContainerRun.
func (s *Service) ContainerRun(srv grpc.BidiStreamingServer[machine.DebugContainerRunRequest, machine.DebugContainerRunResponse]) error { //nolint:gocyclo
	ctx := srv.Context()

	// 1. get the debug container spec
	specReq, err := srv.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive spec: %v", err)
	}

	spec := specReq.GetSpec()
	if spec == nil {
		return status.Errorf(codes.InvalidArgument, "expected debug container spec")
	}

	log.Printf("debug container request received: image=%s args=%v", spec.ImageName, spec.Args)

	ctx, c8dClient, err := ctrhelper.ContainerdInstanceHelper(ctx, spec.GetContainerd())
	if err != nil {
		return err
	}
	defer c8dClient.Close() //nolint:errcheck

	l, err := c8dClient.LeasesService().Create(ctx,
		leases.WithRandomID(),
	)
	if err != nil {
		return fmt.Errorf("failed to create lease: %v", err)
	}

	defer func() {
		if err := c8dClient.LeasesService().Delete(context.Background(), l, leases.SynchronousDelete); err != nil {
			log.Printf("failed to delete lease %s: %v", l.ID, err)
		}
	}()

	ctx = leases.WithLease(ctx, l.ID)

	img, err := c8dClient.GetImage(ctx, spec.ImageName)
	if err != nil {
		return err
	}

	ctr, err := createDebugContainer(ctx, c8dClient, img, spec)
	if err != nil {
		return err
	}

	defer func() {
		cleanupErr := ctr.Delete(context.Background(), client.WithSnapshotCleanup)
		if cleanupErr != nil {
			log.Printf("debug container: failed to delete container %s: %s", ctr.ID(), cleanupErr.Error())
		}

		log.Printf("debug container: container %s deleted", ctr.ID())
	}()

	return runAndAttachContainer(ctx, srv, ctr)
}

func createDebugContainer(
	ctx context.Context,
	c8dClient *client.Client,
	image client.Image,
	spec *machine.DebugContainerRunRequestSpec,
) (client.Container, error) {
	ociOpts := []oci.SpecOpts{
		oci.WithDefaultSpec(),
		oci.WithDefaultUnixDevices,
		oci.WithHostNamespace(specs.NetworkNamespace),
		oci.WithHostNamespace(specs.PIDNamespace),
		oci.WithHostNamespace(specs.IPCNamespace),
		oci.WithTTY,
		oci.WithHostDevices,
		oci.WithAllDevicesAllowed,
		oci.WithHostHostsFile,
		oci.WithWriteableSysfs,
		oci.WithCapabilities(capability.AllGrantableCapabilities()),
		oci.WithHostResolvconf,
		oci.WithMounts([]specs.Mount{
			// mount host / under /host
			{
				Destination: "/host",
				Type:        "bind",
				Source:      "/",
				Options:     []string{"rbind", "rw"},
			},
			{
				Destination: "/sys",
				Type:        "bind",
				Source:      "/sys",
				Options:     []string{"rbind", "rw"},
			},
		}),
		oci.WithSelinuxLabel(""),
		oci.WithApparmorProfile(""),
		oci.WithSeccompUnconfined,
		oci.WithImageConfig(image),
	}

	if len(spec.Args) > 0 {
		ociOpts = append(ociOpts, oci.WithProcessArgs(spec.Args...))
	}

	if len(spec.Env) > 0 {
		envVars := make([]string, 0, len(spec.Env))

		for k, v := range spec.Env {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}

		ociOpts = append(ociOpts, oci.WithEnv(envVars))
	}

	containerID, err := generateContainerID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate container ID: %v", err)
	}

	container, err := c8dClient.NewContainer(ctx, containerID,
		client.WithImage(image),
		client.WithNewSnapshot(containerID+"-snapshot", image),
		client.WithNewSpec(ociOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %v", err)
	}

	return container, nil
}

func runAndAttachContainer(
	ctx context.Context,
	srv grpc.BidiStreamingServer[machine.DebugContainerRunRequest, machine.DebugContainerRunResponse],
	ctr client.Container,
) error {
	grpcStreamer, stdinR, stdoutW := newGrpcStreamWriter(srv)
	stdin := &containerd.StdinCloser{
		Stdin:  stdinR,
		Closer: make(chan struct{}),
	}

	cIo := cio.NewCreator(cio.WithStreams(stdin, stdoutW, stdoutW), cio.WithTerminal)

	task, err := ctr.NewTask(ctx, cIo)
	if err != nil {
		return fmt.Errorf("failed to create task: %v", err)
	}

	defer func() {
		_, err := task.Delete(context.Background(), client.WithProcessKill)
		if err != nil {
			log.Printf("debug container: failed to delete task: %s", err.Error())
		}
	}()

	go stdin.WaitAndClose(context.Background(), task)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := task.Start(ctx); err != nil {
		return fmt.Errorf("failed to start task: %v", err)
	}

	statusC, err := task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for task: %v", err)
	}

	grpcStreamer.stream(statusC, task)

	return nil
}

func generateContainerID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}

	return fmt.Sprintf("debug-%s", hex.EncodeToString(b)), nil
}
