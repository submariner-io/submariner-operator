package broker

import (
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

const (
	GlobalCIDRConfigMapName = "submariner-globalnet-info"
	GlobalnetStatusKey      = "globalnetEnabled"
	ClusterInfoKey          = "clusterinfo"
	GlobalnetCidrRange      = "globalnetCidrRange"
	GlobalnetClusterSize    = "globalnetClusterSize"
)

type ClusterInfo struct {
	ClusterId  string   `json:"cluster_id"`
	GlobalCidr []string `json:"global_cidr"`
}

func CreateGlobalnetConfigMap(config *rest.Config, globalnetEnabled bool, defaultGlobalCidrRange string,
	defaultGlobalClusterSize uint, namespace string) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the core kubernetes clientset: %s", err)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := clientset.CoreV1().ConfigMaps(namespace).Get(GlobalCIDRConfigMapName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			// ConfigMap does not exist on the Broker, create it.
			gnConfigMap, err := NewGlobalnetConfigMap(globalnetEnabled, defaultGlobalCidrRange, defaultGlobalClusterSize, namespace)
			if err == nil {
				_, err = clientset.CoreV1().ConfigMaps(namespace).Create(gnConfigMap)
				return err
			}
		}
		return err
	})
	return retryErr
}

func NewGlobalnetConfigMap(globalnetEnabled bool, defaultGlobalCidrRange string,
	defaultGlobalClusterSize uint, namespace string) (*v1.ConfigMap, error) {
	labels := map[string]string{
		"component": "submariner-globalnet",
	}

	cidrRange, err := json.Marshal(defaultGlobalCidrRange)
	if err != nil {
		return nil, err
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

func UpdateGlobalnetConfigMap(k8sClientset *kubernetes.Clientset, namespace string, newCluster ClusterInfo) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		configMap, err := k8sClientset.CoreV1().ConfigMaps(namespace).Get(GlobalCIDRConfigMapName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		var clusterInfo []ClusterInfo
		err = json.Unmarshal([]byte(configMap.Data[ClusterInfoKey]), &clusterInfo)
		if err != nil {
			return err
		}

		exists := false
		if len(clusterInfo) > 0 {
			for k, value := range clusterInfo {
				if value.ClusterId == newCluster.ClusterId {
					clusterInfo[k].GlobalCidr = newCluster.GlobalCidr
					exists = true
				}
			}
		}

		if !exists {
			var newEntry ClusterInfo
			newEntry.ClusterId = newCluster.ClusterId
			newEntry.GlobalCidr = newCluster.GlobalCidr
			clusterInfo = append(clusterInfo, newEntry)
		}

		data, err := json.MarshalIndent(clusterInfo, "", "\t")
		if err != nil {
			return err
		}

		configMap.Data[ClusterInfoKey] = string(data)
		_, err = k8sClientset.CoreV1().ConfigMaps(namespace).Update(configMap)
		return err
	})
	return retryErr
}

func GetGlobalnetConfigMap(k8sClientset *kubernetes.Clientset, namespace string) (*v1.ConfigMap, error) {
	cm, err := k8sClientset.CoreV1().ConfigMaps(namespace).Get(GlobalCIDRConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cm, nil
}
