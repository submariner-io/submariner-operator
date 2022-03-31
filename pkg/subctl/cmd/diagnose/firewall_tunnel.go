/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package diagnose

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/diagnose"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"k8s.io/client-go/rest"
)

func init() {
	command := &cobra.Command{
		Use:   "inter-cluster <localkubeconfig> <remotekubeconfig>",
		Short: "Check firewall access to setup tunnels between the Gateway node",
		Long:  "This command checks if the firewall configuration allows tunnels to be configured on the Gateway nodes.",
		Args: func(command *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("two kubeconfigs must be specified")
			}

			same, err := cmd.CompareFiles(args[0], args[1])
			if err != nil {
				return err // nolint:wrapcheck // No need to wrap here
			}

			if same {
				return fmt.Errorf("the specified kubeconfig files are the same")
			}

			return nil
		},
		Run: validateTunnelConfig,
	}

	addDiagnoseFWConfigFlags(command)
	addVerboseFlag(command)
	diagnoseFirewallConfigCmd.AddCommand(command)
}

func validateTunnelConfig(command *cobra.Command, args []string) {
	localProducer := restconfig.NewProducerFrom(args[0], "")
	localCfg, err := localProducer.ForCluster()
	utils.ExitOnError("The provided local kubeconfig is invalid", err)

	remoteProducer := restconfig.NewProducerFrom(args[1], "")
	remoteCfg, err := remoteProducer.ForCluster()
	utils.ExitOnError("The provided remote kubeconfig is invalid", err)

	if !validateTunnelConfigAcrossClusters(localCfg.Config, remoteCfg.Config) {
		os.Exit(1)
	}
}

func validateTunnelConfigAcrossClusters(localCfg, remoteCfg *rest.Config) bool {
	localCluster := newCluster(localCfg)

	localCluster.Name = localCluster.Submariner.Spec.ClusterID

	remoteCluster := newCluster(remoteCfg)

	remoteCluster.Name = remoteCluster.Submariner.Spec.ClusterID

	return diagnose.TunnelConfigAcrossClusters(clusterInfoFrom(localCluster), clusterInfoFrom(remoteCluster), diagnose.FirewallOptions{
		ValidationTimeout: validationTimeout,
		VerboseOutput:     verboseOutput,
		PodNamespace:      podNamespace,
	}, cli.NewStatus())
}

func newCluster(cfg *rest.Config) *cmd.Cluster {
	cluster, errMsg := cmd.NewCluster(cfg, "")
	if cluster == nil {
		utils.ExitWithErrorMsg(errMsg)
	}

	if cluster.Submariner == nil {
		utils.ExitWithErrorMsg(cmd.SubmMissingMessage)
	}

	return cluster
}
