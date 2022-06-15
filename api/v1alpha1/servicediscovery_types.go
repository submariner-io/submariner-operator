/*
Copyright 2022.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +k8s:openapi-gen=true

// ServiceDiscoverySpec defines the desired state of ServiceDiscovery
type ServiceDiscoverySpec struct {
	BrokerK8sApiServer       string               `json:"brokerK8sApiServer"`
	BrokerK8sApiServerToken  string               `json:"brokerK8sApiServerToken,omitempty"`
	BrokerK8sCA              string               `json:"brokerK8sCA,omitempty"`
	BrokerK8sSecret          string               `json:"brokerK8sSecret,omitempty"`
	BrokerK8sRemoteNamespace string               `json:"brokerK8sRemoteNamespace"`
	ClusterID                string               `json:"clusterID"`
	Namespace                string               `json:"namespace"`
	Repository               string               `json:"repository,omitempty"`
	Version                  string               `json:"version,omitempty"`
	Debug                    bool                 `json:"debug"`
	GlobalnetEnabled         bool                 `json:"globalnetEnabled,omitempty"`
	BrokerK8sInsecure        bool                 `json:"brokerK8sInsecure,omitempty"`
	CoreDNSCustomConfig      *CoreDNSCustomConfig `json:"coreDNSCustomConfig,omitempty"`
	// +listType=set
	CustomDomains  []string          `json:"customDomains,omitempty"`
	ImageOverrides map[string]string `json:"imageOverrides,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +k8s:openapi-gen=true

// ServiceDiscoveryStatus defines the observed state of ServiceDiscovery
type ServiceDiscoveryStatus struct {
	DeploymentInfo DeploymentInfo `json:"deploymentInfo,omitempty"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:path=servicediscoveries,scope=Namespaced
// +genclient
// +k8s:openapi-gen=true

// ServiceDiscovery is the Schema for the servicediscoveries API.
type ServiceDiscovery struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceDiscoverySpec   `json:"spec,omitempty"`
	Status ServiceDiscoveryStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceDiscoveryList contains a list of ServiceDiscovery
type ServiceDiscoveryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceDiscovery `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceDiscovery{}, &ServiceDiscoveryList{})
}

type CoreDNSCustomConfig struct {
	ConfigMapName string `json:"configMapName,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
}
