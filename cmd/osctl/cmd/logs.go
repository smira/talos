/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package cmd

import (
	"io"
	"os"

	criconstants "github.com/containerd/cri/pkg/constants"
	"github.com/spf13/cobra"
	"github.com/talos-systems/talos/cmd/osctl/pkg/client"
	"github.com/talos-systems/talos/cmd/osctl/pkg/helpers"
	"github.com/talos-systems/talos/internal/pkg/constants"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "Retrieve logs for a process or container",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			helpers.Should(cmd.Usage())
			os.Exit(1)
		}

		setupClient(func(c *client.Client) {
			var namespace string
			if kubernetes {
				namespace = criconstants.K8sContainerdNamespace
			} else {
				namespace = constants.SystemContainerdNamespace
			}

			stream, err := c.Logs(globalCtx, namespace, args[0])
			if err != nil {
				helpers.Fatalf("error fetching logs: %s", err)
			}

			for {
				data, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						return
					}
					helpers.Fatalf("error streaming logs: %s", err)
				}

				_, err = os.Stdout.Write(data.Bytes)
				helpers.Should(err)
			}
		})
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&kubernetes, "kubernetes", "k", false, "use the k8s.io containerd namespace")
	logsCmd.Flags().StringVarP(&target, "target", "t", "", "target the specificed node")
	rootCmd.AddCommand(logsCmd)
}
