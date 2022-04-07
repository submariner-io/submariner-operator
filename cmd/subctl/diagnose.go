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
	"os"

	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/cmd/subctl/execute"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/diagnose"
)

var (
	diagnoseFirewallOptions diagnose.FirewallOptions

	diagnoseKubeProxyOptions struct {
		podNamespace string
	}

	diagnoseCmd = &cobra.Command{
		Use:   "diagnose",
		Short: "Run diagnostic checks on the Submariner deployment and report any issues",
		Long:  "This command runs various diagnostic checks on the Submariner deployment and reports any issues",
	}

	diagnoseCNICmd = &cobra.Command{
		Use:   "cni",
		Short: "Check the CNI network plugin",
		Long:  "This command checks if the detected CNI network plugin is supported by Submariner.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, execute.IfSubmarinerInstalled(diagnose.CNIConfig))
		},
	}

	diagnoseConnectionsCmd = &cobra.Command{
		Use:   "connections",
		Short: "Check the Gateway connections",
		Long:  "This command checks that the Gateway connections to other clusters are all established",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, execute.IfSubmarinerInstalled(diagnose.Connections))
		},
	}

	diagnoseDeploymentCmd = &cobra.Command{
		Use:   "deployment",
		Short: "Check the Submariner deployment",
		Long:  "This command checks that the Submariner components are properly deployed and running with no overlapping CIDRs.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, execute.IfSubmarinerInstalled(diagnose.Deployments))
		},
	}

	diagnoseVersionCmd = &cobra.Command{
		Use:   "k8s-version",
		Short: "Check the Kubernetes version",
		Long:  "This command checks if Submariner can be deployed on the Kubernetes version.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, func(info *cluster.Info, status reporter.Interface) bool {
				return diagnose.K8sVersion(info.ClientProducer.ForKubernetes(), status)
			})
		},
	}

	diagnoseKubeProxyModeCmd = &cobra.Command{
		Use:   "kube-proxy-mode",
		Short: "Check the kube-proxy mode",
		Long:  "This command checks if the kube-proxy mode is supported by Submariner.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, func(info *cluster.Info, status reporter.Interface) bool {
				return diagnose.KubeProxyMode(info.ClientProducer.ForKubernetes(), diagnoseKubeProxyOptions.podNamespace, status)
			})
		},
	}

	diagnoseFirewallCmd = &cobra.Command{
		Use:   "firewall",
		Short: "Check the firewall configuration",
		Long:  "This command checks if the firewall is configured as per Submariner pre-requisites.",
	}

	diagnoseFirewallMetricsCmd = &cobra.Command{
		Use:   "metrics",
		Short: "Check firewall access to metrics",
		Long:  "This command checks if the firewall configuration allows metrics to be accessed from the Gateway nodes.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, execute.IfSubmarinerInstalled(
				func(info *cluster.Info, status reporter.Interface) bool {
					return diagnose.FirewallMetricsConfig(info, diagnoseFirewallOptions, status)
				}))
		},
	}

	diagnoseFirewallVxLANCmd = &cobra.Command{
		Use:   "intra-cluster",
		Short: "Check firewall access for intra-cluster Submariner VxLAN traffic",
		Long:  "This command checks if the firewall configuration allows traffic over vx-submariner interface.",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, execute.IfSubmarinerInstalled(
				func(info *cluster.Info, status reporter.Interface) bool {
					return diagnose.VxLANConfig(info, diagnoseFirewallOptions, status)
				}))
		},
	}

	diagnoseFirewallTunnelCmd = &cobra.Command{
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
		Run: func(command *cobra.Command, args []string) {
			if !diagnose.TunnelConfigAcrossClusters(clusterInfoFromKubeConfig(args[0]), clusterInfoFromKubeConfig(args[1]),
				diagnoseFirewallOptions, cli.NewReporter()) {
				os.Exit(1)
			}
		},
	}

	diagnoseAllCmd = &cobra.Command{
		Use:   "all",
		Short: "Run all diagnostic checks (except those requiring two kubecontexts)",
		Long:  "This command runs all diagnostic checks (except those requiring two kubecontexts) and reports any issues",
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, diagnoseAll)
		},
	}
)

func init() {
	restConfigProducer.AddKubeConfigFlag(diagnoseCmd)
	restConfigProducer.AddInClusterConfigFlag(diagnoseCmd)
	rootCmd.AddCommand(diagnoseCmd)

	addDiagnoseSubCommands()
	addDiagnoseFirewallSubCommands()
}

