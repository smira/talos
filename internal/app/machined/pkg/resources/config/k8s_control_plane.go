// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

import (
	"fmt"

	"github.com/talos-systems/os-runtime/pkg/resource"
	"github.com/talos-systems/os-runtime/pkg/resource/core"
)

// K8sControlPlaneType is type of K8sControlPlane resource.
const K8sControlPlaneType = resource.Type("config/k8sControlPlane")

// K8sControlPlaneAPIServerID is an ID .
const K8sControlPlaneAPIServerID = resource.ID("kube-apiserver")

// K8sControlPlane describes machine type.
type K8sControlPlane struct {
	md resource.Metadata
	// spec stores values of different types depending on ID
	spec interface{}
}

// K8sControlPlaneAPIServerSpec is configuration for kube-apiserver.
type K8sControlPlaneAPIServerSpec struct {
	Image                string            `yaml:"image"`
	ControlPlaneEndpoint string            `yaml:"controlPlaneEndpoint"`
	EtcdServers          []string          `yaml:"etcdServers"`
	LocalPort            int               `yaml:"localPort"`
	ServiceCIDR          string            `yaml:"serviceCIDR"`
	ExtraArgs            map[string]string `yaml:"extraArgs"`
}

// NewK8sControlPlaneAPIServer initializes a K8sControlPlane resource.
func NewK8sControlPlaneAPIServer(spec K8sControlPlaneAPIServerSpec) *K8sControlPlane {
	r := &K8sControlPlane{
		md:   resource.NewMetadata(NamespaceName, K8sControlPlaneType, K8sControlPlaneAPIServerID, resource.VersionUndefined),
		spec: spec,
	}

	r.md.BumpVersion()

	return r
}

// Metadata implements resource.Resource.
func (r *K8sControlPlane) Metadata() *resource.Metadata {
	return &r.md
}

// Spec implements resource.Resource.
func (r *K8sControlPlane) Spec() interface{} {
	return r.spec
}

func (r *K8sControlPlane) String() string {
	return fmt.Sprintf("config.K8sControlPlane(%q)", r.md.ID())
}

// DeepCopy implements resource.Resource.
func (r *K8sControlPlane) DeepCopy() resource.Resource {
	return &K8sControlPlane{
		md:   r.md,
		spec: r.spec,
	}
}

// ResourceDefinition implements core.ResourceDefinitionProvider interface.
func (r *K8sControlPlane) ResourceDefinition() core.ResourceDefinitionSpec {
	return core.ResourceDefinitionSpec{
		Type:             K8sControlPlaneType,
		Aliases:          []resource.Type{"controlPlane"},
		DefaultNamespace: NamespaceName,
	}
}

// APIServer returns K8sControlPlaneApiServerSpec.
func (r *K8sControlPlane) APIServer() K8sControlPlaneAPIServerSpec {
	return r.spec.(K8sControlPlaneAPIServerSpec)
}

// SetAPIServer sets K8sControlPlaneApiServerSpec.
func (r *K8sControlPlane) SetAPIServer(spec K8sControlPlaneAPIServerSpec) {
	r.spec = spec
}
