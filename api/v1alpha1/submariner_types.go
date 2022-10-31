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

package v1alpha1

import (
	"encoding/json"

	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SubmarinerSpec defines the desired state of Submariner.
type SubmarinerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Broker                   string               `json:"broker"`
	BrokerK8sApiServer       string               `json:"brokerK8sApiServer"`
	BrokerK8sApiServerToken  string               `json:"brokerK8sApiServerToken,omitempty"`
	BrokerK8sCA              string               `json:"brokerK8sCA,omitempty"`
	BrokerK8sSecret          string               `json:"brokerK8sSecret,omitempty"`
	BrokerK8sRemoteNamespace string               `json:"brokerK8sRemoteNamespace"`
	CableDriver              string               `json:"cableDriver,omitempty"`
	CeIPSecPSK               string               `json:"ceIPSecPSK,omitempty"`
	CeIPSecPSKSecret         string               `json:"ceIPSecPSKSecret,omitempty"`
	ClusterCIDR              string               `json:"clusterCIDR"`
	ClusterID                string               `json:"clusterID"`
	ColorCodes               string               `json:"colorCodes,omitempty"`
	Repository               string               `json:"repository,omitempty"`
	ServiceCIDR              string               `json:"serviceCIDR"`
	GlobalCIDR               string               `json:"globalCIDR,omitempty"`
	Namespace                string               `json:"namespace"`
	Version                  string               `json:"version,omitempty"`
	CeIPSecIKEPort           int                  `json:"ceIPSecIKEPort,omitempty"`
	CeIPSecNATTPort          int                  `json:"ceIPSecNATTPort,omitempty"`
	CeIPSecDebug             bool                 `json:"ceIPSecDebug"`
	CeIPSecPreferredServer   bool                 `json:"ceIPSecPreferredServer,omitempty"`
	CeIPSecForceUDPEncaps    bool                 `json:"ceIPSecForceUDPEncaps,omitempty"`
	Debug                    bool                 `json:"debug"`
	NatEnabled               bool                 `json:"natEnabled"`
	AirGappedDeployment      bool                 `json:"airGappedDeployment,omitempty"`
	LoadBalancerEnabled      bool                 `json:"loadBalancerEnabled,omitempty"`
	ServiceDiscoveryEnabled  bool                 `json:"serviceDiscoveryEnabled,omitempty"`
	BrokerK8sInsecure        bool                 `json:"brokerK8sInsecure,omitempty"`
	CoreDNSCustomConfig      *CoreDNSCustomConfig `json:"coreDNSCustomConfig,omitempty"`
	// +listType=set
	CustomDomains  []string          `json:"customDomains,omitempty"`
	ImageOverrides map[string]string `json:"imageOverrides,omitempty"`
	// +optional
	ConnectionHealthCheck *HealthCheckSpec `json:"connectionHealthCheck,omitempty"`
}

// SubmarinerStatus defines the observed state of Submariner.
type SubmarinerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	NatEnabled                bool                      `json:"natEnabled"`
	AirGappedDeployment       bool                      `json:"airGappedDeployment,omitempty"`
	ColorCodes                string                    `json:"colorCodes,omitempty"`
	ClusterID                 string                    `json:"clusterID"`
	ServiceCIDR               string                    `json:"serviceCIDR,omitempty"`
	ClusterCIDR               string                    `json:"clusterCIDR,omitempty"`
	GlobalCIDR                string                    `json:"globalCIDR,omitempty"`
	NetworkPlugin             string                    `json:"networkPlugin,omitempty"`
	GatewayDaemonSetStatus    DaemonSetStatusWrapper    `json:"gatewayDaemonSetStatus,omitempty"`
	RouteAgentDaemonSetStatus DaemonSetStatusWrapper    `json:"routeAgentDaemonSetStatus,omitempty"`
	GlobalnetDaemonSetStatus  DaemonSetStatusWrapper    `json:"globalnetDaemonSetStatus,omitempty"`
	LoadBalancerStatus        LoadBalancerStatusWrapper `json:"loadBalancerStatus,omitempty"`
	Gateways                  *[]submv1.GatewayStatus   `json:"gateways,omitempty"`
	DeploymentInfo            DeploymentInfo            `json:"deploymentInfo,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=submariners,scope=Namespaced

// Submariner is the Schema for the submariners API.
type Submariner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubmarinerSpec   `json:"spec,omitempty"`
	Status SubmarinerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SubmarinerList contains a list of Submariner.
type SubmarinerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Submariner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Submariner{}, &SubmarinerList{})
}

type LoadBalancerStatusWrapper struct {
	Status *corev1.LoadBalancerStatus `json:"status,omitempty"`
}

type DaemonSetStatusWrapper struct {
	LastResourceVersion       string                   `json:"lastResourceVersion,omitempty"`
	Status                    *appsv1.DaemonSetStatus  `json:"status,omitempty"`
	NonReadyContainerStates   *[]corev1.ContainerState `json:"nonReadyContainerStates,omitempty"`
	MismatchedContainerImages bool                     `json:"mismatchedContainerImages"`
}

type DeploymentInfo struct {
	KubernetesType        KubernetesType `json:"kubernetesType,omitempty"`
	KubernetesTypeVersion string         `json:"kubernetesTypeVersion,omitempty"`
	KubernetesVersion     string         `json:"kubernetesVersion,omitempty"`
	CloudProvider         CloudProvider  `json:"cloudProvider,omitempty"`
}

type HealthCheckSpec struct {
	Enabled bool `json:"enabled,omitempty"`
	// The interval at which health check pings are sent.
	IntervalSeconds uint64 `json:"intervalSeconds,omitempty"`
	// The maximum number of packets lost at which the health checker will mark the connection as down.
	MaxPacketLossCount uint64 `json:"maxPacketLossCount,omitempty"`
}

type (
	KubernetesType string
	CloudProvider  string
)

const (
	K8s                   KubernetesType = "k8s"
	OCP                                  = "ocp"
	EKS                                  = "eks"
	AKS                                  = "aks"
	GKE                                  = "gke"
	DefaultKubernetesType                = K8s
	Kind                  CloudProvider  = "kind"
	AWS                                  = "aws"
	GCP                                  = "gcp"
	Azure                                = "azure"
	Openstack                            = "openstack"
)

func (s *Submariner) UnmarshalJSON(data []byte) error {
	type submarinerAlias Submariner

	subm := &submarinerAlias{
		Spec: SubmarinerSpec{
			Repository: DefaultRepo,
			Version:    DefaultSubmarinerVersion,
		},
	}

	_ = json.Unmarshal(data, subm)

	*s = Submariner(*subm)

	return nil
}
