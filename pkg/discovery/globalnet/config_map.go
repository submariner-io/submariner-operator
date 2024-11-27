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

package globalnet

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/cidr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	globalCIDRConfigMapName     = "submariner-globalnet-info"
	globalnetEnabledKey         = "globalnetEnabled"
	globalnetCidrRange          = "globalnetCidrRange"
	globalnetClusterSize        = "globalnetClusterSize"
	DefaultGlobalnetCIDR        = "242.0.0.0/8"
	DefaultGlobalnetClusterSize = 65536 // i.e., x.x.x.x/16 subnet mask
)

func CreateConfigMap(ctx context.Context, client controllerClient.Client, globalnetEnabled bool, defaultGlobalCidrRange string,
	defaultGlobalClusterSize uint, namespace string,
) error {
	gnConfigMap, err := NewGlobalnetConfigMap(globalnetEnabled, defaultGlobalCidrRange, defaultGlobalClusterSize, namespace)
	if err != nil {
		return errors.Wrap(err, "error creating config map")
	}

	err = client.Create(ctx, gnConfigMap)
	if err == nil || apierrors.IsAlreadyExists(err) {
		return nil
	}

	return errors.Wrapf(err, "error creating ConfigMap")
}

func NewGlobalnetConfigMap(globalnetEnabled bool, defaultGlobalCidrRange string,
	defaultGlobalClusterSize uint, namespace string,
) (*corev1.ConfigMap, error) {
	labels := map[string]string{
		"component": "submariner-globalnet",
	}

	cidrRange, err := json.Marshal(defaultGlobalCidrRange)
	if err != nil {
		return nil, errors.Wrapf(err, "error marshalling CIDR range")
	}

	var data map[string]string
	if globalnetEnabled {
		data = map[string]string{
			globalnetEnabledKey:  "true",
			globalnetCidrRange:   string(cidrRange),
			globalnetClusterSize: strconv.FormatUint(uint64(defaultGlobalClusterSize), 10),
		}
	} else {
		data = map[string]string{
			globalnetEnabledKey: "false",
		}
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      globalCIDRConfigMapName,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}

	return cm, nil
}

func updateConfigMap(ctx context.Context, client controllerClient.Client, configMap *corev1.ConfigMap, newCluster cidr.ClusterInfo,
) error {
	err := cidr.AddClusterInfoData(configMap, newCluster)
	if err != nil {
		return errors.Wrapf(err, "error adding ClusterInfo")
	}

	err = client.Update(ctx, configMap)

	return errors.Wrapf(err, "error updating ConfigMap")
}

//nolint:wrapcheck // No need to wrap here
func GetConfigMap(ctx context.Context, client controllerClient.Client, namespace string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	return cm, client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: globalCIDRConfigMapName}, cm)
}

//nolint:wrapcheck // No need to wrap here
func DeleteConfigMap(ctx context.Context, client controllerClient.Client, namespace string) error {
	return client.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      globalCIDRConfigMapName,
		Namespace: namespace,
	}})
}
