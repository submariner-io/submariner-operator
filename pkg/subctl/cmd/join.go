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
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/image"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/client"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/secret"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/servicediscoverycr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
	"github.com/submariner-io/submariner-operator/pkg/version"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	natTraversal                  bool
	ignoreRequirements            bool
	globalnetEnabled              bool
	ipsecDebug                    bool
	submarinerDebug               bool
	operatorDebug                 bool
	labelGateway                  bool
	loadBalancerEnabled           bool
	multiActiveGatewayEnabled     bool
	cableDriver                   string
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
	restConfigProducer.AddKubeContextFlag(joinCmd)
	rootCmd.AddCommand(joinCmd)
}

func addJoinFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&clusterID, "clusterid", "", "cluster ID used to identify the tunnels")
	cmd.Flags().StringVar(&serviceCIDR, "servicecidr", "", "service CIDR")
	cmd.Flags().StringVar(&clusterCIDR, "clustercidr", "", "cluster CIDR")
	cmd.Flags().StringVar(&repository, "repository", "", "image repository")
	cmd.Flags().StringVar(&imageVersion, "version", "", "image version")
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
	cmd.Flags().BoolVar(&multiActiveGatewayEnabled, "multi-active-gateway", false,
		"enable/disable Multiple Active Gateways for this cluster")
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
	cmd.Flags().BoolVar(&ignoreRequirements, "ignore-requirements", false, "ignore requirement failures (unsupported)")
}

const (
	SubmarinerNamespace = "submariner-operator" // We currently expect everything in submariner-operator
)

var joinCmd = &cobra.Command{
	Use:     "join",
	Short:   "Connect a cluster to an existing broker",
	Args:    cobra.MaximumNArgs(1),
	PreRunE: restConfigProducer.CheckVersionMismatch,
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
		joinSubmarinerCluster(subctlData)
	},
}

func checkArgumentPassed(args []string) error {
	if len(args) == 0 {
		return errors.New("broker-info.subm file generated by 'subctl deploy-broker' not passed")
	}

	return nil
}

var status = cli.NewStatus()

func joinSubmarinerCluster(subctlData *datafile.SubctlData) {
	// Missing information
	qs := []*survey.Question{}

	determineClusterID()

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

	if len(qs) > 0 {
		answers := struct {
			ClusterID string
		}{}

		err := survey.Ask(qs, &answers)
		// Most likely a programming error
		utils.PanicOnError(err)

		if len(answers.ClusterID) > 0 {
			clusterID = answers.ClusterID
		}
	}

	clientConfig, err := restConfigProducer.ClientConfig().ClientConfig()
	utils.ExitOnError("Error connecting to the target cluster", err)

	clientProducer, err := client.NewProducerFromRestConfig(clientConfig)
	utils.ExitOnError("Error creating client producer", err)

	checkRequirements(clientProducer.ForKubernetes())

	if subctlData.IsConnectivityEnabled() && labelGateway {
		err := handleNodeLabels(clientConfig)
		utils.ExitOnError("Unable to set the gateway node up", err)
	}

	status.Start("Discovering network details")

	networkDetails := getNetworkDetails(clientConfig)

	status.EndWith(cli.Success)

	err = determineServiceCIDR(networkDetails)
	utils.ExitOnError("Error determining the service CIDR", err)

	err = determinePodCIDR(networkDetails)
	utils.ExitOnError("Error determining the pod CIDR", err)

	brokerAdminConfig, err := subctlData.GetBrokerAdministratorConfig()
	utils.ExitOnError("Error retrieving broker admin config", err)

	brokerAdminClientset, err := kubernetes.NewForConfig(brokerAdminConfig)
	utils.ExitOnError("Error retrieving broker admin connection", err)

	brokerNamespace := string(subctlData.ClientToken.Data["namespace"])
	netconfig := globalnet.Config{
		ClusterID:   clusterID,
		GlobalCIDR:  globalnetCIDR,
		ClusterSize: globalnetClusterSize,
	}

	if globalnetEnabled {
		err = globalnet.AllocateAndUpdateGlobalCIDRConfigMap(brokerAdminClientset, brokerNamespace, &netconfig, status)
		exit.OnError(err)
	}

	status.Start("Deploying the Submariner operator")

	operatorImage, err := image.ForOperator(imageVersion, repository, imageOverrideArr)
	utils.ExitOnError("Error overriding Operator Image", err)
	err = submarinerop.Ensure(status, clientProducer, OperatorNamespace, operatorImage, operatorDebug)
	status.EndWith(cli.CheckForError(err))
	utils.ExitOnError("Error deploying the operator", err)

	status.Start("Creating SA for cluster")

	subctlData.ClientToken, err = broker.CreateSAForCluster(brokerAdminClientset, clusterID, brokerNamespace)
	status.EndWith(cli.CheckForError(err))
	utils.ExitOnError("Error creating SA for cluster", err)

	// We need to connect to the broker in all cases
	brokerSecret, err := secret.Ensure(clientProducer.ForKubernetes(), OperatorNamespace, populateBrokerSecret(subctlData))

	utils.ExitOnError("Error creating broker secret for cluster", err)

	if subctlData.IsConnectivityEnabled() {
		status.Start("Deploying Submariner")

		pskSecret, err := secret.Ensure(clientProducer.ForKubernetes(), OperatorNamespace, populatePSKSecret(subctlData))

		utils.ExitOnError("Error creating PSK secret for cluster", err)

		err = submarinercr.Ensure(clientProducer.ForOperator(), OperatorNamespace, populateSubmarinerSpec(subctlData,
			brokerSecret, pskSecret, netconfig))
		if err == nil {
			status.QueueSuccessMessage("Submariner is up and running")
			status.EndWith(cli.Success)
		} else {
			status.QueueFailureMessage("Submariner deployment failed")
			status.EndWith(cli.Failure)
		}

		utils.ExitOnError("Error deploying Submariner", err)
	} else if subctlData.IsServiceDiscoveryEnabled() {
		status.Start("Deploying service discovery only")
		err = servicediscoverycr.Ensure(clientProducer.ForOperator(), OperatorNamespace, populateServiceDiscoverySpec(subctlData, brokerSecret))
		if err == nil {
			status.QueueSuccessMessage("Service discovery is up and running")
			status.EndWith(cli.Success)
		} else {
			status.QueueFailureMessage("Service discovery deployment failed")
			status.EndWith(cli.Failure)
		}
		utils.ExitOnError("Error deploying service discovery", err)
	}
}

