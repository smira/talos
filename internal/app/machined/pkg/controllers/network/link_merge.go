// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package network provides controllers which manage network resources.
package network

import (
	"context"
	"fmt"
	"sort"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/resource"
	"go.uber.org/zap"

	"github.com/talos-systems/talos/pkg/resources/network"
)

// LinkMergeController merges network.LinkSpec in network.ConfigNamespace and produces final network.LinkSpec in network.Namespace.
type LinkMergeController struct{}

// Name implements controller.Controller interface.
func (ctrl *LinkMergeController) Name() string {
	return "network.LinkMergeController"
}

// Inputs implements controller.Controller interface.
func (ctrl *LinkMergeController) Inputs() []controller.Input {
	return []controller.Input{
		{
			Namespace: network.ConfigNamespaceName,
			Type:      network.LinkSpecType,
			Kind:      controller.InputWeak,
		},
		{
			Namespace: network.NamespaceName,
			Type:      network.LinkSpecType,
			Kind:      controller.InputDestroyReady,
		},
	}
}

// Outputs implements controller.Controller interface.
func (ctrl *LinkMergeController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: network.LinkSpecType,
			Kind: controller.OutputShared,
		},
	}
}

// Run implements controller.Controller interface.
//
//nolint:gocyclo
func (ctrl *LinkMergeController) Run(ctx context.Context, r controller.Runtime, logger *zap.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		}

		// list source network configuration resources
		list, err := r.List(ctx, resource.NewMetadata(network.ConfigNamespaceName, network.LinkSpecType, "", resource.VersionUndefined))
		if err != nil {
			return fmt.Errorf("error listing source network routes: %w", err)
		}

		// sort by link name, configuration layer
		sort.Slice(list.Items, func(i, j int) bool {
			left := list.Items[i].(*network.LinkSpec)  //nolint:errcheck,forcetypeassert
			right := list.Items[j].(*network.LinkSpec) //nolint:errcheck,forcetypeassert

			if left.Status().Name < right.Status().Name {
				return false
			}

			if left.Status().Name == right.Status().Name {
				return left.Status().ConfigLayer < right.Status().ConfigLayer
			}

			return true
		})

		// build final link definition merging multiple layers
		links := map[string]*network.LinkSpecSpec{}

		for _, res := range list.Items {
			link := res.(*network.LinkSpec) //nolint:errcheck,forcetypeassert
			id := network.LinkID(link.Status().Name)

			existing, ok := links[id]
			if !ok {
				links[id] = link.Status()
			} else if err = existing.Merge(link.Status()); err != nil {
				logger.Warn("error merging links", zap.Error(err))
			}
		}

		for id, link := range links {
			link := link

			if err = r.Modify(ctx, network.NewLinkSpec(network.NamespaceName, id), func(res resource.Resource) error {
				l := res.(*network.LinkSpec) //nolint:errcheck,forcetypeassert

				*l.Status() = *link

				return nil
			}); err != nil {
				return fmt.Errorf("error updating resource: %w", err)
			}
		}

		// list link for cleanup
		list, err = r.List(ctx, resource.NewMetadata(network.NamespaceName, network.LinkSpecType, "", resource.VersionUndefined))
		if err != nil {
			return fmt.Errorf("error listing resources: %w", err)
		}

		for _, res := range list.Items {
			if _, ok := links[res.Metadata().ID()]; !ok {
				var okToDestroy bool

				okToDestroy, err = r.Teardown(ctx, res.Metadata())
				if err != nil {
					return fmt.Errorf("error cleaning up addresses: %w", err)
				}

				if okToDestroy {
					if err = r.Destroy(ctx, res.Metadata()); err != nil {
						return fmt.Errorf("error cleaning up addresses: %w", err)
					}
				}
			}
		}
	}
}
