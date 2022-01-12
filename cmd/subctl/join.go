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
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/cluster"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/nodes"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/deploy"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/join"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
)

var joinFlags deploy.WithJoinOptions

var joinCmd = &cobra.Command{
	Use:     "join",
	Short:   "Connect a cluster to an existing broker",
	Args:    cobra.MaximumNArgs(1),
	PreRunE: restConfigProducer.CheckVersionMismatch,
	Run: func(cmd *cobra.Command, args []string) {
		status := cli.NewReporter()
		checkArgumentPassed(args)

		brokerInfo, err := broker.ReadInfoFromFile(args[0])
		exit.OnError(status.Error(err, "Error loading the broker information from the given file"))
		status.Success("%s indicates broker is at %s", args[0], brokerInfo.BrokerURL)

		determineClusterID(status)

		clientConfig, err := restConfigProducer.ForCluster()
		exit.OnError(status.Error(err, "Error creating the REST config"))

		clientProducer, err := client.NewProducerFromRestConfig(clientConfig)
		exit.OnError(status.Error(err, "Error creating the client producer"))

		networkDetails := getNetworkDetails(clientProducer, status)
		determinePodCIDR(networkDetails, status)
		determineServiceCIDR(networkDetails, status)

		gatewayNodes, err := nodes.ListGateways(clientProducer.ForKubernetes())
		exit.OnError(status.Error(err, "Error retrieving the gateway nodes"))

		var gatewayNode struct{ Node string }
		// If no Gateway nodes present, get all worker nodes and ask user to select one of them as gateway node
		if len(gatewayNodes) == 0 {
			allWorkerNodeNames, err := nodes.GetAllWorkerNames(clientProducer.ForKubernetes())
			exit.OnError(status.Error(err, "Error listing the worker nodes"))

			gatewayNode, err = askForGatewayNode(allWorkerNodeNames)
			exit.OnError(status.Error(err, "Error getting gateway node"))
		} else {
			fmt.Printf("* There are %d labeled nodes in the cluster:\n", len(gatewayNodes))
			for i := range gatewayNodes {
				fmt.Printf("  - %s\n", gatewayNodes[i])
			}
		}

		err = join.SubmarinerCluster(brokerInfo, &joinFlags, clientProducer, status, gatewayNode)
		exit.OnError(err)
	},
}

func init() {
	addJoinFlags(joinCmd)
	restConfigProducer.AddKubeContextFlag(joinCmd)
	rootCmd.AddCommand(joinCmd)
}

func addJoinFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&joinFlags.ClusterID, "clusterid", "", "cluster ID used to identify the tunnels")
	cmd.Flags().StringVar(&joinFlags.ServiceCIDR, "servicecidr", "", "service CIDR")
	cmd.Flags().StringVar(&joinFlags.ClusterCIDR, "clustercidr", "", "cluster CIDR")
	cmd.Flags().StringVar(&joinFlags.Repository, "repository", "", "image repository")
	cmd.Flags().StringVar(&joinFlags.ImageVersion, "version", "", "image version")
	cmd.Flags().StringVar(&joinFlags.ColorCodes, "colorcodes", submariner.DefaultColorCode, "color codes")
	cmd.Flags().IntVar(&joinFlags.NattPort, "nattport", 4500, "IPsec NATT port")
	cmd.Flags().IntVar(&joinFlags.IkePort, "ikeport", 500, "IPsec IKE port")
	cmd.Flags().BoolVar(&joinFlags.NatTraversal, "natt", true, "enable NAT traversal for IPsec")

	cmd.Flags().BoolVar(&joinFlags.PreferredServer, "preferred-server", false,
		"enable this cluster as a preferred server for dataplane connections")

	cmd.Flags().BoolVar(&joinFlags.LoadBalancerEnabled, "load-balancer", false,
		"enable automatic LoadBalancer in front of the gateways")

	cmd.Flags().BoolVar(&joinFlags.ForceUDPEncaps, "force-udp-encaps", false, "force UDP encapsulation for IPSec")

	cmd.Flags().BoolVar(&joinFlags.IpsecDebug, "ipsec-debug", false, "enable IPsec debugging (verbose logging)")
	cmd.Flags().BoolVar(&joinFlags.SubmarinerDebug, "pod-debug", false,
		"enable Submariner pod debugging (verbose logging in the deployed pods)")
	cmd.Flags().BoolVar(&joinFlags.OperatorDebug, "operator-debug", false, "enable operator debugging (verbose logging)")
	cmd.Flags().BoolVar(&joinFlags.LabelGateway, "label-gateway", true, "label gateways if necessary")
	cmd.Flags().StringVar(&joinFlags.CableDriver, "cable-driver", "", "cable driver implementation")
	cmd.Flags().UintVar(&joinFlags.GlobalnetClusterSize, "globalnet-cluster-size", 0,
		"cluster size for GlobalCIDR allocated to this cluster (amount of global IPs)")
	cmd.Flags().StringVar(&joinFlags.GlobalnetCIDR, "globalnet-cidr", "",
		"GlobalCIDR to be allocated to the cluster")
	cmd.Flags().StringSliceVar(&joinFlags.CustomDomains, "custom-domains", nil,
		"list of domains to use for multicluster service discovery")
	cmd.Flags().StringSliceVar(&joinFlags.ImageOverrideArr, "image-override", nil,
		"override component image")
	cmd.Flags().BoolVar(&joinFlags.HealthCheckEnable, "health-check", true,
		"enable Gateway health check")
	cmd.Flags().Uint64Var(&joinFlags.HealthCheckInterval, "health-check-interval", 1,
		"interval in seconds between health check packets")
	cmd.Flags().Uint64Var(&joinFlags.HealthCheckMaxPacketLossCount, "health-check-max-packet-loss-count", 5,
		"maximum number of packets lost before the connection is marked as down")
	cmd.Flags().BoolVar(&joinFlags.GlobalnetEnabled, "globalnet", true,
		"enable/disable Globalnet for this cluster")
	cmd.Flags().StringVar(&joinFlags.CorednsCustomConfigMap, "coredns-custom-configmap", "",
		"Name of the custom CoreDNS configmap to configure forwarding to lighthouse. It should be in "+
			"<namespace>/<name> format where <namespace> is optional and defaults to kube-system")
	cmd.Flags().BoolVar(&joinFlags.IgnoreRequirements, "ignore-requirements", false, "ignore requirement failures (unsupported)")
}

