// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

import (
	"context"
	"fmt"
	"log"

	"github.com/AlekSi/pointer"
	"github.com/talos-systems/os-runtime/pkg/controller"
	"github.com/talos-systems/os-runtime/pkg/resource"
	"github.com/talos-systems/os-runtime/pkg/state"

	"github.com/talos-systems/talos/internal/app/machined/pkg/resources/config"
	talosconfig "github.com/talos-systems/talos/pkg/machinery/config"
)

// K8sControlPlaneController manages config.K8sControlPlane based on configuration.
type K8sControlPlaneController struct {
}

// Name implements controller.Controller interface.
func (ctrl *K8sControlPlaneController) Name() string {
	return "config.K8sControlPlaneController"
}

// ManagedResources implements controller.Controller interface.
func (ctrl *K8sControlPlaneController) ManagedResources() (resource.Namespace, resource.Type) {
	return config.NamespaceName, config.K8sControlPlaneType
}

// Run implements controller.Controller interface.
func (ctrl *K8sControlPlaneController) Run(ctx context.Context, r controller.Runtime, logger *log.Logger) error {
	if err := r.UpdateDependencies([]controller.Dependency{
		{
			Namespace: config.NamespaceName,
			Type:      config.V1Alpha1Type,
			ID:        pointer.ToString(config.V1Alpha1ID),
			Kind:      controller.DependencyWeak,
		},
	}); err != nil {
		return fmt.Errorf("error setting up dependencies: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		}

		cfg, err := r.Get(ctx, resource.NewMetadata(config.NamespaceName, config.V1Alpha1Type, config.V1Alpha1ID, resource.VersionUndefined))
		if err != nil {
			if state.IsNotFoundError(err) {
				continue
			}

			return fmt.Errorf("error getting config: %w", err)
		}

		cfgProvider := cfg.(*config.V1Alpha1).Config()

		for _, f := range []func(context.Context, controller.Runtime, *log.Logger, talosconfig.Provider) error{
			ctrl.manageAPIServerConfig,
		} {
			if err = f(ctx, r, logger, cfgProvider); err != nil {
				return fmt.Errorf("error updating objects: %w", err)
			}
		}
	}
}

func (ctrl *K8sControlPlaneController) manageAPIServerConfig(ctx context.Context, r controller.Runtime, logger *log.Logger, cfgProvider talosconfig.Provider) error {
	return r.Update(ctx, config.NewK8sControlPlaneAPIServer(config.K8sControlPlaneAPIServerSpec{}), func(r resource.Resource) error {
		r.(*config.K8sControlPlane).SetAPIServer(config.K8sControlPlaneAPIServerSpec{
			Image:                cfgProvider.Cluster().APIServer().Image(),
			ControlPlaneEndpoint: cfgProvider.Cluster().Endpoint().String(),
			EtcdServers:          []string{"https://127.0.0.1:2379"},
			LocalPort:            cfgProvider.Cluster().LocalAPIServerPort(),
			ServiceCIDR:          cfgProvider.Cluster().Network().ServiceCIDR(),
			ExtraArgs:            cfgProvider.Cluster().APIServer().ExtraArgs(),
		})

		return nil
	})
}
