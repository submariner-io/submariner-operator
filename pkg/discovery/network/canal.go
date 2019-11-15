package network

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func discoverCanalFlannelNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {

	// TODO: this must be smarter, looking for the canal daemonset, with labels k8s-app=canal
	//  and then the reference on the container volumes:
	//   - configMap:
	//          defaultMode: 420
	//          name: canal-config
	//        name: flannel-cfg
	cm, err := clientSet.CoreV1().ConfigMaps("kube-system").Get("canal-config", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	podCIDR := extractPodCIDRFromNetConfigJSON(cm)

	if podCIDR == nil {
		return nil, nil
	}

	clusterNetwork := &ClusterNetwork{
		NetworkPlugin: "canal-flannel",
		PodCIDRs:      []string{*podCIDR},
	}

	// Try to networkPluginsDiscovery the service CIDRs using the generic functions
	genNetwork, err := discoverGenericNetwork(clientSet)
	if err != nil && genNetwork != nil {
		clusterNetwork.ServiceCIDRs = genNetwork.ServiceCIDRs
	}

	return clusterNetwork, nil
}

func extractPodCIDRFromNetConfigJSON(cm *v1.ConfigMap) *string {
	netConfJson := cm.Data["net-conf.json"]
	if netConfJson == "" {
		return nil
	}
	var netConf struct {
		Network string `json:"Network"`
		// All the other fields are ignored by Unmarshal
	}
	if err := json.Unmarshal([]byte(netConfJson), &netConf); err == nil {
		return &netConf.Network
	}
	return nil
}