func askForGatewayNode(allWorkerNodeNames []string) (struct{ Node string }, error) {
	qs := []*survey.Question{
		{
			Name: "node",
			Prompt: &survey.Select{
				Message: "Which node should be used as the gateway?",
				Options: allWorkerNodeNames,
			},
		},
	}

	answers := struct {
		Node string
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		return struct{ Node string }{}, err // nolint:wrapcheck // No need to wrap here
	}

	return answers, nil
}

func checkArgumentPassed(args []string) {
	if len(args) == 0 {
		exit.WithMessage("The broker-info.subm file argument generated by 'subctl deploy-broker' is missing")
	}
}

func askForClusterID() (string, error) {
	// Missing information
	qs := []*survey.Question{}

	qs = append(qs, &survey.Question{
		Name:   "clusterID",
		Prompt: &survey.Input{Message: "What is your cluster ID?"},
		Validate: func(val interface{}) error {
			str, ok := val.(string)
			if !ok {
				return nil
			}

			return cluster.IsValidID(str) // nolint:wrapcheck // No need to wrap
		},
	})

	answers := struct {
		ClusterID string
	}{}

	err := survey.Ask(qs, &answers)
	// Most likely a programming error
	if err != nil {
		return "", err // nolint:wrapcheck // No need to wrap
	}

	return answers.ClusterID, nil
}

func askForCIDR(name string) (string, error) {
	qs := []*survey.Question{{
		Name:     "cidr",
		Prompt:   &survey.Input{Message: fmt.Sprintf("What's the %s CIDR for your cluster?", name)},
		Validate: survey.Required,
	}}

	answers := struct {
		Cidr string
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		return "", err // nolint:wrapcheck // No need to wrap here
	}

	return strings.TrimSpace(answers.Cidr), nil
}

func determineClusterID(status reporter.Interface) {
	var err error

	if joinFlags.ClusterID == "" {
		joinFlags.ClusterID, err = restConfigProducer.GetClusterID()
		exit.OnError(status.Error(err, "Error determining cluster ID of the target cluster"))
	}

	if joinFlags.ClusterID != "" {
		err = cluster.IsValidID(joinFlags.ClusterID)
		if err != nil {
			_ = status.Error(err, "Invalid cluster ID")
			joinFlags.ClusterID = ""
		}
	}

	if joinFlags.ClusterID == "" {
		joinFlags.ClusterID, err = askForClusterID()
		exit.OnError(status.Error(err, "Error collecting cluster ID"))
	}
}

func getNetworkDetails(clientProducer client.Producer, status reporter.Interface) *network.ClusterNetwork {
	status.Start("Discovering network details")

	networkDetails, err := network.Discover(clientProducer.ForDynamic(), clientProducer.ForKubernetes(), clientProducer.ForOperator(),
		constants.OperatorNamespace)
	if err != nil {
		status.Warning("Unable to discover network details: %s", err)
	} else if networkDetails == nil {
		status.Warning("No network details discovered")
	}

	status.End()

	if networkDetails != nil {
		networkDetails.Show()
	}

	return networkDetails
}

func determinePodCIDR(nd *network.ClusterNetwork, status reporter.Interface) {
	if joinFlags.ClusterCIDR != "" {
		if nd != nil && len(nd.PodCIDRs) > 0 && nd.PodCIDRs[0] != joinFlags.ClusterCIDR {
			status.Warning("The provided pod CIDR for the cluster (%s) does not match the discovered CIDR (%s)",
				joinFlags.ClusterCIDR, nd.PodCIDRs[0])
		}
	} else if nd == nil || len(nd.PodCIDRs) == 0 {
		var err error
		joinFlags.ClusterCIDR, err = askForCIDR("Pod")
		exit.OnError(status.Error(err, "Error collecting CIDR"))
	}
}

func determineServiceCIDR(nd *network.ClusterNetwork, status reporter.Interface) {
	if joinFlags.ServiceCIDR != "" {
		if nd != nil && len(nd.ServiceCIDRs) > 0 && nd.ServiceCIDRs[0] != joinFlags.ServiceCIDR {
			status.Warning("The provided service CIDR for the cluster (%s) does not match the discovered CIDR (%s)",
				joinFlags.ServiceCIDR, nd.ServiceCIDRs[0])
		}
	} else if nd == nil || len(nd.ServiceCIDRs) == 0 {
		var err error
		joinFlags.ServiceCIDR, err = askForCIDR("Service")
		exit.OnError(status.Error(err, "Error collecting CIDR"))
	}
}
