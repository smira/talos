// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runtime

/*
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/leases"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/containerd/v2/pkg/snapshotters"
	"github.com/containerd/errdefs"
	"github.com/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/runtime-spec/specs-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/siderolabs/talos/internal/app/machined/pkg/system/runner/containerd"
	"github.com/siderolabs/talos/internal/pkg/capability"
	"github.com/siderolabs/talos/internal/pkg/containers/image"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/constants"
)

type imageCache struct {
	ctrdClient *containerdapi.Client

	mu         sync.Mutex
	images     map[containerdapi.Image]chan struct{}
	containers map[containerdapi.Container]chan struct{}
}

const debugContainerImageTTL = 5 * time.Second

func (ic *imageCache) initClientIfNil() {
	if ic.ctrdClient != nil {
		return
	}

	client, err := containerdapi.New(constants.SystemContainerdAddress,
		containerdapi.WithDefaultNamespace(constants.SystemContainerdNamespace))
	if err != nil {
		log.Printf("failed to connect to system containerd: %s", err)
	}

	ic.ctrdClient = client
}

func (ic *imageCache) addImage(img containerdapi.Image) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	usedC := make(chan struct{})

	go func() {
		timer := time.NewTimer(debugContainerImageTTL)
		select {
		case <-timer.C:
			log.Printf("debug container image TTL expired, deleting image %s", img.Name())

			ic.initClientIfNil()

			err := ic.ctrdClient.ImageService().Delete(context.Background(), img.Name(), images.SynchronousDelete())
			if err != nil {
				log.Printf("failed to delete image %s: %v", img.Name(), err)
			}
		case <-usedC:
			log.Printf("debug container image %s marked as used, skipping deletion", img.Name())
		}
	}()

	ic.images[img] = usedC
}

// TODO: client does image pull or image import RPC request
// then a debug container create RPC
// then a debug container run RPC (but these two might be the same)
// TODO: I guess there is already garbage collection done for
// images in the system namespace? maybe a tag should be added,
// so we can be more aggressive in deleting debug images.

func (ic *imageCache) addContainer(ctr containerdapi.Container) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	usedC := make(chan struct{})

	go func() {
		timer := time.NewTimer(debugContainerImageTTL)
		select {
		case <-timer.C:
			log.Printf("debug container image TTL expired, deleting container %s", ctr.ID())

			ic.initClientIfNil()

			ctr, err := ic.ctrdClient.LoadContainer(context.Background(), ctr.ID())
			if err != nil {
				log.Printf("failed to load container %s: %v", ctr.ID(), err)
			}

			err = ctr.Delete(context.Background(), containerdapi.WithSnapshotCleanup)
			if err != nil {
				log.Printf("failed to delete container %s: %v", ctr.ID(), err)
			}
		case <-usedC:
			log.Printf("debug container image %s marked as used, skipping deletion", ctr.ID())
		}
	}()

	ic.containers[ctr] = usedC
}

func (ic *imageCache) markCtrUsed(ctrID string) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	var (
		usedCtr containerdapi.Container
		usedC   chan struct{}
	)

	for ctr, c := range ic.containers {
		if ctr.ID() == ctrID {
			usedCtr = ctr
			usedC = c
		}
	}

	close(usedC)
	delete(ic.containers, usedCtr)
}

func (ic *imageCache) markImageUsed(imgName string) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	var (
		usedImage containerdapi.Image
		usedC     chan struct{}
	)

	for img, c := range ic.images {
		if img.Name() == imgName {
			usedImage = img
			usedC = c
		}
	}

	close(usedC)
	delete(ic.images, usedImage)
}

var ic = &imageCache{
	mu:         sync.Mutex{},
	images:     map[containerdapi.Image]chan struct{}{},
	containers: map[containerdapi.Container]chan struct{}{},
}



*/
