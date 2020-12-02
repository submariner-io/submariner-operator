/*
© 2019 Red Hat, Inc. and others.

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"

	"github.com/submariner-io/submariner-operator/pkg/versions"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SubmarinerSpec defines the desired state of Submariner
// +k8s:openapi-gen=true
type SubmarinerSpec struct {
	Broker                   string `json:"broker"`
	BrokerK8sApiServer       string `json:"brokerK8sApiServer"`
	BrokerK8sApiServerToken  string `json:"brokerK8sApiServerToken"`
	BrokerK8sCA              string `json:"brokerK8sCA"`
	BrokerK8sRemoteNamespace string `json:"brokerK8sRemoteNamespace"`
	CableDriver              string `json:"cableDriver,omitempty"`
	CeIPSecPSK               string `json:"ceIPSecPSK"`
	ClusterCIDR              string `json:"clusterCIDR"`
	ClusterID                string `json:"clusterID"`
	ColorCodes               string `json:"colorCodes,omitempty"`
	Repository               string `json:"repository,omitempty"`
	ServiceCIDR              string `json:"serviceCIDR"`
	GlobalCIDR               string `json:"globalCIDR,omitempty"`
	Namespace                string `json:"namespace"`
	Version                  string `json:"version,omitempty"`
	CeIPSecIKEPort           int    `json:"ceIPSecIKEPort,omitempty"`
	CeIPSecNATTPort          int    `json:"ceIPSecNATTPort,omitempty"`
	CeIPSecDebug             bool   `json:"ceIPSecDebug"`
	Debug                    bool   `json:"debug"`
	NatEnabled               bool   `json:"natEnabled"`
	ServiceDiscoveryEnabled  bool   `json:"serviceDiscoveryEnabled,omitempty"`
	// +listType=set
	CustomDomains  []string          `json:"customDomains,omitempty"`
	ImageOverrides map[string]string `json:"imageOverrides,omitempty"`
	// +optional
	HealthCheckSpec *HealthCheckSpec `json:"healthCheckSpec"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make manifests" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// SubmarinerStatus defines the observed state of Submariner
// +k8s:openapi-gen=true
type SubmarinerStatus struct {
	NatEnabled                bool                    `json:"natEnabled"`
	ColorCodes                string                  `json:"colorCodes,omitempty"`
	ClusterID                 string                  `json:"clusterID"`
	ServiceCIDR               string                  `json:"serviceCIDR,omitempty"`
	ClusterCIDR               string                  `json:"clusterCIDR,omitempty"`
	GlobalCIDR                string                  `json:"globalCIDR,omitempty"`
	NetworkPlugin             string                  `json:"networkPlugin,omitempty"`
	EngineDaemonSetStatus     DaemonSetStatus         `json:"engineDaemonSetStatus,omitempty"`
	RouteAgentDaemonSetStatus DaemonSetStatus         `json:"routeAgentDaemonSetStatus,omitempty"`
	GlobalnetDaemonSetStatus  DaemonSetStatus         `json:"globalnetDaemonSetStatus,omitempty"`
	Gateways                  *[]submv1.GatewayStatus `json:"gateways,omitempty"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make manifests" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

type DaemonSetStatus struct {
	LastResourceVersion       string                   `json:"lastResourceVersion,omitempty"`
	Status                    *appsv1.DaemonSetStatus  `json:"status,omitempty"`
	NonReadyContainerStates   *[]corev1.ContainerState `json:"nonReadyContainerStates,omitempty"`
	MismatchedContainerImages bool                     `json:"mismatchedContainerImages"`
}

type HealthCheckSpec struct {
	HealthCheckEnabled bool `json:"healthCheckEnabled,omitempty"`
	// The interval upon which HealthChecker ping packets are sent
	HealthCheckIntervalSeconds uint64 `json:"healthCheckIntervalSeconds,omitempty"`
	//  Defines how long we should wait for HealthChecker ping packets before we
	//  declare the connection as down
	HealthCheckTimeoutSeconds uint64 `json:"healthCheckTimeoutSeconds,omitempty"`
}

const DefaultColorCode = "blue"

// +kubebuilder:object:root=true

// Submariner is the Schema for the submariners API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=submariners,scope=Namespaced
// +genclient
// +operator-sdk:csv:customresourcedefinitions:displayName="Submariner"
type Submariner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubmarinerSpec   `json:"spec,omitempty"`
	Status SubmarinerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SubmarinerList contains a list of Submariner
type SubmarinerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Submariner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Submariner{}, &SubmarinerList{})
}

func (submariner *Submariner) SetDefaults() {
	if submariner.Spec.Repository == "" {
		// An empty field is converted to the default upstream submariner repository where all images live
		submariner.Spec.Repository = versions.DefaultRepo
	}

	if submariner.Spec.Version == "" {
		submariner.Spec.Version = versions.DefaultSubmarinerVersion
	}

	if submariner.Spec.ColorCodes == "" {
		submariner.Spec.ColorCodes = DefaultColorCode
	}
}
