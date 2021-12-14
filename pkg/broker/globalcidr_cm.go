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
package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	GlobalCIDRConfigMapName = "submariner-globalnet-info"
	GlobalnetStatusKey      = "globalnetEnabled"
	ClusterInfoKey          = "clusterinfo"
	GlobalnetCidrRange      = "globalnetCidrRange"
	GlobalnetClusterSize    = "globalnetClusterSize"
)

type ClusterInfo struct {
	ClusterID  string   `json:"cluster_id"`
	GlobalCidr []string `json:"global_cidr"`
}

func CreateGlobalnetConfigMap(kubeClient kubernetes.Interface, globalnetEnabled bool, defaultGlobalCidrRange string,
	defaultGlobalClusterSize uint, namespace string) error {
	gnConfigMap, err := NewGlobalnetConfigMap(globalnetEnabled, defaultGlobalCidrRange, defaultGlobalClusterSize, namespace)
	if err != nil {
		return errors.Wrap(err, "error creating config map")
	}

	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), gnConfigMap, metav1.CreateOptions{})
	if err == nil || apierrors.IsAlreadyExists(err) {
		return nil
	}

	return errors.Wrapf(err, "error creating ConfigMap")
}

func NewGlobalnetConfigMap(globalnetEnabled bool, defaultGlobalCidrRange string,
	defaultGlobalClusterSize uint, namespace string) (*v1.ConfigMap, error) {
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
			GlobalnetStatusKey:   "true",
			GlobalnetCidrRange:   string(cidrRange),
			GlobalnetClusterSize: fmt.Sprint(defaultGlobalClusterSize),
			ClusterInfoKey:       "[]",
		}
	} else {
		data = map[string]string{
			GlobalnetStatusKey: "false",
			ClusterInfoKey:     "[]",
		}
	}

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GlobalCIDRConfigMapName,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}

	return cm, nil
}

func UpdateGlobalnetConfigMap(k8sClientset *kubernetes.Clientset, namespace string,
	configMap *v1.ConfigMap, newCluster ClusterInfo) error {
	var clusterInfo []ClusterInfo

	err := json.Unmarshal([]byte(configMap.Data[ClusterInfoKey]), &clusterInfo)
	if err != nil {
		return errors.Wrapf(err, "error unmarshalling ClusterInfo")
	}

	exists := false

	for k, value := range clusterInfo {
		if value.ClusterID == newCluster.ClusterID {
			clusterInfo[k].GlobalCidr = newCluster.GlobalCidr
			exists = true
		}
	}

	if !exists {
		var newEntry ClusterInfo
		newEntry.ClusterID = newCluster.ClusterID
		newEntry.GlobalCidr = newCluster.GlobalCidr
		clusterInfo = append(clusterInfo, newEntry)
	}

	data, err := json.MarshalIndent(clusterInfo, "", "\t")
	if err != nil {
		return errors.Wrapf(err, "error marshalling ClusterInfo")
	}

	configMap.Data[ClusterInfoKey] = string(data)
	_, err = k8sClientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})

	return errors.Wrapf(err, "error updating ConfigMap")
}

// nolint:wrapcheck // No need to wrap here
func GetGlobalnetConfigMap(kubeClient kubernetes.Interface, namespace string) (*v1.ConfigMap, error) {
	return kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), GlobalCIDRConfigMapName, metav1.GetOptions{})
}
