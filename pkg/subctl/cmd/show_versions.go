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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/kubernetes"
)

var showVersionsCmd = &cobra.Command{
	Use:     "versions",
	Short:   "Shows submariner component versions",
	Long:    `This command shows the versions of the submariner components in the cluster.`,
	PreRunE: checkVersionMismatch,
	Run:     showVersions,
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

func getSubmarinerVersion(submariner *v1alpha1.Submariner, versions []versionImageInfo) []versionImageInfo {
	versions = append(versions, newVersionInfoFrom(submariner.Spec.Repository, submarinercr.SubmarinerName, submariner.Spec.Version))
	return versions
}

func getOperatorVersion(clientSet kubernetes.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	operatorConfig, err := clientSet.AppsV1().Deployments(OperatorNamespace).Get(context.TODO(), names.OperatorComponent, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	operatorFullImageStr := operatorConfig.Spec.Template.Spec.Containers[0].Image
	version, repository := images.ParseOperatorImage(operatorFullImageStr)
	versions = append(versions, newVersionInfoFrom(repository, names.OperatorComponent, version))
	return versions, nil
}

func getServiceDiscoveryVersions(submarinerClient submarinerclientset.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	lighthouseAgentConfig, err := submarinerClient.SubmarinerV1alpha1().ServiceDiscoveries(OperatorNamespace).Get(
		context.TODO(), names.ServiceDiscoveryCrName, v1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			return versions, nil
		}
		return nil, err
	}

	versions = append(versions, newVersionInfoFrom(lighthouseAgentConfig.Spec.Repository, names.ServiceDiscoveryCrName,
		lighthouseAgentConfig.Spec.Version))
	return versions, nil
}

func getVersions(config *rest.Config, submariner *v1alpha1.Submariner) []versionImageInfo {
	var versions []versionImageInfo

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	exitOnError("Unable to get Submariner client", err)

	clientSet, err := kubernetes.NewForConfig(config)
	exitOnError("Unable to get the Operator config", err)

	versions = getSubmarinerVersion(submariner, versions)
	exitOnError("Unable to get the Submariner versions", err)

	versions, err = getOperatorVersion(clientSet, versions)
	exitOnError("Unable to get the Operator version", err)

	versions, err = getServiceDiscoveryVersions(submarinerClient, versions)
	exitOnError("Unable to get the Service-Discovery version", err)

	return versions
}

func showVersionsFor(config *rest.Config, submariner *v1alpha1.Submariner) {
	versions := getVersions(config, submariner)
	printVersions(versions)
}

func showVersions(cmd *cobra.Command, args []string) {
	configs, err := restconfig.ForClusters(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)
	for _, item := range configs {
		fmt.Println()
		fmt.Printf("Showing information for cluster %q:\n", item.ClusterName)
		submariner := getSubmarinerResource(item.Config)
		if submariner == nil {
			fmt.Println(SubmMissingMessage)
		} else {
			showVersionsFor(item.Config, submariner)
		}
	}
}

func printVersions(versions []versionImageInfo) {
	template := "%-32.31s%-54.53s%-16.15s\n"
	fmt.Printf(template, "COMPONENT", "REPOSITORY", "VERSION")
	for _, item := range versions {
		fmt.Printf(
			template,
			item.component,
			item.repository,
			item.version)
	}
}
