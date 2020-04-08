/*
© 2019 Red Hat, Inc. and others.

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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"

	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	submariner "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	lighthouse "github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/deploy"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
	"github.com/submariner-io/submariner-operator/pkg/versions"
	submarinerClientset "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
)

var (
	clusterID            string
	serviceCIDR          string
	clusterCIDR          string
	globalCIDR           string
	repository           string
	imageVersion         string
	nattPort             int
	ikePort              int
	colorCodes           string
	disableNat           bool
	ipsecDebug           bool
	submarinerDebug      bool
	noLabel              bool
	brokerClusterContext string
	cableDriver          string
	disableOpenShiftCVO  bool
	clienttoken          *v1.Secret
	globalnetClusterSize uint
)

func init() {
	addJoinFlags(joinCmd)
	addKubeconfigFlag(joinCmd)
	rootCmd.AddCommand(joinCmd)

}

func addJoinFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&clusterID, "clusterid", "", "cluster ID used to identify the tunnels")
	cmd.Flags().StringVar(&serviceCIDR, "servicecidr", "", "service CIDR")
	cmd.Flags().StringVar(&clusterCIDR, "clustercidr", "", "cluster CIDR")
	cmd.Flags().StringVar(&repository, "repository", "", "image repository")
	cmd.Flags().StringVar(&imageVersion, "version", "", "image version")
	cmd.Flags().StringVarP(&operatorImage, "operator-image", "o", DefaultOperatorImage,
		"the operator image you wish to use")
	cmd.Flags().StringVar(&colorCodes, "colorcodes", "blue", "color codes")
	cmd.Flags().IntVar(&nattPort, "nattport", 4500, "IPsec NATT port")
	cmd.Flags().IntVar(&ikePort, "ikeport", 500, "IPsec IKE port")
	cmd.Flags().BoolVar(&disableNat, "disable-nat", false, "Disable NAT for IPsec")
	cmd.Flags().BoolVar(&ipsecDebug, "ipsec-debug", false, "Enable IPsec debugging (verbose logging)")
	cmd.Flags().BoolVar(&submarinerDebug, "subm-debug", false, "Enable Submariner debugging (verbose logging)")
	cmd.Flags().BoolVar(&noLabel, "no-label", false, "skip gateway labeling")
	cmd.Flags().StringVar(&brokerClusterContext, "broker-cluster-context", "", "Broker cluster context")
	cmd.Flags().StringVar(&cableDriver, "cable-driver", "", "Cable driver implementation")
	cmd.Flags().BoolVar(&disableOpenShiftCVO, "disable-cvo", false,
		"disable OpenShift's cluster version operator if necessary, without prompting")
	cmd.Flags().UintVar(&globalnetClusterSize, "globalnet-cluster-size", 0,
		"Cluster size for GlobalCIDR allocated to this cluster (amount of global IPs)")
}

const (
	SubmarinerNamespace = "submariner-operator" // We currently expect everything in submariner-operator
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "connect a cluster to an existing broker",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := checkArgumentPassed(args)
		exitOnError("Argument missing", err)
		subctlData, err := datafile.NewFromFile(args[0])
		if subctlData.ServiceDiscovery {
			err = checkBrokerContextPassed(brokerClusterContext)
		}
		exitOnError("Argument missing", err)
		exitOnError("Error loading the broker information from the given file", err)
		fmt.Printf("* %s says broker is at: %s\n", args[0], subctlData.BrokerURL)
		exitOnError("Error connecting to broker cluster", err)
		config, err := getRestConfig(kubeConfig, kubeContext)
		exitOnError("Error connecting to the target cluster", err)
		joinSubmarinerCluster(config, subctlData)
	},
}

func checkArgumentPassed(args []string) error {
	if len(args) == 0 {
		return errors.New("broker-info.subm file generated by 'subctl deploy-broker' not passed")
	}
	return nil
}

func checkBrokerContextPassed(brokerClusterContext string) error {
	if brokerClusterContext == "" {
		return errors.New("brokerClusterContext is not passed")
	}
	return nil
}

func joinSubmarinerCluster(config *rest.Config, subctlData *datafile.SubctlData) {

	// Missing information
	var qs = []*survey.Question{}

	if subctlData.ServiceDiscovery && !disableOpenShiftCVO {
		cvoEnabled, err := isOpenShiftCVOEnabled(config)
		exitOnError("Unable to check for the OpenShift CVO", err)
		if cvoEnabled {
			// Out of sequence question so we can abort early
			disable := false
			err = survey.AskOne(&survey.Confirm{
				Message: "Enabling service discovery on OpenShift will disable OpenShift updates, do you want to continue?",
			}, &disable)
			if err == io.EOF {
				fmt.Println("\nsubctl is running non-interactively, please specify --disable-cvo to confirm the above")
				os.Exit(1)
			}
			// Most likely a programming error
			panicOnError(err)
			if !disable {
				return
			}
		}
	}

	if valid, _ := isValidClusterID(clusterID); !valid {
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
	if len(colorCodes) == 0 {
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
		panicOnError(err)

		if len(answers.ClusterID) > 0 {
			clusterID = answers.ClusterID
		}
		if len(answers.ColorCodes) > 0 {
			colorCodes = answers.ColorCodes
		}
	}
	if subctlData.GlobalnetCidrRange != "" && globalnetClusterSize != 0 && globalnetClusterSize != subctlData.GlobalnetClusterSize {
		clusterSize, err := globalnet.GetValidClusterSize(subctlData.GlobalnetCidrRange, globalnetClusterSize)
		if err != nil || clusterSize == 0 {
			exitOnError("Invalid globalnet-cluster-size", err)
		}
		subctlData.GlobalnetClusterSize = clusterSize
	}

	if !noLabel {
		err := handleNodeLabels(config)
		exitOnError("Unable to set the gateway node up", err)
	}

	status := cli.NewStatus()

	status.Start("Deploying the Submariner operator")
	err := submarinerop.Ensure(status, config, OperatorNamespace, operatorImage)
	status.End(err == nil)
	exitOnError("Error deploying the operator", err)

	if subctlData.ServiceDiscovery {
		status.Start("Deploying multi cluster service discovery")
		err = lighthouse.Ensure(status, config, "", "", false, kubeConfig, kubeContext)
		status.End(err == nil)
		exitOnError("Error deploying multi cluster service discovery", err)

		status.Start("Joining to Kubefed control plane")
		args := []string{"join"}
		if kubeConfig != "" {
			args = append(args, "--kubeconfig", kubeConfig)
		}
		if kubeContext != "" {
			args = append(args, "--cluster-context", kubeContext)
		}
		args = append(args, "--kubefed-namespace", "kubefed-operator",
			clusterID, "--host-cluster-context", brokerClusterContext)
		out, err := exec.Command("kubefedctl", args...).CombinedOutput()
		if err != nil {
			err = fmt.Errorf("kubefedctl join failed: %s\n%s", err, out)
		}
		status.End(err == nil)
		exitOnError("Error joining to Kubefed control plane", err)
	}

	fmt.Printf("* Discovering network details\n")
	networkDetails := getNetworkDetails(config)

	serviceCIDR, err = getServiceCIDR(serviceCIDR, networkDetails)
	exitOnError("Error determining the service CIDR", err)

	clusterCIDR, err = getPodCIDR(clusterCIDR, networkDetails)
	exitOnError("Error determining the pod CIDR", err)

	status.Start("Discovering multi cluster details")
	globalNetworks := getGlobalNetworks(subctlData)

	if subctlData.GlobalnetCidrRange == "" {
		// Globalnet not enabled
		err = checkOverlappingServiceCidr(globalNetworks)
		status.End(err == nil)
		exitOnError("Error validating overlapping ServiceCIDRs", err)
		err = checkOverlappingClusterCidr(globalNetworks)
		status.End(err == nil)
		exitOnError("Error validating overlapping ClusterCIDRs", err)
	} else if globalNetworks[clusterID] == nil || globalNetworks[clusterID].GlobalCIDRs == nil || len(globalNetworks[clusterID].GlobalCIDRs) <= 0 {
		// Globalnet enabled, no globalCidr configured on this cluster
		globalCIDR, err = globalnet.AllocateGlobalCIDR(globalNetworks, subctlData)
		status.End(err == nil)
		status.QueueSuccessMessage(fmt.Sprintf("Allocated GlobalCIDR: %s", globalCIDR))
		exitOnError("Globalnet failed", err)
	} else {
		// Globalnet enabled, globalCidr already configured on this cluster
		globalCIDR = globalNetworks[clusterID].GlobalCIDRs[0]
		status.QueueSuccessMessage(fmt.Sprintf("Cluster already has GlobalCIDR allocated: %s", globalNetworks[clusterID].GlobalCIDRs[0]))
	}

	status.Start("Creating SA for cluster")
	brokerAdminClientset, err := kubernetes.NewForConfig(subctlData.GetBrokerAdministratorConfig())
	exitOnError("Error retrieving broker admin config", err)
	clienttoken, err = broker.CreateSAForCluster(brokerAdminClientset, clusterID)
	status.End(err == nil)
	exitOnError("Error creating SA for cluster", err)

	status.Start("Deploying Submariner")
	err = submarinercr.Ensure(config, OperatorNamespace, populateSubmarinerSpec(subctlData))
	status.End(err == nil)
	exitOnError("Error deploying Submariner", err)
}

func checkOverlappingServiceCidr(networks map[string]*globalnet.GlobalNetwork) error {
	for k, v := range networks {
		overlap, err := globalnet.IsOverlappingCIDR(v.ServiceCIDRs, serviceCIDR)
		if err != nil {
			return fmt.Errorf("unable to validate overlapping ServiceCIDR: %s", err)
		}
		if overlap && k != clusterID {
			return fmt.Errorf("invalid service CIDR: %s overlaps with cluster %s", serviceCIDR, k)
		}
	}
	return nil
}

func checkOverlappingClusterCidr(networks map[string]*globalnet.GlobalNetwork) error {
	for k, v := range networks {
		overlap, err := globalnet.IsOverlappingCIDR(v.ClusterCIDRs, clusterCIDR)
		if err != nil {
			return fmt.Errorf("unable to validate overlapping ClusterCIDR: %s", err)
		}
		if overlap && k != clusterID {
			return fmt.Errorf("invalid ClusterCIDR: %s overlaps with cluster %s", clusterCIDR, k)
		}
	}
	return nil
}

func getGlobalNetworks(subctlData *datafile.SubctlData) map[string]*globalnet.GlobalNetwork {

	brokerConfig := subctlData.GetBrokerAdministratorConfig()
	brokerSubmClient, err := submarinerClientset.NewForConfig(brokerConfig)
	exitOnError("Unable to create submariner rest client for broker cluster", err)
	brokerNamespace := string(subctlData.ClientToken.Data["namespace"])
	globalNetworks, err := globalnet.Discover(brokerSubmClient, brokerNamespace)
	exitOnError("Error trying to discover multi-cluster network details", err)
	if globalNetworks != nil {
		globalnet.ShowNetworks(globalNetworks)
	}
	return globalNetworks
}

func getNetworkDetails(config *rest.Config) *network.ClusterNetwork {

	dynClient, clientSet, err := getClients(config)
	exitOnError("Unable to set the Kubernetes cluster connection up", err)
	networkDetails, err := network.Discover(dynClient, clientSet)
	if err != nil {
		fmt.Printf("Error trying to discover network details: %s\n", err)
	} else if networkDetails != nil {
		networkDetails.Show()
	}
	return networkDetails
}

func getPodCIDR(clusterCIDR string, nd *network.ClusterNetwork) (string, error) {
	if clusterCIDR != "" {
		if nd != nil && len(nd.PodCIDRs) > 0 && nd.PodCIDRs[0] != clusterCIDR {
			fmt.Printf("WARNING: your provided cluster CIDR for the pods (%s) does not match discovered (%s)\n",
				clusterCIDR, nd.PodCIDRs[0])
		}
		return clusterCIDR, nil
	} else if nd != nil && len(nd.PodCIDRs) > 0 {
		return nd.PodCIDRs[0], nil
	} else {
		return askForCIDR("Pod")
	}
}

func getServiceCIDR(serviceCIDR string, nd *network.ClusterNetwork) (string, error) {
	if serviceCIDR != "" {
		if nd != nil && len(nd.ServiceCIDRs) > 0 && nd.ServiceCIDRs[0] != serviceCIDR {
			fmt.Printf("WARNING: your provided service CIDR (%s) does not match discovered (%s)\n",
				serviceCIDR, nd.ServiceCIDRs[0])
		}
		return serviceCIDR, nil
	} else if nd != nil && len(nd.ServiceCIDRs) > 0 {
		return nd.ServiceCIDRs[0], nil
	} else {
		return askForCIDR("ClusterIP service")
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
		return false, fmt.Errorf("Cluster IDs must be valid DNS-1123 names, with only lowercase alphanumerics,\n"+
			"'.' or '-' (and the first and last characters must be alphanumerics).\n"+
			"%s doesn't meet these requirements\n", clusterID)
	}
	return true, nil
}

func isOpenShiftCVOEnabled(config *rest.Config) (bool, error) {
	_, clientSet, err := getClients(config)
	if err != nil {
		return false, err
	}
	deployments := clientSet.AppsV1().Deployments("openshift-cluster-version")
	scale, err := deployments.GetScale("cluster-version-operator", metav1.GetOptions{})
	if err != nil {
		if k8serrs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return scale.Spec.Replicas > 0, nil
}

func populateSubmarinerSpec(subctlData *datafile.SubctlData) submariner.SubmarinerSpec {
	brokerURL := subctlData.BrokerURL
	if idx := strings.Index(brokerURL, "://"); idx >= 0 {
		// Submariner doesn't work with a schema prefix
		brokerURL = brokerURL[(idx + 3):]
	}

	if len(repository) == 0 {
		// Default repository
		// This is handled in the operator after 0.0.1 (of the operator)
		repository = versions.DefaultSubmarinerRepo
	}

	if len(imageVersion) == 0 {
		// Default engine version
		// This is handled in the operator after 0.0.1 (of the operator)
		imageVersion = versions.DefaultSubmarinerVersion
	}

	submarinerSpec := submariner.SubmarinerSpec{
		Repository:               repository,
		Version:                  imageVersion,
		CeIPSecNATTPort:          nattPort,
		CeIPSecIKEPort:           ikePort,
		CeIPSecDebug:             ipsecDebug,
		CeIPSecPSK:               base64.StdEncoding.EncodeToString(subctlData.IPSecPSK.Data["psk"]),
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(subctlData.ClientToken.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(subctlData.ClientToken.Data["namespace"]),
		BrokerK8sApiServerToken:  string(clienttoken.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		Broker:                   "k8s",
		NatEnabled:               !disableNat,
		Debug:                    submarinerDebug,
		ColorCodes:               colorCodes,
		ClusterID:                clusterID,
		ServiceCIDR:              serviceCIDR,
		ClusterCIDR:              clusterCIDR,
		Namespace:                SubmarinerNamespace,
		CableDriver:              cableDriver,
                ServiceDiscoveryEnabled:  subctlData.ServiceDiscovery,
	}

	if globalCIDR != "" {
		submarinerSpec.GlobalCIDR = globalCIDR
	}
	return submarinerSpec
}
