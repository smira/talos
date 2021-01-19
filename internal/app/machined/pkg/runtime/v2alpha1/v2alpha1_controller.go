// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v2alpha1

import (
	"context"
	"log"

	"github.com/talos-systems/os-runtime/pkg/controller"
	osruntime "github.com/talos-systems/os-runtime/pkg/controller/runtime"

	"github.com/talos-systems/talos/internal/app/machined/pkg/controllers/config"
	"github.com/talos-systems/talos/internal/app/machined/pkg/controllers/k8s"
	"github.com/talos-systems/talos/internal/app/machined/pkg/controllers/legacy"
	"github.com/talos-systems/talos/internal/app/machined/pkg/runtime"
)

// Controller implements runtime.V2Controller.
type Controller struct {
	controllerRuntime *osruntime.Runtime

	legacyRuntime runtime.Runtime
}

// NewController creates Controller.
func NewController(legacyRuntime runtime.Runtime, loggingManager runtime.LoggingManager) (*Controller, error) {
	ctrl := &Controller{
		legacyRuntime: legacyRuntime,
	}

	logWriter, err := loggingManager.ServiceLog("controller-runtime").Writer()
	if err != nil {
		return nil, err
	}

	logger := log.New(logWriter, "controller-runtime: ", log.Flags())

	ctrl.controllerRuntime, err = osruntime.NewRuntime(legacyRuntime.State().V2().Resources(), logger)

	return ctrl, err
}

// Run the controller runtime.
func (ctrl *Controller) Run(ctx context.Context) error {
	for _, c := range []controller.Controller{
		&legacy.ServiceController{
			LegacyEvents: ctrl.legacyRuntime.Events(),
		},
		&config.MachineTypeController{},
		&config.K8sControlPlaneController{},
		&k8s.ControlPlaneStaticPodController{},
		&k8s.KubeletStaticPodController{},
	} {
		if err := ctrl.controllerRuntime.RegisterController(c); err != nil {
			return err
		}
	}

	return ctrl.controllerRuntime.Run(ctx)
}
