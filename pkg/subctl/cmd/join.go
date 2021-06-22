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

package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	cmdVersion "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/version"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"

	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"k8s.io/client-go/rest"

	submariner "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/servicediscoverycr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
	"github.com/submariner-io/submariner-operator/pkg/versions"
)

var (
	clusterID                     string
	serviceCIDR                   string
	clusterCIDR                   string
	globalnetCIDR                 string
	repository                    string
	imageVersion                  string
	nattPort                      int
	ikePort                       int
	preferredServer               bool
	forceUDPEncaps                bool
	colorCodes                    string
	natTraversal                  bool
	globalnetEnabled              bool
	ipsecDebug                    bool
	submarinerDebug               bool
	operatorDebug                 bool
	labelGateway                  bool
	loadBalancerEnabled           bool
	cableDriver                   string
	clienttoken                   *v1.Secret
	globalnetClusterSize          uint
	customDomains                 []string
	imageOverrideArr              []string
	healthCheckEnable             bool
	healthCheckInterval           uint64
	healthCheckMaxPacketLossCount uint64
	corednsCustomConfigMap        string
)

func init() {
	addJoinFlags(joinCmd)
	AddKubeContextFlag(joinCmd)
	rootCmd.AddCommand(joinCmd)
}

func addJoinFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&clusterID, "clusterid", "", "cluster ID used to identify the tunnels")
	cmd.Flags().StringVar(&serviceCIDR, "servicecidr", "", "service CIDR")
	cmd.Flags().StringVar(&clusterCIDR, "clustercidr", "", "cluster CIDR")
	cmd.Flags().StringVar(&repository, "repository", "", "image repository")
	cmd.Flags().StringVar(&imageVersion, "version", "", "image version")
	cmd.Flags().StringVar(&colorCodes, "colorcodes", submariner.DefaultColorCode, "color codes")
	cmd.Flags().IntVar(&nattPort, "nattport", 4500, "IPsec NATT port")
	cmd.Flags().IntVar(&ikePort, "ikeport", 500, "IPsec IKE port")
	cmd.Flags().BoolVar(&natTraversal, "natt", true, "enable NAT traversal for IPsec")

	cmd.Flags().BoolVar(&preferredServer, "preferred-server", false,
		"enable this cluster as a preferred server for dataplane connections")

	cmd.Flags().BoolVar(&loadBalancerEnabled, "load-balancer", false,
		"enable automatic LoadBalancer in front of the gateways")

	cmd.Flags().BoolVar(&forceUDPEncaps, "force-udp-encaps", false, "force UDP encapsulation for IPSec")

	cmd.Flags().BoolVar(&ipsecDebug, "ipsec-debug", false, "enable IPsec debugging (verbose logging)")
	cmd.Flags().BoolVar(&submarinerDebug, "pod-debug", false,
		"enable Submariner pod debugging (verbose logging in the deployed pods)")
	cmd.Flags().BoolVar(&operatorDebug, "operator-debug", false, "enable operator debugging (verbose logging)")
	cmd.Flags().BoolVar(&labelGateway, "label-gateway", true, "label gateways if necessary")
	cmd.Flags().StringVar(&cableDriver, "cable-driver", "", "cable driver implementation")
	cmd.Flags().UintVar(&globalnetClusterSize, "globalnet-cluster-size", 0,
		"cluster size for GlobalCIDR allocated to this cluster (amount of global IPs)")
	cmd.Flags().StringVar(&globalnetCIDR, "globalnet-cidr", "",
		"GlobalCIDR to be allocated to the cluster")
	cmd.Flags().StringSliceVar(&customDomains, "custom-domains", nil,
		"list of domains to use for multicluster service discovery")
	cmd.Flags().StringSliceVar(&imageOverrideArr, "image-override", nil,
		"override component image")
	cmd.Flags().BoolVar(&healthCheckEnable, "health-check", true,
		"enable Gateway health check")
	cmd.Flags().Uint64Var(&healthCheckInterval, "health-check-interval", 1,
		"interval in seconds between health check packets")
	cmd.Flags().Uint64Var(&healthCheckMaxPacketLossCount, "health-check-max-packet-loss-count", 5,
		"maximum number of packets lost before the connection is marked as down")
	cmd.Flags().BoolVar(&globalnetEnabled, "globalnet", true,
		"enable/disable Globalnet for this cluster")
	cmd.Flags().StringVar(&corednsCustomConfigMap, "coredns-custom-configmap", "",
		"Name of the custom CoreDNS configmap to configure forwarding to lighthouse. It should be in "+
			"<namespace>/<name> format where <namespace> is optional and defaults to kube-system")
}

