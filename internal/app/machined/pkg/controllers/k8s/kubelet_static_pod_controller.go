// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/AlekSi/pointer"
	"github.com/talos-systems/os-runtime/pkg/controller"
	"github.com/talos-systems/os-runtime/pkg/resource"
	"github.com/talos-systems/os-runtime/pkg/state"
	"gopkg.in/yaml.v3"

	"github.com/talos-systems/talos/internal/app/machined/pkg/resources/k8s"
	"github.com/talos-systems/talos/internal/app/machined/pkg/resources/legacy"
	"github.com/talos-systems/talos/pkg/machinery/constants"
)

// KubeletStaticPodController renders static pod definitions and manages k8s.StaticPodStatus.
type KubeletStaticPodController struct {
}

// Name implements controller.Controller interface.
func (ctrl *KubeletStaticPodController) Name() string {
	return "k8s.KubeletStaticPodController"
}

// ManagedResources implements controller.Controller interface.
func (ctrl *KubeletStaticPodController) ManagedResources() (resource.Namespace, resource.Type) {
	return k8s.ControlPlaneNamespaceName, k8s.StaticPodStatusType
}

// Run implements controller.Controller interface.
//
//nolint: gocyclo
func (ctrl *KubeletStaticPodController) Run(ctx context.Context, r controller.Runtime, logger *log.Logger) error {
	if err := r.UpdateDependencies([]controller.Dependency{
		{
			Namespace: k8s.ControlPlaneNamespaceName,
			Type:      k8s.StaticPodType,
			Kind:      controller.DependencyHard,
		},
		{
			Namespace: legacy.NamespaceName,
			Type:      legacy.ServiceType,
			ID:        pointer.ToString("kubelet"),
			Kind:      controller.DependencyWeak,
		},
	}); err != nil {
		return fmt.Errorf("error setting up dependencies: %w", err)
	}

	defer logger.Print("exited the loop")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		}

		kubeletResource, err := r.Get(ctx, resource.NewMetadata(legacy.NamespaceName, legacy.ServiceType, "kubelet", resource.VersionUndefined))
		if err != nil {
			if state.IsNotFoundError(err) {
				if err = ctrl.teardownAll(ctx, r, logger); err != nil {
					return fmt.Errorf("error tearing down: %w", err)
				}

				continue
			}

			return err
		}

		if !kubeletResource.(*legacy.Service).Running() {
			if err = ctrl.teardownAll(ctx, r, logger); err != nil {
				return fmt.Errorf("error tearing down: %w", err)
			}

			continue
		}

		staticPods, err := r.List(ctx, resource.NewMetadata(k8s.ControlPlaneNamespaceName, k8s.StaticPodType, "", resource.VersionUndefined))
		if err != nil {
			return fmt.Errorf("error listing static pods: %w", err)
		}

		for _, staticPod := range staticPods.Items {
			switch staticPod.Metadata().Phase() {
			case resource.PhaseRunning:
				if err = ctrl.runPod(ctx, r, logger, staticPod.(*k8s.StaticPod)); err != nil {
					return fmt.Errorf("error running pod: %w", err)
				}
			case resource.PhaseTearingDown:
				if err = ctrl.teardownPod(ctx, r, logger, staticPod.(*k8s.StaticPod)); err != nil {
					return fmt.Errorf("error tearing down pod: %w", err)
				}
			}
		}
	}
}

func (ctrl *KubeletStaticPodController) runPod(ctx context.Context, r controller.Runtime, logger *log.Logger, staticPod *k8s.StaticPod) error {
	staticPodStatus := k8s.NewStaticPodStatus(staticPod.Metadata().Namespace(), staticPod.Metadata().ID())

	if err := r.AddFinalizer(ctx, staticPod.Metadata(), staticPodStatus.String()); err != nil {
		return err
	}

	logger.Print("added finalizer")

	renderedPod, err := yaml.Marshal(staticPod.Spec())
	if err != nil {
		return nil
	}

	logger.Print("marshaled")

	podPath := filepath.Join(constants.ManifestsDirectory, fmt.Sprintf("%s.yaml", staticPod.Metadata().ID()))

	logger.Print("podPath", podPath)

	existingPod, err := ioutil.ReadFile(podPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	if bytes.Equal(renderedPod, existingPod) {
		return nil
	}

	logger.Printf("rendered static pod definition %q", podPath)

	return ioutil.WriteFile(podPath, renderedPod, 0o600)
}

func (ctrl *KubeletStaticPodController) teardownPod(ctx context.Context, r controller.Runtime, logger *log.Logger, staticPod *k8s.StaticPod) error {
	return nil
}

func (ctrl *KubeletStaticPodController) teardownAll(ctx context.Context, r controller.Runtime, logger *log.Logger) error {
	return nil
}
