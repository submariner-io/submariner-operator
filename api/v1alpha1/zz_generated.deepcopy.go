//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Broker) DeepCopyInto(out *Broker) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Broker.
func (in *Broker) DeepCopy() *Broker {
	if in == nil {
		return nil
	}
	out := new(Broker)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Broker) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BrokerList) DeepCopyInto(out *BrokerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Broker, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BrokerList.
func (in *BrokerList) DeepCopy() *BrokerList {
	if in == nil {
		return nil
	}
	out := new(BrokerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BrokerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BrokerSpec) DeepCopyInto(out *BrokerSpec) {
	*out = *in
	if in.Components != nil {
		in, out := &in.Components, &out.Components
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DefaultCustomDomains != nil {
		in, out := &in.DefaultCustomDomains, &out.DefaultCustomDomains
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BrokerSpec.
func (in *BrokerSpec) DeepCopy() *BrokerSpec {
	if in == nil {
		return nil
	}
	out := new(BrokerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BrokerStatus) DeepCopyInto(out *BrokerStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BrokerStatus.
func (in *BrokerStatus) DeepCopy() *BrokerStatus {
	if in == nil {
		return nil
	}
	out := new(BrokerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CoreDNSCustomConfig) DeepCopyInto(out *CoreDNSCustomConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CoreDNSCustomConfig.
func (in *CoreDNSCustomConfig) DeepCopy() *CoreDNSCustomConfig {
	if in == nil {
		return nil
	}
	out := new(CoreDNSCustomConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DaemonSetStatusWrapper) DeepCopyInto(out *DaemonSetStatusWrapper) {
	*out = *in
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(appsv1.DaemonSetStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.NonReadyContainerStates != nil {
		in, out := &in.NonReadyContainerStates, &out.NonReadyContainerStates
		*out = new([]corev1.ContainerState)
		if **in != nil {
			in, out := *in, *out
			*out = make([]corev1.ContainerState, len(*in))
			for i := range *in {
				(*in)[i].DeepCopyInto(&(*out)[i])
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DaemonSetStatusWrapper.
func (in *DaemonSetStatusWrapper) DeepCopy() *DaemonSetStatusWrapper {
	if in == nil {
		return nil
	}
	out := new(DaemonSetStatusWrapper)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentInfo) DeepCopyInto(out *DeploymentInfo) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentInfo.
func (in *DeploymentInfo) DeepCopy() *DeploymentInfo {
	if in == nil {
		return nil
	}
	out := new(DeploymentInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HealthCheckSpec) DeepCopyInto(out *HealthCheckSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HealthCheckSpec.
func (in *HealthCheckSpec) DeepCopy() *HealthCheckSpec {
	if in == nil {
		return nil
	}
	out := new(HealthCheckSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadBalancerStatusWrapper) DeepCopyInto(out *LoadBalancerStatusWrapper) {
	*out = *in
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(corev1.LoadBalancerStatus)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadBalancerStatusWrapper.
func (in *LoadBalancerStatusWrapper) DeepCopy() *LoadBalancerStatusWrapper {
	if in == nil {
		return nil
	}
	out := new(LoadBalancerStatusWrapper)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceDiscovery) DeepCopyInto(out *ServiceDiscovery) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceDiscovery.
func (in *ServiceDiscovery) DeepCopy() *ServiceDiscovery {
	if in == nil {
		return nil
	}
	out := new(ServiceDiscovery)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ServiceDiscovery) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceDiscoveryList) DeepCopyInto(out *ServiceDiscoveryList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ServiceDiscovery, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceDiscoveryList.
func (in *ServiceDiscoveryList) DeepCopy() *ServiceDiscoveryList {
	if in == nil {
		return nil
	}
	out := new(ServiceDiscoveryList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ServiceDiscoveryList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceDiscoverySpec) DeepCopyInto(out *ServiceDiscoverySpec) {
	*out = *in
	if in.CoreDNSCustomConfig != nil {
		in, out := &in.CoreDNSCustomConfig, &out.CoreDNSCustomConfig
		*out = new(CoreDNSCustomConfig)
		**out = **in
	}
	if in.CustomDomains != nil {
		in, out := &in.CustomDomains, &out.CustomDomains
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ImageOverrides != nil {
		in, out := &in.ImageOverrides, &out.ImageOverrides
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceDiscoverySpec.
func (in *ServiceDiscoverySpec) DeepCopy() *ServiceDiscoverySpec {
	if in == nil {
		return nil
	}
	out := new(ServiceDiscoverySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceDiscoveryStatus) DeepCopyInto(out *ServiceDiscoveryStatus) {
	*out = *in
	out.DeploymentInfo = in.DeploymentInfo
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceDiscoveryStatus.
func (in *ServiceDiscoveryStatus) DeepCopy() *ServiceDiscoveryStatus {
	if in == nil {
		return nil
	}
	out := new(ServiceDiscoveryStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Submariner) DeepCopyInto(out *Submariner) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Submariner.
func (in *Submariner) DeepCopy() *Submariner {
	if in == nil {
		return nil
	}
	out := new(Submariner)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Submariner) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SubmarinerList) DeepCopyInto(out *SubmarinerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Submariner, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SubmarinerList.
func (in *SubmarinerList) DeepCopy() *SubmarinerList {
	if in == nil {
		return nil
	}
	out := new(SubmarinerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SubmarinerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SubmarinerSpec) DeepCopyInto(out *SubmarinerSpec) {
	*out = *in
	if in.CoreDNSCustomConfig != nil {
		in, out := &in.CoreDNSCustomConfig, &out.CoreDNSCustomConfig
		*out = new(CoreDNSCustomConfig)
		**out = **in
	}
	if in.CustomDomains != nil {
		in, out := &in.CustomDomains, &out.CustomDomains
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ImageOverrides != nil {
		in, out := &in.ImageOverrides, &out.ImageOverrides
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ConnectionHealthCheck != nil {
		in, out := &in.ConnectionHealthCheck, &out.ConnectionHealthCheck
		*out = new(HealthCheckSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SubmarinerSpec.
func (in *SubmarinerSpec) DeepCopy() *SubmarinerSpec {
	if in == nil {
		return nil
	}
	out := new(SubmarinerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SubmarinerStatus) DeepCopyInto(out *SubmarinerStatus) {
	*out = *in
	in.GatewayDaemonSetStatus.DeepCopyInto(&out.GatewayDaemonSetStatus)
	in.RouteAgentDaemonSetStatus.DeepCopyInto(&out.RouteAgentDaemonSetStatus)
	in.GlobalnetDaemonSetStatus.DeepCopyInto(&out.GlobalnetDaemonSetStatus)
	in.LoadBalancerStatus.DeepCopyInto(&out.LoadBalancerStatus)
	if in.Gateways != nil {
		in, out := &in.Gateways, &out.Gateways
		*out = new([]v1.GatewayStatus)
		if **in != nil {
			in, out := *in, *out
			*out = make([]v1.GatewayStatus, len(*in))
			for i := range *in {
				(*in)[i].DeepCopyInto(&(*out)[i])
			}
		}
	}
	out.DeploymentInfo = in.DeploymentInfo
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SubmarinerStatus.
func (in *SubmarinerStatus) DeepCopy() *SubmarinerStatus {
	if in == nil {
		return nil
	}
	out := new(SubmarinerStatus)
	in.DeepCopyInto(out)
	return out
}
