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
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/join"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
)

var joinOptions join.Options

var joinCmd = &cobra.Command{
	Use:     "join",
	Short:   "Connect a cluster to an existing broker",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := checkArgumentPassed(args)
		exit.OnError("Argument missing", err)
		subctlData, err := datafile.NewFromFile(args[0])
		exit.OnError("Error loading the broker information from the given file", err)
		fmt.Printf("* %s says broker is at: %s\n", args[0], subctlData.BrokerURL)
		exit.OnError("Error connecting to broker cluster", err)
		err = join.SubmarinerCluster(joinOptions, kubeContext, kubeConfig, subctlData)
		exit.OnError("Error joining cluster", err)
	},
}

func init() {
	addJoinFlags(joinCmd)
	addKubeContextFlag(joinCmd)
	rootCmd.AddCommand(joinCmd)
}

func addJoinFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&joinOptions.ClusterID, "clusterid", "", "cluster ID used to identify the tunnels")
	cmd.Flags().StringVar(&joinOptions.ServiceCIDR, "servicecidr", "", "service CIDR")
	cmd.Flags().StringVar(&joinOptions.ClusterCIDR, "clustercidr", "", "cluster CIDR")
	cmd.Flags().StringVar(&joinOptions.Repository, "repository", "", "image repository")
	cmd.Flags().StringVar(&joinOptions.ImageVersion, "version", "", "image version")
	cmd.Flags().StringVar(&joinOptions.ColorCodes, "colorcodes", submariner.DefaultColorCode, "color codes")
	cmd.Flags().IntVar(&joinOptions.NattPort, "nattport", 4500, "IPsec NATT port")
	cmd.Flags().IntVar(&joinOptions.IkePort, "ikeport", 500, "IPsec IKE port")
	cmd.Flags().BoolVar(&joinOptions.NatTraversal, "natt", true, "enable NAT traversal for IPsec")

	cmd.Flags().BoolVar(&joinOptions.PreferredServer, "preferred-server", false,
		"enable this cluster as a preferred server for dataplane connections")

	cmd.Flags().BoolVar(&joinOptions.LoadBalancerEnabled, "load-balancer", false,
		"enable automatic LoadBalancer in front of the gateways")

	cmd.Flags().BoolVar(&joinOptions.ForceUDPEncaps, "force-udp-encaps", false, "force UDP encapsulation for IPSec")

	cmd.Flags().BoolVar(&joinOptions.IpsecDebug, "ipsec-debug", false, "enable IPsec debugging (verbose logging)")
	cmd.Flags().BoolVar(&joinOptions.SubmarinerDebug, "pod-debug", false,
		"enable Submariner pod debugging (verbose logging in the deployed pods)")
	cmd.Flags().BoolVar(&joinOptions.OperatorDebug, "operator-debug", false, "enable operator debugging (verbose logging)")
	cmd.Flags().BoolVar(&joinOptions.LabelGateway, "label-gateway", true, "label gateways if necessary")
	cmd.Flags().StringVar(&joinOptions.CableDriver, "cable-driver", "", "cable driver implementation")
	cmd.Flags().UintVar(&joinOptions.GlobalnetClusterSize, "globalnet-cluster-size", 0,
		"cluster size for GlobalCIDR allocated to this cluster (amount of global IPs)")
	cmd.Flags().StringVar(&joinOptions.GlobalnetCIDR, "globalnet-cidr", "",
		"GlobalCIDR to be allocated to the cluster")
	cmd.Flags().StringSliceVar(&joinOptions.CustomDomains, "custom-domains", nil,
		"list of domains to use for multicluster service discovery")
	cmd.Flags().StringSliceVar(&joinOptions.ImageOverrideArr, "image-override", nil,
		"override component image")
	cmd.Flags().BoolVar(&joinOptions.HealthCheckEnable, "health-check", true,
		"enable Gateway health check")
	cmd.Flags().Uint64Var(&joinOptions.HealthCheckInterval, "health-check-interval", 1,
		"interval in seconds between health check packets")
	cmd.Flags().Uint64Var(&joinOptions.HealthCheckMaxPacketLossCount, "health-check-max-packet-loss-count", 5,
		"maximum number of packets lost before the connection is marked as down")
	cmd.Flags().BoolVar(&joinOptions.GlobalnetEnabled, "globalnet", true,
		"enable/disable Globalnet for this cluster")
	cmd.Flags().StringVar(&joinOptions.CorednsCustomConfigMap, "coredns-custom-configmap", "",
		"Name of the custom CoreDNS configmap to configure forwarding to lighthouse. It should be in "+
			"<namespace>/<name> format where <namespace> is optional and defaults to kube-system")
	cmd.Flags().BoolVar(&joinOptions.IgnoreRequirements, "ignore-requirements", false, "ignore requirement failures (unsupported)")
}

func checkArgumentPassed(args []string) error {
	if len(args) == 0 {
		return errors.New("broker-info.subm file generated by 'subctl deploy-broker' not passed")
	}
	return nil
}