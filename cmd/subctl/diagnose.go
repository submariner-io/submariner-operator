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

package subctl

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/cmd/subctl/execute"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/diagnose"
)

var (
	diagnoseCmd = &cobra.Command{
		Use:   "diagnose",
		Short: "Run diagnostic checks on the Submariner deployment and report any issues",
		Long:  "This command runs various diagnostic checks on the Submariner deployment and reports any issues",
	}
	cniCmd = &cobra.Command{
		Use:   "cni",
		Short: "Check the CNI network plugin",
		Long:  "This command checks if the detected CNI network plugin is supported by Submariner.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, diagnose.CNIConfig)
		},
	}
	connectionsCmd = &cobra.Command{
		Use:   "connections",
		Short: "Check the Gateway connections",
		Long:  "This command checks that the Gateway connections to other clusters are all established",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, diagnose.Connections)
		},
	}
	deploymentCmd = &cobra.Command{
		Use:   "deployment",
		Short: "Check the Submariner deployment",
		Long:  "This command checks that the Submariner components are properly deployed and running with no overlapping CIDRs.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, diagnose.Deployments)
		},
	}
	versionCmd = &cobra.Command{
		Use:   "k8s-version",
		Short: "Check the Kubernetes version",
		Long:  "This command checks if Submariner can be deployed on the Kubernetes version.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, diagnose.K8sVersion)
		},
	}
	kpModeCmd = &cobra.Command{
		Use:   "kube-proxy-mode",
		Short: "Check the kube-proxy mode",
		Long:  "This command checks if the kube-proxy mode is supported by Submariner.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, diagnose.KubeProxyMode)
		},
	}
	allCmd = &cobra.Command{
		Use:   "all",
		Short: "Run all diagnostic checks (except those requiring two kubecontexts)",
		Long:  "This command runs all diagnostic checks (except those requiring two kubecontexts) and reports any issues",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, diagnose.All)
		},
	}
	diagnoseFirewallConfigCmd = &cobra.Command{
		Use:   "firewall",
		Short: "Check the firewall configuration",
		Long:  "This command checks if the firewall is configured as per Submariner pre-requisites.",
	}
	firewallMetricsCmd = &cobra.Command{
		Use:   "metrics",
		Short: "Check firewall access to metrics",
		Long:  "This command checks if the firewall configuration allows metrics to be accessed from the Gateway nodes.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, diagnose.FirewallMetricsConfig)
		},
	}
	firewallVxLANCmd = &cobra.Command{
		Use:   "intra-cluster",
		Short: "Check firewall access for intra-cluster Submariner VxLAN traffic",
		Long:  "This command checks if the firewall configuration allows traffic over vx-submariner interface.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, diagnose.VxLANConfig)
		},
	}
	firewallTunnelCmd = &cobra.Command{
		Use:   "inter-cluster <localkubeconfig> <remotekubeconfig>",
		Short: "Check firewall access to setup tunnels between the Gateway node",
		Long:  "This command checks if the firewall configuration allows tunnels to be configured on the Gateway nodes.",
		Args: func(command *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("two kubeconfigs must be specified")
			}

			same, err := compareFiles(args[0], args[1])
			if err != nil {
				return err
			}

			if same {
				return fmt.Errorf("the specified kubeconfig files are the same")
			}

			return nil
		},
		Run: validateTunnelConfig,
	}
)

func init() {
	restConfigProducer.AddKubeConfigFlag(diagnoseCmd)
	restConfigProducer.AddInClusterConfigFlag(diagnoseCmd)
	rootCmd.AddCommand(diagnoseCmd)
	addDiagnoseSubCmd()
	addFirewallSubSubCmd()
}

func addDiagnoseSubCmd() {
	addNamespaceFlag(kpModeCmd)
	diagnoseCmd.AddCommand(cniCmd)
	diagnoseCmd.AddCommand(connectionsCmd)
	diagnoseCmd.AddCommand(deploymentCmd)
	diagnoseCmd.AddCommand(versionCmd)
	diagnoseCmd.AddCommand(kpModeCmd)
	diagnoseCmd.AddCommand(allCmd)
	diagnoseCmd.AddCommand(diagnoseFirewallConfigCmd)
}

func addFirewallSubSubCmd() {
	addDiagnoseFWConfigFlags(firewallMetricsCmd)
	addDiagnoseFWConfigFlags(firewallVxLANCmd)
	addDiagnoseFWConfigFlags(firewallTunnelCmd)
	addVerboseFlag(firewallMetricsCmd)
	addVerboseFlag(firewallVxLANCmd)
	addVerboseFlag(firewallTunnelCmd)
	diagnoseFirewallConfigCmd.AddCommand(firewallMetricsCmd)
	diagnoseFirewallConfigCmd.AddCommand(firewallVxLANCmd)
	diagnoseFirewallConfigCmd.AddCommand(firewallTunnelCmd)
}

func addVerboseFlag(command *cobra.Command) {
	command.Flags().BoolVar(&diagnose.VerboseOutput, "verbose", false, "produce verbose output")
}

func addNamespaceFlag(command *cobra.Command) {
	command.Flags().StringVar(&diagnose.KubeProxyPodNamespace, "namespace", "default",
		"namespace in which validation pods should be deployed")
}

func addDiagnoseFWConfigFlags(command *cobra.Command) {
	command.Flags().UintVar(&diagnose.ValidationTimeout, "validation-timeout", 90,
		"timeout in seconds while validating the connection attempt")
	addNamespaceFlag(command)
}

func validateTunnelConfig(command *cobra.Command, args []string) {
	localProducer := restconfig.NewProducerFrom(args[0], "")
	localCfg, err := localProducer.ForCluster()
	exit.OnErrorWithMessage(err, "The provided local kubeconfig is invalid")

	localClientProducer, err := client.NewProducerFromRestConfig(localCfg)
	exit.OnErrorWithMessage(err, "Error creating local client producer")

	remoteProducer := restconfig.NewProducerFrom(args[1], "")
	remoteCfg, err := remoteProducer.ForCluster()
	exit.OnErrorWithMessage(err, "The provided remote kubeconfig is invalid")

	remoteClientProducer, err := client.NewProducerFromRestConfig(remoteCfg)
	exit.OnErrorWithMessage(err, "Error creating remote client producer")

	status := cli.NewReporter()
	if !diagnose.ValidateTunnelConfigAcrossClusters(localClientProducer, remoteClientProducer, status) {
		exit.WithMessage("Error validating tunnel creation across the specified clusters")
	}
}
