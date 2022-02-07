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
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	operatorclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type versionImageInfo struct {
	component  string
	repository string
	version    string
}

func Versions(newCluster *cluster.Info, status reporter.Interface) bool {
	if newCluster.Submariner == nil {
		status.Warning(constants.SubmMissingMessage)

		return true
	}

	return getVersions(newCluster, status)
}

func getVersions(newCluster *cluster.Info, status reporter.Interface) bool {
	status.Start("Showing versions")

	var versions []versionImageInfo

	versions = getSubmarinerVersion(newCluster.Submariner, versions)

	versions, err := getOperatorVersion(newCluster.ClientProducer.ForKubernetes(), versions)
	if err != nil {
		status.Failure("Unable to get the Operator version %s", err)
		return false
	}

	versions, err = getServiceDiscoveryVersions(newCluster.ClientProducer.ForOperator(), versions)
	if err != nil {
		status.Failure("Unable to get the Service-Discovery version %s", err)
		return false
	}

	status.End()
	printVersions(versions)

	return true
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

func newVersionInfoFrom(repository, component, version string) versionImageInfo {
	return versionImageInfo{
		component:  component,
		repository: repository,
		version:    version,
	}
}

func getSubmarinerVersion(submariner *v1alpha1.Submariner, versions []versionImageInfo) []versionImageInfo {
	versions = append(versions, newVersionInfoFrom(submariner.Spec.Repository, constants.SubmarinerName, submariner.Spec.Version))
	return versions
}

func getOperatorVersion(clientSet kubernetes.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	operatorConfig, err := clientSet.AppsV1().Deployments(constants.OperatorNamespace).
		Get(context.TODO(), names.OperatorComponent, v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving Deployment")
	}

	operatorFullImageStr := operatorConfig.Spec.Template.Spec.Containers[0].Image
	version, repository := images.ParseOperatorImage(operatorFullImageStr)
	versions = append(versions, newVersionInfoFrom(repository, names.OperatorComponent, version))

	return versions, nil
}

func getServiceDiscoveryVersions(operatorClient operatorclientset.Interface, versions []versionImageInfo) ([]versionImageInfo, error) {
	lighthouseAgentConfig, err := operatorClient.SubmarinerV1alpha1().ServiceDiscoveries(constants.OperatorNamespace).Get(
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
