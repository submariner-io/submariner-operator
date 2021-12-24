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

// nolint:gocyclo // Ignore for now.
package join

import (
	"fmt"
	"strings"

	"github.com/submariner-io/submariner-operator/internal/cluster"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/image"
	"github.com/submariner-io/submariner-operator/internal/nodes"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/deploy"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/secret"
	"github.com/submariner-io/submariner-operator/pkg/version"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// nolint:gocognit // Ignore for now.
func SubmarinerCluster(brokerInfo *broker.Info, jo *deploy.WithJoinOptions, restConfigProducer restconfig.Producer,
	status reporter.Interface, gatewayNode struct{ Node string }) error {
	status.Start("Trying to join cluster %s", jo.ClusterID)

	if jo.ClusterID == "" {
		var err error

		jo.ClusterID, err = cluster.DetermineID(restConfigProducer)
		if err != nil {
			return status.Error(err, "Error determining cluster ID of the target cluster")
		}
	}

	if valid, err := cluster.IsValidID(jo.ClusterID); !valid {
		fmt.Printf("Error: %s\n", err.Error())
	}

	err := isValidCustomCoreDNSConfig(jo.CorednsCustomConfigMap)
	if err != nil {
		return status.Error(err, "error validating custom CoreDNS config")
	}

	clientConfig, err := restConfigProducer.ForCluster()
	if err != nil {
		return status.Error(err, "Error connecting to the target cluster")
	}

	clientProducer, err := client.NewProducerFromRestConfig(clientConfig)
	if err != nil {
		return status.Error(err, "error creating client producer")
	}

	if !jo.IgnoreRequirements {
		err = checkRequirements(clientProducer.ForKubernetes())
		if err != nil {
			return status.Error(err, "Error checking Submariner requirements")
		}
	}

	if brokerInfo.IsConnectivityEnabled() && jo.LabelGateway {
		err := nodes.LabelGateways(clientProducer.ForKubernetes(), gatewayNode)
		if err != nil {
			return status.Error(err, "Unable to set the gateway node up")
		}
	}

	status.End()

	status.Start("Discovering network details")

	networkDetails, err := cluster.GetNetworkDetails(clientProducer, status)
	if err != nil {
		return status.Error(err, "Unable to discover network details")
	}

	serviceCIDR, serviceCIDRautoDetected, err := cluster.GetServiceCIDR(jo.ServiceCIDR, networkDetails, status)
	if err != nil {
		return status.Error(err, "Error determining the service CIDR")
	}

	clusterCIDR, clusterCIDRautoDetected, err := cluster.GetPodCIDR(jo.ClusterCIDR, networkDetails, status)
	if err != nil {
		return status.Error(err, "Error determining the pod CIDR")
	}

	status.End()

	status.Start("Gathering relevant information from Broker")

	brokerAdminConfig, err := brokerInfo.GetBrokerAdministratorConfig()
	if err != nil {
		return status.Error(err, "Error retrieving broker admin config")
	}

	brokerAdminClientset, err := kubernetes.NewForConfig(brokerAdminConfig)
	if err != nil {
		return status.Error(err, "Error retrieving broker admin connection")
	}

	brokerNamespace := string(brokerInfo.ClientToken.Data["namespace"])
	netconfig := globalnet.Config{
		ClusterID:               jo.ClusterID,
		GlobalnetCIDR:           jo.GlobalnetCIDR,
		ServiceCIDR:             serviceCIDR,
		ServiceCIDRAutoDetected: serviceCIDRautoDetected,
		ClusterCIDR:             clusterCIDR,
		ClusterCIDRAutoDetected: clusterCIDRautoDetected,
		GlobalnetClusterSize:    jo.GlobalnetClusterSize,
	}

	if jo.GlobalnetEnabled {
		status.Start("Discovering multi cluster details")

		err = globalnet.AllocateAndUpdateGlobalCIDRConfigMap(jo.ClusterID, brokerAdminClientset, brokerNamespace, &netconfig)
		if err != nil {
			return status.Error(err, "Error Discovering multi cluster details")
		}
	}

	status.End()

	status.Start("Deploying the Submariner operator")

	err = deploy.Operator(status, jo.ImageVersion, jo.Repository, jo.ImageOverrideArr, jo.OperatorDebug, clientProducer)
	if err != nil {
		return status.Error(err, "Error deploying the operator")
	}

	status.End()

	status.Start("Creating SA for cluster")

	brokerInfo.ClientToken, err = broker.CreateSAForCluster(brokerAdminClientset, jo.ClusterID, brokerNamespace)
	if err != nil {
		return status.Error(err, "Error creating SA for cluster")
	}

	status.End()

	status.Start("Connecting to Broker")

	// We need to connect to the broker in all cases
	brokerSecret, err := secret.Ensure(clientProducer.ForKubernetes(), constants.OperatorNamespace, populateBrokerSecret(brokerInfo))
	if err != nil {
		return status.Error(err, "Error creating broker secret for cluster")
	}

	status.End()

	imageOverrides, err := image.GetOverrides(jo.ImageOverrideArr)
	if err != nil {
		return status.Error(err, "Error overriding Operator image")
	}

	if brokerInfo.IsConnectivityEnabled() {
		status.Start("Deploying submariner")

		err := deploy.Submariner(clientProducer, jo, brokerInfo, brokerSecret, netconfig, imageOverrides, status)
		if err != nil {
			return status.Error(err, "Error deploying the Submariner resource")
		}

		status.Success("Submariner is up and running")
		status.End()
	} else if brokerInfo.IsServiceDiscoveryEnabled() {
		status.Start("Deploying service discovery only")

		err := deploy.ServiceDiscovery(clientProducer, jo, brokerInfo, brokerSecret, imageOverrides, status)
		if err != nil {
			return status.Error(err, "Error deploying the ServiceDiscovery resource")
		}

		status.Success("Service discovery is up and running")
		status.End()
	}

	return nil
}

func checkRequirements(kubeClient kubernetes.Interface) error {
	_, failedRequirements, err := version.CheckRequirements(kubeClient)
	// We display failed requirements even if an error occurred
	if len(failedRequirements) > 0 {
		fmt.Println("The target cluster fails to meet Submariner's requirements:")

		for i := range failedRequirements {
			fmt.Printf("* %s\n", (failedRequirements)[i])
		}

		return err // nolint:wrapcheck // No need to wrap
	}

	return err // nolint:wrapcheck // No need to wrap
}

func populateBrokerSecret(brokerInfo *broker.Info) *v1.Secret {
	// We need to copy the broker token secret as an opaque secret to store it in the connecting cluster
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "broker-secret-",
		},
		Type: v1.SecretTypeOpaque,
		Data: brokerInfo.ClientToken.Data,
	}
}

func isValidCustomCoreDNSConfig(corednsCustomConfigMap string) error {
	if corednsCustomConfigMap != "" && strings.Count(corednsCustomConfigMap, "/") > 1 {
		return fmt.Errorf("coredns-custom-configmap should be in <namespace>/<name> format, namespace is optional")
	}

	return nil
}
