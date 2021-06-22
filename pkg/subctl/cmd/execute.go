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
	"os"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	subClientsetv1 "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Cluster struct {
	Config     *rest.Config
	Name       string
	KubeClient kubernetes.Interface
	DynClient  dynamic.Interface
	SubmClient subClientsetv1.Interface
	Submariner *v1alpha1.Submariner
}

func NewCluster(config *rest.Config, clusterName string) (*Cluster, string) {
	cluster := &Cluster{
		Config: config,
		Name:   clusterName,
	}

	var err error

	cluster.KubeClient, err = kubernetes.NewForConfig(cluster.Config)
	if err != nil {
		return nil, fmt.Sprintf("Error creating kubernetes client: %v", err)
	}

	cluster.DynClient, err = dynamic.NewForConfig(cluster.Config)
	if err != nil {
		return nil, fmt.Sprintf("Error creating dynamic client: %v", err)
	}

	cluster.SubmClient, err = subClientsetv1.NewForConfig(cluster.Config)
	if err != nil {
		return nil, fmt.Sprintf("Error creating Submariner client: %v", err)
	}

	cluster.Submariner, err = getSubmarinerResourceWithError(cluster.Config)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Sprintf("Error retrieving Submariner resource: %v", err)
	}

	return cluster, ""
}

func (c *Cluster) GetGateways() (*submarinerv1.GatewayList, error) {
	gateways, err := c.SubmClient.SubmarinerV1().Gateways(OperatorNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return gateways, nil
}

func ExecuteMultiCluster(run func(*Cluster) bool) {
	success := true

	for _, config := range restconfig.MustGetForClusters(kubeConfig, kubeContexts) {
		fmt.Printf("Cluster %q\n", config.ClusterName)

		cluster, errMsg := NewCluster(config.Config, config.ClusterName)
		if cluster == nil {
			success = false
			fmt.Println(errMsg)
			fmt.Println()
			continue
		}

		success = run(cluster) && success
		fmt.Println()
	}

	if !success {
		os.Exit(1)
	}
}
