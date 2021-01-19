// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runtime

import (
	"github.com/talos-systems/go-blockdevice/blockdevice/probe"
	"github.com/talos-systems/os-runtime/pkg/state"
	"github.com/talos-systems/os-runtime/pkg/state/registry"

	"github.com/talos-systems/talos/pkg/machinery/config"
)

// State defines the state.
type State interface {
	Platform() Platform
	Machine() MachineState
	Cluster() ClusterState
	V2() V2State
}

// Machine defines the runtime parameters.
type Machine interface {
	State() MachineState
	Config() config.MachineConfig
}

// MachineState defines the machined state.
type MachineState interface {
	Disk() *probe.ProbedBlockDevice
	Close() error
	Installed() bool
	IsInstallStaged() bool
	StagedInstallImageRef() string
	StagedInstallOptions() []byte
}

// ClusterState defines the cluster state.
type ClusterState interface{}

// V2State defines the next generation (v2) interface binding into v1 runtime.
type V2State interface {
	Resources() state.State

	NamespaceRegistry() *registry.NamespaceRegistry
	ResourceRegistry() *registry.ResourceRegistry

	SetConfig(config.Provider) error
}
