// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package output provides writers in different formats.
package output

import (
	"fmt"

	"github.com/talos-systems/os-runtime/pkg/resource"
)

// Writer interface.
type Writer interface {
	WriteHeader(definition resource.Resource) error
	WriteResource(node string, r resource.Resource) error
	Flush() error
}

// NewWriter builds writer from type.
func NewWriter(format string) (Writer, error) {
	switch format {
	case "table":
		return NewTable(), nil
	case "yaml":
		return NewYAML(), nil
	default:
		return nil, fmt.Errorf("output format %q is not supported", format)
	}
}
