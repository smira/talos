// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package legacy

import (
	"context"
	"log"
	"sync"

	"github.com/talos-systems/os-runtime/pkg/controller"
	"github.com/talos-systems/os-runtime/pkg/resource"
	"github.com/talos-systems/os-runtime/pkg/state"

	"github.com/talos-systems/talos/internal/app/machined/pkg/resources/legacy"
	"github.com/talos-systems/talos/internal/app/machined/pkg/runtime"
	"github.com/talos-systems/talos/pkg/machinery/api/machine"
)

// ServiceController manages legacy.Service based on services subsystem state.
type ServiceController struct {
	LegacyEvents runtime.Watcher
}

// Name implements controller.Controller interface.
func (ctrl *ServiceController) Name() string {
	return "legacy.ServiceController"
}

// ManagedResources implements controller.Controller interface.
func (ctrl *ServiceController) ManagedResources() (resource.Namespace, resource.Type) {
	return legacy.NamespaceName, legacy.ServiceType
}

// Run implements controller.Controller interface.
//
//nolint: gocyclo
func (ctrl *ServiceController) Run(ctx context.Context, r controller.Runtime, logger *log.Logger) error {
	var wg sync.WaitGroup

	wg.Add(1)

	if err := ctrl.LegacyEvents.Watch(func(eventCh <-chan runtime.Event) {
		defer wg.Done()

		for {
			var (
				event runtime.Event
				ok    bool
			)

			select {
			case <-ctx.Done():
				return
			case event, ok = <-eventCh:
				if !ok {
					return
				}
			}

			if msg, ok := event.Payload.(*machine.ServiceStateEvent); ok {
				service := legacy.NewService(msg.Service)

				switch msg.Action { //nolint: exhaustive
				case machine.ServiceStateEvent_RUNNING:
					if err := r.Update(ctx, service, func(r resource.Resource) error {
						r.(*legacy.Service).SetRunning(true)

						return nil
					}); err != nil {
						logger.Printf("failed creating service resource %s: %s", service, err)
					}
				default:
					if err := r.Destroy(ctx, service.Metadata()); err != nil && !state.IsNotFoundError(err) {
						logger.Printf("failed destroying service resource %s: %s", service, err)
					}
				}
			}
		}
	}, runtime.WithTailEvents(-1)); err != nil {
		return err
	}

	wg.Wait()

	return nil
}
