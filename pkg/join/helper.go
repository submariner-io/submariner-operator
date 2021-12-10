package join

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/coreos/go-semver/semver"
	submarinerv1a1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/image"
	"github.com/submariner-io/submariner-operator/internal/resource"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/version"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"regexp"
	"strings"
	"time"
)

func checkVersionMismatch(kubeContext, kubeConfig string) error {
	config, err := restconfig.ForCluster(kubeConfig, kubeContext)
	if err != nil {
		return fmt.Errorf("the provided kubeconfig is invalid: %s", err)
	}

	submariner := resource.GetSubmariner(config)

	if submariner != nil && submariner.Spec.Version != "" {
		subctlVer, _ := semver.NewVersion(version.Version)
		submarinerVer, _ := semver.NewVersion(submariner.Spec.Version)

		if subctlVer != nil && submarinerVer != nil && subctlVer.LessThan(*submarinerVer) {
			return fmt.Errorf(
				"the subctl version %q is older than the deployed Submariner version %q. Please upgrade your subctl version",
				version.Version, submariner.Spec.Version)
		}
	}

	return nil
}

func handleNodeLabels(config *rest.Config) error {
	_, clientset, err := restconfig.Clients(config)
	utils.ExitOnError("Unable to set the Kubernetes cluster connection up", err)
	// List Submariner-labeled nodes
	const submarinerGatewayLabel = "submariner.io/gateway"
	const trueLabel = "true"
	selector := labels.SelectorFromSet(map[string]string{submarinerGatewayLabel: trueLabel})
	labeledNodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	if len(labeledNodes.Items) > 0 {
		fmt.Printf("* There are %d labeled nodes in the cluster:\n", len(labeledNodes.Items))
		for _, node := range labeledNodes.Items {
			fmt.Printf("  - %s\n", node.GetName())
		}
	} else {
		answer, err := askForGatewayNode(clientset)
		if err != nil {
			return err
		}
		if answer.Node == "" {
			fmt.Printf("* No worker node found to label as the gateway\n")
		} else {
			err = addLabelsToNode(clientset, answer.Node, map[string]string{submarinerGatewayLabel: trueLabel})
			utils.ExitOnError("Error labeling the gateway node", err)
		}
	}
	return nil
}