const (
	SubmarinerNamespace = "submariner-operator" // We currently expect everything in submariner-operator
)

var joinCmd = &cobra.Command{
	Use:     "join",
	Short:   "Connect a cluster to an existing broker",
	Args:    cobra.MaximumNArgs(1),
	PreRunE: checkVersionMismatch,
	Run: func(cmd *cobra.Command, args []string) {
		err := checkArgumentPassed(args)
		utils.ExitOnError("Argument missing", err)
		subctlData, err := datafile.NewFromFile(args[0])
		utils.ExitOnError("Argument missing", err)
		utils.ExitOnError("Error loading the broker information from the given file", err)
		fmt.Printf("* %s says broker is at: %s\n", args[0], subctlData.BrokerURL)
		utils.ExitOnError("Error connecting to broker cluster", err)
		err = isValidCustomCoreDNSConfig()
		utils.ExitOnError("Invalid Custom CoreDNS configuration", err)
		config := restconfig.ClientConfig(kubeConfig, kubeContext)
		joinSubmarinerCluster(config, kubeContext, subctlData)
	},
}

func checkArgumentPassed(args []string) error {
	if len(args) == 0 {
		return errors.New("broker-info.subm file generated by 'subctl deploy-broker' not passed")
	}
	return nil
}

var status = cli.NewStatus()

func joinSubmarinerCluster(config clientcmd.ClientConfig, contextName string, subctlData *datafile.SubctlData) {
	// Missing information
	var qs = []*survey.Question{}

	if clusterID == "" {
		rawConfig, err := config.RawConfig()
		// This will be fatal later, no point in continuing
		utils.ExitOnError("Error connecting to the target cluster", err)
		clusterName := restconfig.ClusterNameFromContext(rawConfig, contextName)
		if clusterName != nil {
			clusterID = *clusterName
		}
	}

	if valid, err := isValidClusterID(clusterID); !valid {
		fmt.Printf("Error: %s\n", err.Error())
		qs = append(qs, &survey.Question{
			Name:   "clusterID",
			Prompt: &survey.Input{Message: "What is your cluster ID?"},
			Validate: func(val interface{}) error {
				str, ok := val.(string)
				if !ok {
					return nil
				}
				_, err := isValidClusterID(str)
				return err
			},
		})
	}
	if colorCodes == "" {
		qs = append(qs, &survey.Question{
			Name:     "colorCodes",
			Prompt:   &survey.Input{Message: "What color codes should be used (e.g. \"blue\")?"},
			Validate: survey.Required,
		})
	}

	if len(qs) > 0 {
		answers := struct {
			ClusterID  string
			ColorCodes string
		}{}

		err := survey.Ask(qs, &answers)
		// Most likely a programming error
		utils.PanicOnError(err)

		if len(answers.ClusterID) > 0 {
			clusterID = answers.ClusterID
		}
		if len(answers.ColorCodes) > 0 {
			colorCodes = answers.ColorCodes
		}
	}

	clientConfig, err := config.ClientConfig()
	utils.ExitOnError("Error connecting to the target cluster", err)

	_, failedRequirements, err := cmdVersion.CheckRequirements(clientConfig)
	// We display failed requirements even if an error occurred
	if len(failedRequirements) > 0 {
		fmt.Println("The target cluster fails to meet Submariner's requirements:")
		for i := range failedRequirements {
			fmt.Printf("* %s\n", (failedRequirements)[i])
		}
		utils.ExitOnError("Unable to check all requirements", err)
		os.Exit(1)
	}
	utils.ExitOnError("Unable to check requirements", err)

	if subctlData.IsConnectivityEnabled() && labelGateway {
		err := handleNodeLabels(clientConfig)
		utils.ExitOnError("Unable to set the gateway node up", err)
	}

	status.Start("Discovering network details")
	networkDetails := getNetworkDetails(clientConfig)
	status.End(cli.Success)

	serviceCIDR, serviceCIDRautoDetected, err := getServiceCIDR(serviceCIDR, networkDetails)
	utils.ExitOnError("Error determining the service CIDR", err)

	clusterCIDR, clusterCIDRautoDetected, err := getPodCIDR(clusterCIDR, networkDetails)
	utils.ExitOnError("Error determining the pod CIDR", err)

	brokerAdminConfig, err := subctlData.GetBrokerAdministratorConfig()
	utils.ExitOnError("Error retrieving broker admin config", err)
	brokerAdminClientset, err := kubernetes.NewForConfig(brokerAdminConfig)
	utils.ExitOnError("Error retrieving broker admin connection", err)
	brokerNamespace := string(subctlData.ClientToken.Data["namespace"])

	netconfig := globalnet.Config{ClusterID: clusterID,
		GlobalnetCIDR:           globalnetCIDR,
		ServiceCIDR:             serviceCIDR,
		ServiceCIDRAutoDetected: serviceCIDRautoDetected,
		ClusterCIDR:             clusterCIDR,
		ClusterCIDRAutoDetected: clusterCIDRautoDetected,
		GlobalnetClusterSize:    globalnetClusterSize}

	if globalnetEnabled {
		err = AllocateAndUpdateGlobalCIDRConfigMap(brokerAdminClientset, brokerNamespace, &netconfig)
		utils.ExitOnError("Error Discovering multi cluster details", err)
	}

	status.Start("Deploying the Submariner operator")

	err = submarinerop.Ensure(status, clientConfig, OperatorNamespace, operatorImage(), operatorDebug)
	status.End(cli.CheckForError(err))
	utils.ExitOnError("Error deploying the operator", err)

	status.Start("Creating SA for cluster")
	clienttoken, err = broker.CreateSAForCluster(brokerAdminClientset, clusterID)
	status.End(cli.CheckForError(err))
	utils.ExitOnError("Error creating SA for cluster", err)

	if subctlData.IsConnectivityEnabled() {
		status.Start("Deploying Submariner")
		err = submarinercr.Ensure(clientConfig, OperatorNamespace, populateSubmarinerSpec(subctlData, netconfig))
		if err == nil {
			status.QueueSuccessMessage("Submariner is up and running")
			status.End(cli.Success)
		} else {
			status.QueueFailureMessage("Submariner deployment failed")
			status.End(cli.Failure)
		}

		utils.ExitOnError("Error deploying Submariner", err)
	} else if subctlData.IsServiceDiscoveryEnabled() {
		status.Start("Deploying service discovery only")
		err = servicediscoverycr.Ensure(clientConfig, OperatorNamespace, populateServiceDiscoverySpec(subctlData))
		if err == nil {
			status.QueueSuccessMessage("Service discovery is up and running")
			status.End(cli.Success)
		} else {
			status.QueueFailureMessage("Service discovery deployment failed")
			status.End(cli.Failure)
		}
		utils.ExitOnError("Error deploying service discovery", err)
	}
}

