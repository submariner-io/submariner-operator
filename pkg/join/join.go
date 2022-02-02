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

package join

import (
	goerrors "errors"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/image"
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

func ClusterToBroker(brokerInfo *broker.Info, options *Options, clientProducer client.Producer,
	status reporter.Interface) error {
	err := checkRequirements(clientProducer.ForKubernetes(), options.IgnoreRequirements, status)
	if err != nil {
		return err
	}

	err = isValidCustomCoreDNSConfig(options.CoreDNSCustomConfigMap)
	if err != nil {
		return status.Error(err, "error validating custom CoreDNS config")
	}

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
		ClusterID:   options.ClusterID,
		GlobalCIDR:  options.GlobalnetCIDR,
		ClusterSize: options.GlobalnetClusterSize,
	}

	status.End()

	if options.GlobalnetEnabled {
		err = globalnet.AllocateAndUpdateGlobalCIDRConfigMap(brokerAdminClientset, brokerNamespace, &netconfig, status)
		if err != nil {
			return errors.Wrap(err, "unable to determine the global CIDR")
		}
	}

	status.Start("Deploying the Submariner operator")

	err = deploy.Operator(status, options.ImageVersion, options.Repository, options.ImageOverrideArr, options.OperatorDebug, clientProducer)
	if err != nil {
		return status.Error(err, "Error deploying the operator")
	}

	status.End()

	status.Start("Creating SA for cluster")

	brokerInfo.ClientToken, err = broker.CreateSAForCluster(brokerAdminClientset, options.ClusterID, brokerNamespace)
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

	imageOverrides, err := image.GetOverrides(options.ImageOverrideArr)
	if err != nil {
		return status.Error(err, "Error overriding Operator image")
	}

	if brokerInfo.IsConnectivityEnabled() {
		status.Start("Deploying submariner")

		err := deploy.Submariner(clientProducer, submarinerOptionsFrom(options), brokerInfo, brokerSecret, netconfig,
			imageOverrides, status)
		if err != nil {
			return status.Error(err, "Error deploying the Submariner resource")
		}

		status.Success("Submariner is up and running")
		status.End()
	} else if brokerInfo.IsServiceDiscoveryEnabled() {
		status.Start("Deploying service discovery only")

		err := deploy.ServiceDiscovery(clientProducer, serviceDiscoveryOptionsFrom(options), brokerInfo, brokerSecret,
			imageOverrides, status)
		if err != nil {
			return status.Error(err, "Error deploying the ServiceDiscovery resource")
		}

		status.Success("Service discovery is up and running")
		status.End()
	}

	return nil
}

func submarinerOptionsFrom(joinOptions *Options) *deploy.SubmarinerOptions {
	return &deploy.SubmarinerOptions{
		PreferredServer:               joinOptions.PreferredServer,
		ForceUDPEncaps:                joinOptions.ForceUDPEncaps,
		NATTraversal:                  joinOptions.NATTraversal,
		IPSecDebug:                    joinOptions.IPSecDebug,
		SubmarinerDebug:               joinOptions.SubmarinerDebug,
		LoadBalancerEnabled:           joinOptions.LoadBalancerEnabled,
		HealthCheckEnabled:            joinOptions.HealthCheckEnabled,
		NATTPort:                      joinOptions.NATTPort,
		IKEPort:                       joinOptions.IKEPort,
		HealthCheckInterval:           joinOptions.HealthCheckInterval,
		HealthCheckMaxPacketLossCount: joinOptions.HealthCheckMaxPacketLossCount,
		ClusterID:                     joinOptions.ClusterID,
		CableDriver:                   joinOptions.CableDriver,
		CoreDNSCustomConfigMap:        joinOptions.CoreDNSCustomConfigMap,
		Repository:                    joinOptions.Repository,
		ImageVersion:                  joinOptions.ImageVersion,
		CustomDomains:                 joinOptions.CustomDomains,
		ServiceCIDR:                   joinOptions.ServiceCIDR,
		ClusterCIDR:                   joinOptions.ClusterCIDR,
	}
}

func serviceDiscoveryOptionsFrom(joinOptions *Options) *deploy.ServiceDiscoveryOptions {
	return &deploy.ServiceDiscoveryOptions{
		SubmarinerDebug:        joinOptions.SubmarinerDebug,
		ClusterID:              joinOptions.ClusterID,
		CoreDNSCustomConfigMap: joinOptions.CoreDNSCustomConfigMap,
		Repository:             joinOptions.Repository,
		ImageVersion:           joinOptions.ImageVersion,
		CustomDomains:          joinOptions.CustomDomains,
	}
}

func checkRequirements(kubeClient kubernetes.Interface, ignoreRequirements bool, status reporter.Interface) error {
	_, failedRequirements, err := version.CheckRequirements(kubeClient)

	if len(failedRequirements) > 0 {
		msg := "The target cluster fails to meet Submariner's version requirements:\n"
		for i := range failedRequirements {
			msg += fmt.Sprintf("* %s\n", failedRequirements[i])
		}

		if !ignoreRequirements {
			status.Failure(msg)

			return goerrors.New("version requirements not met")
		}

		status.Warning(msg)
	}

	return status.Error(err, "unable to check version requirements")
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
