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
package show

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func init() {
	showCmd.AddCommand(&cobra.Command{
		Use:     "versions",
		Short:   "Shows submariner component versions",
		Long:    `This command shows the versions of the submariner components in the cluster.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(showVersions)
		},
	})
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
	operatorConfig, err := clientSet.AppsV1().Deployments(cmd.OperatorNamespace).Get(context.TODO(), names.OperatorComponent, v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving Deployment")
	}

	operatorFullImageStr := operatorConfig.Spec.Template.Spec.Containers[0].Image
	version, repository := images.ParseOperatorImage(operatorFullImageStr)
	versions = append(versions, newVersionInfoFrom(repository, names.OperatorComponent, version))

	return versions, nil
}

func getServiceDiscoveryVersions(submarinerClient submarinerclientset.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	lighthouseAgentConfig, err := submarinerClient.SubmarinerV1alpha1().ServiceDiscoveries(cmd.OperatorNamespace).Get(
		context.TODO(), names.ServiceDiscoveryCrName, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return versions, nil
		}

		return nil, errors.Wrap(err, "error retrieving Submariner resource")
	}

	versions = append(versions, newVersionInfoFrom(lighthouseAgentConfig.Spec.Repository, names.ServiceDiscoveryCrName,
		lighthouseAgentConfig.Spec.Version))

	return versions, nil
}

func getVersions(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()
	status.Start("Showing versions")

	var versions []versionImageInfo
	submarinerClient, err := submarinerclientset.NewForConfig(cluster.Config)
	utils.ExitOnError("Unable to get the Submariner client", err)

	versions = getSubmarinerVersion(cluster.Submariner, versions)

	versions, err = getOperatorVersion(cluster.KubeClient, versions)
	utils.ExitOnError("Unable to get the Operator version", err)

	versions, err = getServiceDiscoveryVersions(submarinerClient, versions)
	utils.ExitOnError("Unable to get the Service-Discovery version", err)

	printVersions(versions)
	status.End(cli.Success)

	return true
}

func showVersions(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	if cluster.Submariner == nil {
		status.Start(cmd.SubmMissingMessage)
		status.End(cli.Warning)

		return true
	}

	return getVersions(cluster)
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