func AllocateAndUpdateGlobalCIDRConfigMap(brokerAdminClientset *kubernetes.Clientset, brokerNamespace string,
	netconfig *globalnet.Config) error {
	status.Start("Discovering multi cluster details")
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		globalnetInfo, globalnetConfigMap, err := globalnet.GetGlobalNetworks(brokerAdminClientset, brokerNamespace)
		if err != nil {
			return fmt.Errorf("error reading Global network details on Broker: %s", err)
		}

		netconfig.GlobalnetCIDR, err = globalnet.ValidateGlobalnetConfiguration(globalnetInfo, *netconfig)
		if err != nil {
			return fmt.Errorf("error validating Globalnet configuration: %s", err)
		}

		if globalnetInfo.GlobalnetEnabled {
			netconfig.GlobalnetCIDR, err = globalnet.AssignGlobalnetIPs(globalnetInfo, *netconfig)
			if err != nil {
				return fmt.Errorf("error assigning Globalnet IPs: %s", err)
			}

			if globalnetInfo.GlobalCidrInfo[clusterID] == nil ||
				globalnetInfo.GlobalCidrInfo[clusterID].GlobalCIDRs[0] != netconfig.GlobalnetCIDR {
				var newClusterInfo broker.ClusterInfo
				newClusterInfo.ClusterID = clusterID
				newClusterInfo.GlobalCidr = []string{netconfig.GlobalnetCIDR}

				err = broker.UpdateGlobalnetConfigMap(brokerAdminClientset, brokerNamespace, globalnetConfigMap, newClusterInfo)
				return err
			}
		}
		return err
	})
	return retryErr
}

