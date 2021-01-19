// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package k8s

import (
	"fmt"

	"github.com/talos-systems/os-runtime/pkg/resource"
	"github.com/talos-systems/os-runtime/pkg/resource/core"
)

// StaticPodStatusType is type of StaticPodStatus resource.
const StaticPodStatusType = resource.Type("k8s/staticPodStatus")

// StaticPodStatus resource holds definition of kubelet static pod.
type StaticPodStatus struct {
	md   resource.Metadata
	spec StaticPodStatusSpec
}

// StaticPodStatusSpec describes kubelet static pod status.
type StaticPodStatusSpec struct {
	Running bool `yaml:"running"`
}

// NewStaticPodStatus initializes a StaticPodStatus resource.
func NewStaticPodStatus(namespace resource.Namespace, id resource.ID) *StaticPodStatus {
	r := &StaticPodStatus{
		md:   resource.NewMetadata(namespace, StaticPodStatusType, id, resource.VersionUndefined),
		spec: StaticPodStatusSpec{},
	}

	r.md.BumpVersion()

	return r
}

// Metadata implements resource.Resource.
func (r *StaticPodStatus) Metadata() *resource.Metadata {
	return &r.md
}

// Spec implements resource.Resource.
func (r *StaticPodStatus) Spec() interface{} {
	return r.spec
}

func (r *StaticPodStatus) String() string {
	return fmt.Sprintf("k8s.StaticPodStatus(%q)", r.md.ID())
}

// DeepCopy implements resource.Resource.
func (r *StaticPodStatus) DeepCopy() resource.Resource {
	return &StaticPodStatus{
		md:   r.md,
		spec: r.spec,
	}
}

// ResourceDefinition implements core.ResourceDefinitionProvider interface.
func (r *StaticPodStatus) ResourceDefinition() core.ResourceDefinitionSpec {
	return core.ResourceDefinitionSpec{
		Type:             StaticPodStatusType,
		Aliases:          []resource.Type{"podStatus"},
		DefaultNamespace: ControlPlaneNamespaceName,
	}
}

// SetStatus sets pod status.
func (r *StaticPodStatus) SetStatus(status StaticPodStatusSpec) {
	r.spec = status
}
