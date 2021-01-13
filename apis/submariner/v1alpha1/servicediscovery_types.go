/*
© 2021 Red Hat, Inc. and others.

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
	"github.com/submariner-io/submariner-operator/pkg/versions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add 	must have json tags for the fields to be serialized.

// ServiceDiscoverySpec defines the desired state of ServiceDiscovery
// +k8s:openapi-gen=true
type ServiceDiscoverySpec struct {
	BrokerK8sApiServer       string `json:"brokerK8sApiServer"`
	BrokerK8sApiServerToken  string `json:"brokerK8sApiServerToken"`
	BrokerK8sCA              string `json:"brokerK8sCA"`
	BrokerK8sRemoteNamespace string `json:"brokerK8sRemoteNamespace"`
	ClusterID                string `json:"clusterID"`
	Namespace                string `json:"namespace"`
	Repository               string `json:"repository,omitempty"`
	Version                  string `json:"version,omitempty"`
	Debug                    bool   `json:"debug"`
	GlobalnetEnabled         bool   `json:"globalnetEnabled,omitempty"`
	// +listType=set
	CustomDomains  []string          `json:"customDomains,omitempty"`
	ImageOverrides map[string]string `json:"imageOverrides,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make manifests" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// ServiceDiscoveryStatus defines the observed state of ServiceDiscovery
// +k8s:openapi-gen=true
type ServiceDiscoveryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make manifests" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +kubebuilder:object:root=true

// ServiceDiscovery is the Schema for the servicediscoveries API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=servicediscoveries,scope=Namespaced
// +genclient
// +operator-sdk:csv:customresourcedefinitions:displayName="Lighthouse"
type ServiceDiscovery struct {
	Status            ServiceDiscoveryStatus `json:"status,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ServiceDiscoverySpec `json:"spec,omitempty"`
	metav1.TypeMeta   `json:",inline"`
}

// +kubebuilder:object:root=true

// ServiceDiscoveryList contains a list of ServiceDiscovery
type ServiceDiscoveryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceDiscovery `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceDiscovery{}, &ServiceDiscoveryList{})
}

func (serviceDiscovery *ServiceDiscovery) SetDefaults() {
	if serviceDiscovery.Spec.Version == "" {
		serviceDiscovery.Spec.Version = versions.DefaultLighthouseVersion
	}
}