func getNetworkDetails(config *rest.Config) *network.ClusterNetwork {
	dynClient, clientSet, err := restconfig.Clients(config)
	utils.ExitOnError("Unable to set the Kubernetes cluster connection up", err)

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	utils.ExitOnError("Unable to get the Submariner client", err)

	networkDetails, err := network.Discover(dynClient, clientSet, submarinerClient, OperatorNamespace)
	if err != nil {
		status.QueueWarningMessage(fmt.Sprintf("Error trying to discover network details: %s", err))
	} else if networkDetails != nil {
		networkDetails.Show()
	}
	return networkDetails
}

func getPodCIDR(clusterCIDR string, nd *network.ClusterNetwork) (cidrType string, autodetected bool, err error) {
	if clusterCIDR != "" {
		if nd != nil && len(nd.PodCIDRs) > 0 && nd.PodCIDRs[0] != clusterCIDR {
			status.QueueWarningMessage(fmt.Sprintf("Your provided cluster CIDR for the pods (%s) does not match discovered (%s)\n",
				clusterCIDR, nd.PodCIDRs[0]))
		}
		return clusterCIDR, false, nil
	} else if nd != nil && len(nd.PodCIDRs) > 0 {
		return nd.PodCIDRs[0], true, nil
	} else {
		cidrType, err = askForCIDR("Pod")
		return cidrType, false, err
	}
}

func getServiceCIDR(serviceCIDR string, nd *network.ClusterNetwork) (cidrType string, autodetected bool, err error) {
	if serviceCIDR != "" {
		if nd != nil && len(nd.ServiceCIDRs) > 0 && nd.ServiceCIDRs[0] != serviceCIDR {
			status.QueueWarningMessage(fmt.Sprintf("Your provided service CIDR (%s) does not match discovered (%s)\n",
				serviceCIDR, nd.ServiceCIDRs[0]))
		}
		return serviceCIDR, false, nil
	} else if nd != nil && len(nd.ServiceCIDRs) > 0 {
		return nd.ServiceCIDRs[0], true, nil
	} else {
		cidrType, err = askForCIDR("ClusterIP service")
		return cidrType, false, err
	}
}

func askForCIDR(name string) (string, error) {
	var qs = []*survey.Question{{
		Name:     "cidr",
		Prompt:   &survey.Input{Message: fmt.Sprintf("What's the %s CIDR for your cluster?", name)},
		Validate: survey.Required,
	}}

	answers := struct {
		Cidr string
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		return "", err
	} else {
		return strings.TrimSpace(answers.Cidr), nil
	}
}

func isValidClusterID(clusterID string) (bool, error) {
	// Make sure the clusterid is a valid DNS-1123 string
	if match, _ := regexp.MatchString("^[a-z0-9][a-z0-9.-]*[a-z0-9]$", clusterID); !match {
		return false, fmt.Errorf("cluster IDs must be valid DNS-1123 names, with only lowercase alphanumerics,\n"+
			"'.' or '-' (and the first and last characters must be alphanumerics).\n"+
			"%s doesn't meet these requirements", clusterID)
	}
	return true, nil
}