func addDiagnoseSubCommands() {
	addDiagnosePodNamespaceFlag(diagnoseKubeProxyModeCmd, &diagnoseKubeProxyOptions.podNamespace)

	diagnoseCmd.AddCommand(diagnoseCNICmd)
	diagnoseCmd.AddCommand(diagnoseConnectionsCmd)
	diagnoseCmd.AddCommand(diagnoseDeploymentCmd)
	diagnoseCmd.AddCommand(diagnoseVersionCmd)
	diagnoseCmd.AddCommand(diagnoseKubeProxyModeCmd)
	diagnoseCmd.AddCommand(diagnoseAllCmd)
	diagnoseCmd.AddCommand(diagnoseFirewallCmd)
}

func addDiagnoseFirewallSubCommands() {
	addDiagnoseFWConfigFlags(diagnoseFirewallMetricsCmd)
	addDiagnoseFWConfigFlags(diagnoseFirewallVxLANCmd)
	addDiagnoseFWConfigFlags(diagnoseFirewallTunnelCmd)

	diagnoseFirewallCmd.AddCommand(diagnoseFirewallMetricsCmd)
	diagnoseFirewallCmd.AddCommand(diagnoseFirewallVxLANCmd)
	diagnoseFirewallCmd.AddCommand(diagnoseFirewallTunnelCmd)
}

func addDiagnosePodNamespaceFlag(command *cobra.Command, value *string) {
	command.Flags().StringVar(value, "namespace", "default", "namespace in which validation pods should be deployed")
}

func addDiagnoseFWConfigFlags(command *cobra.Command) {
	command.Flags().UintVar(&diagnoseFirewallOptions.ValidationTimeout, "validation-timeout", 90,
		"timeout in seconds while validating the connection attempt")
	command.Flags().BoolVar(&diagnoseFirewallOptions.VerboseOutput, "verbose", false, "produce verbose output")
	addDiagnosePodNamespaceFlag(command, &diagnoseFirewallOptions.PodNamespace)
}

func clusterInfoFromKubeConfig(kubeConfig string) *cluster.Info {
	producer := restconfig.NewProducerFrom(kubeConfig, "")
	config, err := producer.ForCluster()
	exit.OnErrorWithMessage(err, fmt.Sprintf("The provided kubeconfig %q is invalid", kubeConfig))

	clientProducer, err := client.NewProducerFromRestConfig(config.Config)
	exit.OnErrorWithMessage(err, fmt.Sprintf("Error creating client producer for kubeconfig %q", kubeConfig))

	clusterInfo, err := cluster.NewInfo("", clientProducer, nil)
	exit.OnErrorWithMessage(err, fmt.Sprintf("Error initializing cluster information for kubeconfig %q", kubeConfig))

	if clusterInfo.Submariner == nil {
		exit.WithMessage(constants.SubmarinerNotInstalled)
	}

	clusterInfo.Name = clusterInfo.Submariner.Spec.ClusterID

	return clusterInfo
}

func diagnoseAll(clusterInfo *cluster.Info, status reporter.Interface) bool {
	success := diagnose.K8sVersion(clusterInfo.ClientProducer.ForKubernetes(), status)

	fmt.Println()

	if clusterInfo.Submariner == nil {
		status.Warning(constants.SubmarinerNotInstalled)

		return success
	}

	success = diagnose.CNIConfig(clusterInfo, status) && success

	fmt.Println()

	success = diagnose.Connections(clusterInfo, status) && success

	fmt.Println()

	success = diagnose.Deployments(clusterInfo, status) && success

	fmt.Println()

	success = diagnose.KubeProxyMode(clusterInfo.ClientProducer.ForKubernetes(), diagnoseKubeProxyOptions.podNamespace, status) && success

	fmt.Println()

	success = diagnose.FirewallMetricsConfig(clusterInfo, diagnoseFirewallOptions, status) && success

	fmt.Println()

	success = diagnose.VxLANConfig(clusterInfo, diagnoseFirewallOptions, status) && success

	fmt.Println()

	success = diagnose.GlobalnetConfig(clusterInfo, status) && success

	fmt.Println()

	fmt.Printf("Skipping inter-cluster firewall check as it requires two kubeconfigs." +
		" Please run \"subctl diagnose firewall inter-cluster\" command manually.\n")

	return success
}
