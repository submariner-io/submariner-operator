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

package cluster

import (
	"context"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/client"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type Info struct {
	Name           string
	RestConfig     *rest.Config
	ClientProducer client.Producer
	Submariner     *v1alpha1.Submariner
}

func NewInfo(clusterName string, clientProducer client.Producer, config *rest.Config) (*Info, error) {
	info := &Info{
		Name:           clusterName,
		RestConfig:     config,
		ClientProducer: clientProducer,
	}

	submariner, err := info.ClientProducer.ForOperator().SubmarinerV1alpha1().Submariners(constants.SubmarinerNamespace).
		Get(context.TODO(), constants.SubmarinerName, metav1.GetOptions{})
	if err == nil {
		info.Submariner = submariner
	} else if !apierrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "error retrieving Submariner")
	}

	return info, nil
}

func (c *Info) GetGateways() ([]submarinerv1.Gateway, error) {
	gateways, err := c.ClientProducer.ForSubmariner().SubmarinerV1().
		Gateways(constants.OperatorNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return []submarinerv1.Gateway{}, nil
		}

		return nil, err // nolint:wrapcheck // error can't be wrapped.
	}

	return gateways.Items, nil
}

func (c *Info) HasSingleNode() (bool, error) {
	nodes, err := c.ClientProducer.ForKubernetes().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "error listing Nodes")
	}

	return len(nodes.Items) == 1, nil
}

func (c *Info) GetLocalEndpoint() (*submarinerv1.Endpoint, error) {
	endpoints, err := c.ClientProducer.ForSubmariner().SubmarinerV1().Endpoints(constants.OperatorNamespace).List(
		context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error listing Endpoints")
	}

	for i := range endpoints.Items {
		if endpoints.Items[i].Spec.ClusterID == c.Submariner.Spec.ClusterID {
			return &endpoints.Items[i], nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    submarinerv1.SchemeGroupVersion.Group,
		Resource: "endpoints",
	}, "local Endpoint")
}

func (c *Info) GetAnyRemoteEndpoint() (*submarinerv1.Endpoint, error) {
	endpoints, err := c.ClientProducer.ForSubmariner().SubmarinerV1().Endpoints(constants.OperatorNamespace).List(
		context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error listing Endpoints")
	}

	for i := range endpoints.Items {
		if endpoints.Items[i].Spec.ClusterID != c.Submariner.Spec.ClusterID {
			return &endpoints.Items[i], nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    submarinerv1.SchemeGroupVersion.Group,
		Resource: "endpoints",
	}, "remote Endpoint")
}