func populateSubmarinerSpec(subctlData *datafile.SubctlData, netconfig globalnet.Config) submariner.SubmarinerSpec {
	brokerURL := subctlData.BrokerURL
	if idx := strings.Index(brokerURL, "://"); idx >= 0 {
		// Submariner doesn't work with a schema prefix
		brokerURL = brokerURL[(idx + 3):]
	}

	// if our network discovery code was capable of discovering those CIDRs
	// we don't need to explicitly set it in the operator
	crServiceCIDR := ""
	if !netconfig.ServiceCIDRAutoDetected {
		crServiceCIDR = netconfig.ServiceCIDR
	}

	crClusterCIDR := ""
	if !netconfig.ClusterCIDRAutoDetected {
		crClusterCIDR = netconfig.ClusterCIDR
	}

	if customDomains == nil && subctlData.CustomDomains != nil {
		customDomains = *subctlData.CustomDomains
	}

	submarinerSpec := submariner.SubmarinerSpec{
		Repository:               getImageRepo(),
		Version:                  getImageVersion(),
		CeIPSecNATTPort:          nattPort,
		CeIPSecIKEPort:           ikePort,
		CeIPSecDebug:             ipsecDebug,
		CeIPSecForceUDPEncaps:    forceUDPEncaps,
		CeIPSecPreferredServer:   preferredServer,
		CeIPSecPSK:               base64.StdEncoding.EncodeToString(subctlData.IPSecPSK.Data["psk"]),
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(subctlData.ClientToken.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(subctlData.ClientToken.Data["namespace"]),
		BrokerK8sApiServerToken:  string(clienttoken.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		Broker:                   "k8s",
		NatEnabled:               natTraversal,
		Debug:                    submarinerDebug,
		ColorCodes:               colorCodes,
		ClusterID:                clusterID,
		ServiceCIDR:              crServiceCIDR,
		ClusterCIDR:              crClusterCIDR,
		Namespace:                SubmarinerNamespace,
		CableDriver:              cableDriver,
		ServiceDiscoveryEnabled:  subctlData.IsServiceDiscoveryEnabled(),
		ImageOverrides:           getImageOverrides(),
		LoadBalancerEnabled:      loadBalancerEnabled,
		ConnectionHealthCheck: &submariner.HealthCheckSpec{
			Enabled:            healthCheckEnable,
			IntervalSeconds:    healthCheckInterval,
			MaxPacketLossCount: healthCheckMaxPacketLossCount,
		},
	}
	if netconfig.GlobalnetCIDR != "" {
		submarinerSpec.GlobalCIDR = netconfig.GlobalnetCIDR
	}
	if corednsCustomConfigMap != "" {
		namespace, name := getCustomCoreDNSParams()
		submarinerSpec.CoreDNSCustomConfig = &submariner.CoreDNSCustomConfig{
			ConfigMapName: name,
			Namespace:     namespace,
		}
	}
	if len(customDomains) > 0 {
		submarinerSpec.CustomDomains = customDomains
	}
	return submarinerSpec
}

func getImageVersion() string {
	version := imageVersion

	if imageVersion == "" {
		version = versions.DefaultSubmarinerOperatorVersion
	}

	return version
}

func getImageRepo() string {
	repo := repository

	if repository == "" {
		repo = versions.DefaultRepo
	}

	return repo
}

func removeSchemaPrefix(brokerURL string) string {
	if idx := strings.Index(brokerURL, "://"); idx >= 0 {
		// Submariner doesn't work with a schema prefix
		brokerURL = brokerURL[(idx + 3):]
	}

	return brokerURL
}

func populateServiceDiscoverySpec(subctlData *datafile.SubctlData) *submariner.ServiceDiscoverySpec {
	brokerURL := removeSchemaPrefix(subctlData.BrokerURL)

	if customDomains == nil && subctlData.CustomDomains != nil {
		customDomains = *subctlData.CustomDomains
	}

	serviceDiscoverySpec := submariner.ServiceDiscoverySpec{
		Repository:               repository,
		Version:                  imageVersion,
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(subctlData.ClientToken.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(subctlData.ClientToken.Data["namespace"]),
		BrokerK8sApiServerToken:  string(clienttoken.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		Debug:                    submarinerDebug,
		ClusterID:                clusterID,
		Namespace:                SubmarinerNamespace,
		ImageOverrides:           getImageOverrides(),
	}

	if corednsCustomConfigMap != "" {
		namespace, name := getCustomCoreDNSParams()
		serviceDiscoverySpec.CoreDNSCustomConfig = &submariner.CoreDNSCustomConfig{
			ConfigMapName: name,
			Namespace:     namespace,
		}
	}

	if len(customDomains) > 0 {
		serviceDiscoverySpec.CustomDomains = customDomains
	}
	return &serviceDiscoverySpec
}

func operatorImage() string {
	version := imageVersion
	repo := repository

	if imageVersion == "" {
		version = versions.DefaultSubmarinerOperatorVersion
	}

	if repository == "" {
		repo = versions.DefaultRepo
	}

	return images.GetImagePath(repo, version, names.OperatorImage, names.OperatorComponent, getImageOverrides())
}

func getImageOverrides() map[string]string {
	if len(imageOverrideArr) > 0 {
		imageOverrides := make(map[string]string)
		for _, s := range imageOverrideArr {
			key := strings.Split(s, "=")[0]
			value := strings.Split(s, "=")[1]
			imageOverrides[key] = value
		}
		return imageOverrides
	}
	return nil
}

func isValidCustomCoreDNSConfig() error {
	if corednsCustomConfigMap != "" && strings.Count(corednsCustomConfigMap, "/") > 1 {
		return fmt.Errorf("coredns-custom-configmap should be in <namespace>/<name> format, namespace is optional")
	}
	return nil
}

func getCustomCoreDNSParams() (namespace, name string) {
	if corednsCustomConfigMap != "" {
		name = corednsCustomConfigMap
		paramList := strings.Split(corednsCustomConfigMap, "/")
		if len(paramList) > 1 {
			namespace = paramList[0]
			name = paramList[1]
		}
	}
	return namespace, name
}