func askForGatewayNode(clientset kubernetes.Interface) (struct{ Node string }, error) {
	// List the worker nodes and select one
	workerNodes, err := clientset.CoreV1().Nodes().List(
		context.TODO(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker"})
	if err != nil {
		return struct{ Node string }{}, err
	}
	if len(workerNodes.Items) == 0 {
		// In some deployments (like KIND), worker nodes are not explicitly labelled. So list non-master nodes.
		workerNodes, err = clientset.CoreV1().Nodes().List(
			context.TODO(), metav1.ListOptions{LabelSelector: "!node-role.kubernetes.io/master"})
		if err != nil {
			return struct{ Node string }{}, err
		}
		if len(workerNodes.Items) == 0 {
			return struct{ Node string }{}, nil
		}
	}

	if len(workerNodes.Items) == 1 {
		return struct{ Node string }{workerNodes.Items[0].GetName()}, nil
	}
	allNodeNames := []string{}
	for _, node := range workerNodes.Items {
		allNodeNames = append(allNodeNames, node.GetName())
	}
	var qs = []*survey.Question{
		{
			Name: "node",
			Prompt: &survey.Select{
				Message: "Which node should be used as the gateway?",
				Options: allNodeNames},
		},
	}
	answers := struct {
		Node string
	}{}
	err = survey.Ask(qs, &answers)
	if err != nil {
		return struct{ Node string }{}, err
	}
	return answers, nil
}

// this function was sourced from:
// https://github.com/kubernetes/kubernetes/blob/a3ccea9d8743f2ff82e41b6c2af6dc2c41dc7b10/test/utils/density_utils.go#L36
func addLabelsToNode(c kubernetes.Interface, nodeName string, labelsToAdd map[string]string) error {
	var tokens = make([]string, 0, len(labelsToAdd))
	for k, v := range labelsToAdd {
		tokens = append(tokens, fmt.Sprintf("%q:%q", k, v))
	}

	labelString := "{" + strings.Join(tokens, ",") + "}"
	patch := fmt.Sprintf(`{"metadata":{"labels":%v}}`, labelString)

	// retry is necessary because nodes get updated every 10 seconds, and a patch can happen
	// in the middle of an update

	var lastErr error
	err := wait.ExponentialBackoff(nodeLabelBackoff, func() (bool, error) {
		_, lastErr = c.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
		if lastErr != nil {
			if !k8serrors.IsConflict(lastErr) {
				return false, lastErr
			}
			return false, nil
		} else {
			return true, nil
		}
	})

	if errors.Is(err, wait.ErrWaitTimeout) {
		return lastErr
	}

	return err
}

var nodeLabelBackoff = wait.Backoff{
	Steps:    10,
	Duration: 1 * time.Second,
	Factor:   1.2,
	Jitter:   1,
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

			if globalnetInfo.GlobalCidrInfo[netconfig.ClusterID] == nil ||
				globalnetInfo.GlobalCidrInfo[netconfig.ClusterID].GlobalCIDRs[0] != netconfig.GlobalnetCIDR {
				var newClusterInfo broker.ClusterInfo
				newClusterInfo.ClusterID = netconfig.ClusterID
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

	networkDetails, err := network.Discover(dynClient, clientSet, submarinerClient, constants.OperatorNamespace)
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

	if len(clusterID) > 63 {
		return false, fmt.Errorf("the cluster ID %q has a length of %d characters which exceeds the maximum"+
			" supported length of 63", clusterID, len(clusterID))
	}

	return true, nil
}

func populateSubmarinerSpec(subctlData *datafile.SubctlData, netconfig globalnet.Config, jo Options) submarinerv1a1.SubmarinerSpec {
	brokerURL := removeSchemaPrefix(subctlData.BrokerURL)

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

	if jo.CustomDomains == nil && subctlData.CustomDomains != nil {
		jo.CustomDomains = *subctlData.CustomDomains
	}

	imageOverrides, err := image.GetOverrides(jo.ImageOverrideArr)
	utils.ExitOnError("Error overriding Operator image", err)

	submarinerSpec := submarinerv1a1.SubmarinerSpec{
		Repository:               getImageRepo(jo.Repository),
		Version:                  getImageVersion(jo.ImageVersion),
		CeIPSecNATTPort:          jo.NattPort,
		CeIPSecIKEPort:           jo.IkePort,
		CeIPSecDebug:             jo.IpsecDebug,
		CeIPSecForceUDPEncaps:    jo.ForceUDPEncaps,
		CeIPSecPreferredServer:   jo.PreferredServer,
		CeIPSecPSK:               base64.StdEncoding.EncodeToString(subctlData.IPSecPSK.Data["psk"]),
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(subctlData.ClientToken.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(subctlData.ClientToken.Data["namespace"]),
		BrokerK8sApiServerToken:  string(jo.Clienttoken.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		Broker:                   "k8s",
		NatEnabled:               jo.NatTraversal,
		Debug:                    jo.SubmarinerDebug,
		ColorCodes:               jo.ColorCodes,
		ClusterID:                jo.ClusterID,
		ServiceCIDR:              crServiceCIDR,
		ClusterCIDR:              crClusterCIDR,
		Namespace:                constants.SubmarinerNamespace,
		CableDriver:              jo.CableDriver,
		ServiceDiscoveryEnabled:  subctlData.IsServiceDiscoveryEnabled(),
		ImageOverrides:           imageOverrides,
		LoadBalancerEnabled:      jo.LoadBalancerEnabled,
		ConnectionHealthCheck: &submarinerv1a1.HealthCheckSpec{
			Enabled:            jo.HealthCheckEnable,
			IntervalSeconds:    jo.HealthCheckInterval,
			MaxPacketLossCount: jo.HealthCheckMaxPacketLossCount,
		},
	}
	if netconfig.GlobalnetCIDR != "" {
		submarinerSpec.GlobalCIDR = netconfig.GlobalnetCIDR
	}
	if jo.CorednsCustomConfigMap != "" {
		namespace, name := getCustomCoreDNSParams(jo.CorednsCustomConfigMap)
		submarinerSpec.CoreDNSCustomConfig = &submarinerv1a1.CoreDNSCustomConfig{
			ConfigMapName: name,
			Namespace:     namespace,
		}
	}
	if len(jo.CustomDomains) > 0 {
		submarinerSpec.CustomDomains = jo.CustomDomains
	}
	return submarinerSpec
}

func getImageVersion(imageVersion string) string {
	if imageVersion == "" {
		return submarinerv1a1.DefaultSubmarinerOperatorVersion
	}
	return imageVersion
}

func getImageRepo(repository string) string {
	if repository == "" {
		return submarinerv1a1.DefaultRepo
	}

	return repository
}

func removeSchemaPrefix(brokerURL string) string {
	if idx := strings.Index(brokerURL, "://"); idx >= 0 {
		// Submariner doesn't work with a schema prefix
		brokerURL = brokerURL[(idx + 3):]
	}

	return brokerURL
}

func populateServiceDiscoverySpec(subctlData *datafile.SubctlData, jo Options) *submarinerv1a1.ServiceDiscoverySpec {
	brokerURL := removeSchemaPrefix(subctlData.BrokerURL)

	if jo.CustomDomains == nil && subctlData.CustomDomains != nil {
		jo.CustomDomains = *subctlData.CustomDomains
	}

	imageOverrides, err := image.GetOverrides(jo.ImageOverrideArr)
	utils.ExitOnError("Error overriding Operator image", err)

	serviceDiscoverySpec := submarinerv1a1.ServiceDiscoverySpec{
		Repository:               jo.Repository,
		Version:                  jo.ImageVersion,
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(subctlData.ClientToken.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(subctlData.ClientToken.Data["namespace"]),
		BrokerK8sApiServerToken:  string(jo.Clienttoken.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		Debug:                    jo.SubmarinerDebug,
		ClusterID:                jo.ClusterID,
		Namespace:                constants.SubmarinerNamespace,
		ImageOverrides:           imageOverrides,
	}
	return &serviceDiscoverySpec
}

func isValidCustomCoreDNSConfig(corednsCustomConfigMap string) error {
	if corednsCustomConfigMap != "" && strings.Count(corednsCustomConfigMap, "/") > 1 {
		return fmt.Errorf("coredns-custom-configmap should be in <namespace>/<name> format, namespace is optional")
	}
	return nil
}

func getCustomCoreDNSParams(corednsCustomConfigMap string) (namespace, name string) {
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