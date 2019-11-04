package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServiceDiscoverySpec defines the desired state of ServiceDiscovery
// +k8s:openapi-gen=true
type ServiceDiscoverySpec struct {
	Version    string                      `json:"version,omitempty"`
	Repository string                      `json:"repository,omitempty"`
	Namespace  string                      `json:"namespace,omitempty"`
	Controller *ServiceDiscoveryController `json:"controller,omitempty"`
	Agent      *ServiceDiscoveryAgent      `json:"agent,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

type ServiceDiscoveryController struct {
	Enabled        bool   `json:"enabled"`
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

type ServiceDiscoveryAgent struct {
	Enabled bool `json:"enabled"`
	// TODO: INSERT ADDITIONAL SPEC FIELDS
}

// ServiceDiscoveryStatus defines the observed state of ServiceDiscovery
// +k8s:openapi-gen=true
type ServiceDiscoveryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceDiscovery is the Schema for the servicediscoveries API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=servicediscoveries,scope=Namespaced
type ServiceDiscovery struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ServiceDiscoverySpec   `json:"spec,omitempty"`
	Status            ServiceDiscoveryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceDiscoveryList contains a list of ServiceDiscovery
type ServiceDiscoveryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceDiscovery `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceDiscovery{}, &ServiceDiscoveryList{})
}
