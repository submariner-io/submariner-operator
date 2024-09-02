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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BrokerSpec defines the desired state of Broker.
type BrokerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// List of the components to be installed - any of [service-discovery, connectivity].
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Components"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Components []string `json:"components,omitempty"`

	// List of domains to use for multi-cluster service discovery.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Default Custom Domains"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:advanced"}
	DefaultCustomDomains []string `json:"defaultCustomDomains,omitempty"`

	// GlobalCIDR supernet range for allocating GlobalCIDRs to each cluster.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Globalnet CIDR Range"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:fieldDependency:globalnetEnabled:true","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GlobalnetCIDRRange string `json:"globalnetCIDRRange,omitempty"`

	// ClustersetIP supernet range for allocating ClustersetIPCIDRs to each cluster.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="ClustersetIP CIDR Range"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:advanced"}
	// +optional
	ClustersetIPCIDRRange string `json:"clustersetIPCIDRRange,omitempty"`

	// Default cluster size for GlobalCIDR allocated to each cluster (amount of global IPs).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Default Globalnet Cluster Size"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:fieldDependency:globalnetEnabled:true","urn:alm:descriptor:com.tectonic.ui:advanced"}
	DefaultGlobalnetClusterSize uint `json:"defaultGlobalnetClusterSize,omitempty"`

	// Enable support for Overlapping CIDRs in connecting clusters.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Globalnet"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	GlobalnetEnabled bool `json:"globalnetEnabled,omitempty"`

	// Enable ClustersetIP default for connecting clusters.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable ClustersetIP"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +optional
	ClustersetIPEnabled bool `json:"clustersetIPEnabled,omitempty"`
}

// BrokerStatus defines the observed state of Broker.
type BrokerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=brokers,scope=Namespaced

// Broker is the Schema for the brokers API.
// +operator-sdk:csv:customresourcedefinitions:displayName="Submariner Broker",resources={{Deployment,v1,submariner-operator}}
type Broker struct { //nolint:govet // we want to keep the traditional order
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BrokerSpec   `json:"spec,omitempty"`
	Status BrokerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BrokerList contains a list of Broker.
type BrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Broker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Broker{}, &BrokerList{})
}