func checkRequirements(k8sclient kubernetes.Interface) {
	_, failedRequirements, err := version.CheckRequirements(k8sclient)
	// We display failed requirements even if an error occurred
	if len(failedRequirements) > 0 {
		fmt.Println("The target cluster fails to meet Submariner's requirements:")

		for i := range failedRequirements {
			fmt.Printf("* %s\n", (failedRequirements)[i])
		}

		if !ignoreRequirements {
			utils.ExitOnError("Unable to check all requirements", err)
			os.Exit(1)
		}
	}

	utils.ExitOnError("Unable to check requirements", err)
}

func determineClusterID() {
	if clusterID == "" {
		clusterName, err := restConfigProducer.ClusterNameFromContext()
		utils.ExitOnError("Error connecting to the target cluster", err)

		if clusterName != nil {
			clusterID = *clusterName
		}
	}
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

func determinePodCIDR(nd *network.ClusterNetwork) error {
	if clusterCIDR != "" {
		if nd != nil && len(nd.PodCIDRs) > 0 && nd.PodCIDRs[0] != clusterCIDR {
			status.QueueWarningMessage(fmt.Sprintf("Your provided cluster CIDR for the pods (%s) does not match discovered (%s)\n",
				clusterCIDR, nd.PodCIDRs[0]))
		}
	} else if nd == nil || len(nd.ServiceCIDRs) == 0 {
		var err error
		clusterCIDR, err = askForCIDR("Pod")

		return err
	}

	return nil
}

func determineServiceCIDR(nd *network.ClusterNetwork) error {
	if serviceCIDR != "" {
		if nd != nil && len(nd.ServiceCIDRs) > 0 && nd.ServiceCIDRs[0] != serviceCIDR {
			status.QueueWarningMessage(fmt.Sprintf("Your provided service CIDR (%s) does not match discovered (%s)\n",
				serviceCIDR, nd.ServiceCIDRs[0]))
		}
	} else if nd == nil || len(nd.ServiceCIDRs) == 0 {
		var err error
		serviceCIDR, err = askForCIDR("Service")

		return err
	}

	return nil
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

func isValidClusterID(clusterID string) (bool, error) {
	// Make sure the clusterid is a valid DNS-1123 string
	if match, _ := regexp.MatchString("^[a-z0-9][a-z0-9.-]*[a-z0-9]$", clusterID); !match {
		return false, fmt.Errorf("cluster IDs must be valid DNS-1123 names, with only lowercase alphanumerics,\n"+
			"'.' or '-' (and the first and last characters must be alphanumerics).\n"+
			"%s doesn't meet these requirements", clusterID)
	}

	if len(clusterID) > 63 {
		return false, fmt.Errorf("the cluster ID %q has a length of %d characters which exceeds the maximum"+
			" supported length of 63", clusterID, len(clusterID))
	}

	return true, nil
}

func populateBrokerSecret(subctlData *datafile.SubctlData) *v1.Secret {
	// We need to copy the broker token secret as an opaque secret to store it in the connecting cluster
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "broker-secret-",
		},
		Type: v1.SecretTypeOpaque,
		Data: subctlData.ClientToken.Data,
	}
}

