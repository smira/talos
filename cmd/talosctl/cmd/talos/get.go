// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package talos

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/talos-systems/talos/cmd/talosctl/cmd/talos/output"
	"github.com/talos-systems/talos/pkg/machinery/client"
)

var getCmdFlags struct {
	namespace string

	output string
}

// getCmd represents the get (resources) command.
var getCmd = &cobra.Command{
	Use:     "get <type> [<id>]",
	Aliases: []string{"g"},
	Short:   "Get a specific resource or list of resources.",
	Long:    ``,
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return WithClient(func(ctx context.Context, c *client.Client) error {
			out, err := output.NewWriter(getCmdFlags.output)
			if err != nil {
				return err
			}

			resourceType := args[0]

			var resourceID string

			if len(args) == 2 {
				resourceID = args[1]
			}

			defer out.Flush() //nolint: errcheck

			var headerWritten bool

			if resourceID != "" {
				resp, err := c.Resources.Get(ctx, getCmdFlags.namespace, resourceType, resourceID)

				for _, msg := range resp {
					if msg.Definition != nil && !headerWritten {
						if e := out.WriteHeader(msg.Definition); e != nil {
							return e
						}

						headerWritten = true
					}

					if msg.Resource != nil {
						if e := out.WriteResource(msg.Metadata.GetHostname(), msg.Resource); e != nil {
							return e
						}
					}

				}

				if err != nil {
					return err
				}
			} else {
				listClient, err := c.Resources.List(ctx, getCmdFlags.namespace, resourceType)
				if err != nil {
					return err
				}

				for {
					msg, err := listClient.Recv()
					if err != nil {
						if err == io.EOF || status.Code(err) == codes.Canceled {
							return nil
						}

						return err
					}

					if msg.Metadata.GetError() != "" {
						fmt.Fprintf(os.Stderr, "%s: %s\n", msg.Metadata.GetHostname(), msg.Metadata.GetError())

						continue
					}

					if msg.Definition != nil && !headerWritten {
						if e := out.WriteHeader(msg.Definition); e != nil {
							return e
						}

						headerWritten = true
					}

					if msg.Resource != nil {
						if err := out.WriteResource(msg.Metadata.GetHostname(), msg.Resource); err != nil {
							return err
						}
					}
				}
			}

			return nil
		})
	},
}

func init() {
	getCmd.Flags().StringVar(&getCmdFlags.namespace, "namespace", "", "resource namespace (default is to use default namespace per resource)")
	getCmd.Flags().StringVarP(&getCmdFlags.output, "output", "o", "table", "output mode (table, yaml)")
	addCommand(getCmd)
}
