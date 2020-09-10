package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/controller/submariner"
	"github.com/submariner-io/submariner-operator/pkg/images"
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

func newVersionInfoFrom(repository, component, version string) versionImageInfo {
	return versionImageInfo{
		component:  component,
		repository: repository,
		version:    version,
	}
}

func getSubmarinerVersion(submarinerClient submarinerclientset.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	existingCfg, err := submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).Get(submarinercr.SubmarinerName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	versions = append(versions, newVersionInfoFrom(existingCfg.Spec.Repository, submarinercr.SubmarinerName, existingCfg.Spec.Version))
	return versions, nil
}

func getOperatorVersion(clientSet kubernetes.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	operatorConfig, err := clientSet.AppsV1().Deployments(OperatorNamespace).Get(deployment.OperatorName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	operatorFullImageStr := operatorConfig.Spec.Template.Spec.Containers[0].Image
	version, repository := images.ParseOperatorImage(operatorFullImageStr)
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
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)
	for _, item := range configs {
		fmt.Println()
		fmt.Printf("Showing information for cluster %q:\n", item.context)
		versions := getVersions(item.config)
		printVersions(versions)
	}
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