func populatePSKSecret(subctlData *datafile.SubctlData) *v1.Secret {
	return subctlData.IPSecPSK
}

func populateSubmarinerSpec(subctlData *datafile.SubctlData, brokerSecret, pskSecret *v1.Secret,
	netconfig globalnet.Config) *submariner.SubmarinerSpec {
	brokerURL := subctlData.BrokerURL
	if idx := strings.Index(brokerURL, "://"); idx >= 0 {
		// Submariner doesn't work with a schema prefix
		brokerURL = brokerURL[(idx + 3):]
	}

	if customDomains == nil && subctlData.CustomDomains != nil {
		customDomains = *subctlData.CustomDomains
	}

	imageOverrides, err := image.GetOverrides(imageOverrideArr)
	utils.ExitOnError("Error overriding Operator image", err)

	// For backwards compatibility, the connection information is populated through the secret and individual components
	// TODO skitt This will be removed in the release following 0.12
	submarinerSpec := &submariner.SubmarinerSpec{
		Repository:                getImageRepo(),
		Version:                   getImageVersion(),
		CeIPSecNATTPort:           nattPort,
		CeIPSecIKEPort:            ikePort,
		CeIPSecDebug:              ipsecDebug,
		CeIPSecForceUDPEncaps:     forceUDPEncaps,
		CeIPSecPreferredServer:    preferredServer,
		CeIPSecPSK:                base64.StdEncoding.EncodeToString(subctlData.IPSecPSK.Data["psk"]),
		CeIPSecPSKSecret:          pskSecret.ObjectMeta.Name,
		BrokerK8sCA:               base64.StdEncoding.EncodeToString(brokerSecret.Data["ca.crt"]),
		BrokerK8sRemoteNamespace:  string(brokerSecret.Data["namespace"]),
		BrokerK8sApiServerToken:   string(brokerSecret.Data["token"]),
		BrokerK8sApiServer:        brokerURL,
		BrokerK8sSecret:           brokerSecret.ObjectMeta.Name,
		Broker:                    "k8s",
		NatEnabled:                natTraversal,
		Debug:                     submarinerDebug,
		ClusterID:                 clusterID,
		ServiceCIDR:               serviceCIDR,
		ClusterCIDR:               clusterCIDR,
		Namespace:                 SubmarinerNamespace,
		CableDriver:               cableDriver,
		ServiceDiscoveryEnabled:   subctlData.IsServiceDiscoveryEnabled(),
		ImageOverrides:            imageOverrides,
		LoadBalancerEnabled:       loadBalancerEnabled,
		MultiActiveGatewayEnabled: multiActiveGatewayEnabled,
		ConnectionHealthCheck: &submariner.HealthCheckSpec{
			Enabled:            healthCheckEnable,
			IntervalSeconds:    healthCheckInterval,
			MaxPacketLossCount: healthCheckMaxPacketLossCount,
		},
	}
	if netconfig.GlobalCIDR != "" {
		submarinerSpec.GlobalCIDR = netconfig.GlobalCIDR
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
	if imageVersion == "" {
		return submariner.DefaultSubmarinerOperatorVersion
	}

	return imageVersion
}

func getImageRepo() string {
	repo := repository

	if repository == "" {
		repo = submariner.DefaultRepo
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

func populateServiceDiscoverySpec(subctlData *datafile.SubctlData, brokerSecret *v1.Secret) *submariner.ServiceDiscoverySpec {
	brokerURL := removeSchemaPrefix(subctlData.BrokerURL)

	if customDomains == nil && subctlData.CustomDomains != nil {
		customDomains = *subctlData.CustomDomains
	}

	imageOverrides, err := image.GetOverrides(imageOverrideArr)
	utils.ExitOnError("Error overriding Operator image", err)

	serviceDiscoverySpec := submariner.ServiceDiscoverySpec{
		Repository:               repository,
		Version:                  imageVersion,
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(brokerSecret.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(brokerSecret.Data["namespace"]),
		BrokerK8sApiServerToken:  string(brokerSecret.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		BrokerK8sSecret:          brokerSecret.ObjectMeta.Name,
		Debug:                    submarinerDebug,
		ClusterID:                clusterID,
		Namespace:                SubmarinerNamespace,
		ImageOverrides:           imageOverrides,
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
