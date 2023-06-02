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

	// Type of broker (must be "k8s").
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Broker"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Broker string `json:"broker"`

	// The broker API URL.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Broker API Server"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	BrokerK8sApiServer string `json:"brokerK8sApiServer"`

	// The broker API Token.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Broker API Token"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:password"}
	BrokerK8sApiServerToken string `json:"brokerK8sApiServerToken,omitempty"`

	// The broker certificate authority.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Broker API CA"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:password"}
	BrokerK8sCA string `json:"brokerK8sCA,omitempty"`

	BrokerK8sSecret string `json:"brokerK8sSecret,omitempty"`

	// The Broker namespace.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Broker Remote Namespace"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	BrokerK8sRemoteNamespace string `json:"brokerK8sRemoteNamespace"`

	// Cable driver implementation - any of [libreswan, wireguard, vxlan].
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Cable Driver"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:libreswan","urn:alm:descriptor:com.tectonic.ui:select:vxlan","urn:alm:descriptor:com.tectonic.ui:select:wireguard"}
	CableDriver string `json:"cableDriver,omitempty"`

	// The IPsec Pre-Shared Key which must be identical in all route agents across the cluster.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="IPsec Pre-Shared Key"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:password"}
	CeIPSecPSK string `json:"ceIPSecPSK,omitempty"`

	CeIPSecPSKSecret string `json:"ceIPSecPSKSecret,omitempty"`

	// The cluster CIDR.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Cluster CIDR"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ClusterCIDR string `json:"clusterCIDR"`

	// The cluster ID used to identify the tunnels.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Cluster ID"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ClusterID string `json:"clusterID"`

	ColorCodes string `json:"colorCodes,omitempty"`

	// The image repository.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Repository"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:advanced"}
	Repository string `json:"repository,omitempty"`

	// The service CIDR.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service CIDR"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ServiceCIDR string `json:"serviceCIDR"`

	// The Global CIDR super-net range for allocating GlobalCIDRs to each cluster.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Global CIDR"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GlobalCIDR string `json:"globalCIDR,omitempty"`

	// The namespace in which to deploy the submariner operator.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Namespace string `json:"namespace"`

	// The image tag.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Version"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:advanced"}
	Version string `json:"version,omitempty"`

	// The IPsec IKE port (500 usually).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="IPsec IKE Port"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	CeIPSecIKEPort int `json:"ceIPSecIKEPort,omitempty"`

	// The IPsec NAT traversal port (4500 usually).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="IPsec NATT Port"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:fieldDependency:natEnabled:true"}
	CeIPSecNATTPort int `json:"ceIPSecNATTPort,omitempty"`

	// Enable logging IPsec debugging information.
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="IPsec Debug"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch","urn:alm:descriptor:com.tectonic.ui:advanced"}
	CeIPSecDebug bool `json:"ceIPSecDebug"`

	// Enable this cluster as a preferred server for data-plane connections.
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="IPsec Preferred Server"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch","urn:alm:descriptor:com.tectonic.ui:advanced"}
	CeIPSecPreferredServer bool `json:"ceIPSecPreferredServer,omitempty"`

	// Force UDP encapsulation for IPsec.
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="IPsec Force UDP Encapsulation"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch","urn:alm:descriptor:com.tectonic.ui:advanced"}
	CeIPSecForceUDPEncaps bool `json:"ceIPSecForceUDPEncaps,omitempty"`

	// Enable operator debugging.
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Debug"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch","urn:alm:descriptor:com.tectonic.ui:advanced"}
	Debug bool `json:"debug"`

	// Enable NAT between clusters.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable NAT"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	NatEnabled bool `json:"natEnabled"`

	AirGappedDeployment bool `json:"airGappedDeployment,omitempty"`

	// Enable automatic Load Balancer in front of the gateways.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Load Balancer"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	LoadBalancerEnabled bool `json:"loadBalancerEnabled,omitempty"`

	// Enable support for Service Discovery (Lighthouse).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Service Discovery"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	ServiceDiscoveryEnabled bool `json:"serviceDiscoveryEnabled,omitempty"`

	BrokerK8sInsecure bool `json:"brokerK8sInsecure,omitempty"`

	// Name of the custom CoreDNS configmap to configure forwarding to Lighthouse.
	// It should be in <namespace>/<name> format where <namespace> is optional and defaults to kube-system.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="CoreDNS Custom Config"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	CoreDNSCustomConfig *CoreDNSCustomConfig `json:"coreDNSCustomConfig,omitempty"`

	// List of domains to use for multi-cluster service discovery.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom Domains"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	// +listType=set
	CustomDomains []string `json:"customDomains,omitempty"`

	// Override component images.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image Overrides"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImageOverrides map[string]string `json:"imageOverrides,omitempty"`

	// The gateway connection health check.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Connection Health Check"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	// +optional
	ConnectionHealthCheck *HealthCheckSpec `json:"connectionHealthCheck,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// SubmarinerStatus defines the observed state of Submariner.
type SubmarinerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The current NAT status.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="NAT Enabled"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	NatEnabled bool `json:"natEnabled"`

	AirGappedDeployment bool `json:"airGappedDeployment,omitempty"`

	ColorCodes string `json:"colorCodes,omitempty"`

	// The current cluster ID.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Cluster ID"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ClusterID string `json:"clusterID"`

	// The current service CIDR.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Service CIDR"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ServiceCIDR string `json:"serviceCIDR,omitempty"`

	// The current cluster CIDR.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Cluster CIDR"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ClusterCIDR string `json:"clusterCIDR,omitempty"`

	// The current global CIDR.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Global CIDR"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	GlobalCIDR string `json:"globalCIDR,omitempty"`

	// The current network plugin.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Network Plugin"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	NetworkPlugin string `json:"networkPlugin,omitempty"`

	// The status of the gateway DaemonSet.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Gateway DaemonSet Status"
	GatewayDaemonSetStatus DaemonSetStatusWrapper `json:"gatewayDaemonSetStatus,omitempty"`

	// The status of the route agent DaemonSet.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Route Agent DaemonSet Status"
	RouteAgentDaemonSetStatus DaemonSetStatusWrapper `json:"routeAgentDaemonSetStatus,omitempty"`

	// The status of the Globalnet DaemonSet.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Globalnet DaemonSet Status"
	GlobalnetDaemonSetStatus DaemonSetStatusWrapper `json:"globalnetDaemonSetStatus,omitempty"`

	// The status of the load balancer DaemonSet.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Load Balancer DaemonSet Status"
	LoadBalancerStatus LoadBalancerStatusWrapper `json:"loadBalancerStatus,omitempty"`

	// Status of the gateways in the cluster.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Gateways"
	Gateways *[]submv1.GatewayStatus `json:"gateways,omitempty"`

	// Information about the deployment.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Deployment Information"
	DeploymentInfo DeploymentInfo `json:"deploymentInfo,omitempty"`

	// The image version in use by the various Submariner DaemonSets and Deployments.
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Version"
	Version string `json:"version,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=submariners,scope=Namespaced

// Submariner is the Schema for the submariners API.
// +operator-sdk:csv:customresourcedefinitions:displayName="Submariner",resources={{Deployment,v1,submariner-operator}}
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
	// Enable the connection health check.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Connection Health Checks"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Enabled bool `json:"enabled,omitempty"`

	// The interval at which health check pings are sent.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Connection Health Check Interval"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:fieldDependency:connectionHealthCheck.enabled:true"}
	IntervalSeconds uint64 `json:"intervalSeconds,omitempty"`

	// The maximum number of packets lost at which the health checker will mark the connection as down.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Maximum Packet Loss"
	//nolint:lll // Markers can't be wrapped
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:fieldDependency:connectionHealthCheck.enabled:true"}
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
