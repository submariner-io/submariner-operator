package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/controller/submariner"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/deployment"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/kubernetes"
)

var showVersionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "Shows submariner component versions",
	Long:  `This command shows the versions of the submariner components in the cluster.`,
	Run:   showVersions,
}

func init() {
	showCmd.AddCommand(showVersionsCmd)
}

type versionImageInfo struct {
	component  string
	repository string
	version    string
}

func newVersionInfoFrom(repository string, component string, version string) versionImageInfo {
	v := versionImageInfo{
		component:  component,
		repository: repository,
		version:    version,
	}
	return v
}

func getSubmarinerVersion(submarinerClient submarinerclientset.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	existingCfg, err := submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).Get(submarinercr.SubmarinerName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	repository := existingCfg.Spec.Repository
	if strings.Contains(repository, "localhost") && !strings.Contains(repository, submarinercr.SubmarinerName) {
		repository += "/" + submarinercr.SubmarinerName
	}
	versions = append(versions, newVersionInfoFrom(repository, submarinercr.SubmarinerName, existingCfg.Spec.Version))
	return versions, nil
}

func getOperatorVersion(clientSet kubernetes.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	operatorConfig, err := clientSet.AppsV1().Deployments(OperatorNamespace).Get(deployment.OperatorName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	operatorFullImageStr := operatorConfig.Spec.Template.Spec.Containers[0].Image
	i := strings.LastIndex(operatorFullImageStr, ":")
	var repository string
	var version string
	if i == -1 {
		repository = operatorFullImageStr
	} else {
		repository = operatorFullImageStr[:i]
		version = operatorFullImageStr[i+1:]
	}

	versions = append(versions, newVersionInfoFrom(repository, deployment.OperatorName, version))
	return versions, nil
}

func getServiceDiscoveryVersions(submarinerClient submarinerclientset.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	lighthouseAgentConfig, err := submarinerClient.SubmarinerV1alpha1().ServiceDiscoveries(OperatorNamespace).Get(
		submariner.ServiceDiscoveryCrName, v1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			return versions, nil
		}
		return nil, err
	}

	versions = append(versions, newVersionInfoFrom(lighthouseAgentConfig.Spec.Repository, submariner.ServiceDiscoveryCrName,
		lighthouseAgentConfig.Spec.Version))
	return versions, nil
}

func getVersions(config *rest.Config) []versionImageInfo {
	var versions []versionImageInfo

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	exitOnError("Unable to get Submariner client", err)

	clientSet, err := kubernetes.NewForConfig(config)
	exitOnError("Unable to get the Operator config", err)

	versions, err = getSubmarinerVersion(submarinerClient, versions)
	exitOnError("Unable to get the Submariner versions", err)

	versions, err = getOperatorVersion(clientSet, versions)
	exitOnError("Unable to get the Operator version", err)

	versions, err = getServiceDiscoveryVersions(submarinerClient, versions)
	exitOnError("Unable to get the Service-Discovery version", err)

	return versions
}

func showVersions(cmd *cobra.Command, args []string) {
	config, err := getRestConfig(kubeConfig, kubeContext)
	exitOnError("Error connecting to the target cluster", err)

	versions := getVersions(config)
	printVersions(versions)
}

func showVersionsFromConfig(config *rest.Config) {
	versions := getVersions(config)
	printVersions(versions)
}

func printVersions(versions []versionImageInfo) {
	template := "%-32s%-54s%-16s\n"
	fmt.Printf(template, "COMPONENT", "REPOSITORY", "VERSION")
	for _, item := range versions {
		fmt.Printf(
			template,
			item.component,
			item.repository,
			item.version)
	}
}
