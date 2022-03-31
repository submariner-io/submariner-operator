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

package diagnose

import (
	"context"
	"fmt"

	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	"github.com/submariner-io/submariner/pkg/globalnet/constants"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	mcsv1a1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

func GlobalnetConfig(clusterInfo *cluster.Info, status reporter.Interface) bool {
	mustHaveSubmariner(clusterInfo)

	if clusterInfo.Submariner.Spec.GlobalCIDR == "" {
		status.Success("Globalnet is not installed - skipping")
		return true
	}

	status.Start("Checking Globalnet configuration")
	defer status.End()

	tracker := reporter.NewTracker(status)

	checkClusterGlobalEgressIPs(clusterInfo, tracker)
	checkGlobalEgressIPs(clusterInfo, tracker)
	checkGlobalIngressIPs(clusterInfo, tracker)

	if tracker.HasFailures() {
		return false
	}

	status.Success("Globalnet is properly configured and functioning")

	return true
}

func checkClusterGlobalEgressIPs(clusterInfo *cluster.Info, status reporter.Interface) {
	clusterGlobalEgress, err := clusterInfo.ClientProducer.ForSubmariner().SubmarinerV1().ClusterGlobalEgressIPs(
		corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.Failure("Error listing the ClusterGlobalEgressIP resources: %v", err)
		return
	}

	if len(clusterGlobalEgress.Items) != 1 {
		status.Failure(
			"Found %d ClusterGlobalEgressIP resources but only the default instance (%q) is supported",
			len(clusterGlobalEgress.Items), constants.ClusterGlobalEgressIPName)
	}

	foundDefaultResource := false
	index := 0

	for index = range clusterGlobalEgress.Items {
		if clusterGlobalEgress.Items[index].Name == constants.ClusterGlobalEgressIPName {
			foundDefaultResource = true
			break
		}
	}

	if !foundDefaultResource {
		status.Failure("Couldn't find the default ClusterGlobalEgressIP resource(%q)", constants.ClusterGlobalEgressIPName)
		return
	}

	clusterGlobalEgressIP := clusterGlobalEgress.Items[index]

	numberOfIPs := 1

	if clusterGlobalEgressIP.Spec.NumberOfIPs != nil {
		numberOfIPs = *clusterGlobalEgressIP.Spec.NumberOfIPs
	}

	if numberOfIPs != len(clusterGlobalEgressIP.Status.AllocatedIPs) {
		status.Failure("The number of requested IPs (%d) does not match the number allocated (%d) for ClusterGlobalEgressIP %q",
			numberOfIPs, len(clusterGlobalEgressIP.Status.AllocatedIPs), clusterGlobalEgressIP.Name)
	}

	condition := meta.FindStatusCondition(clusterGlobalEgressIP.Status.Conditions, string(submarinerv1.GlobalEgressIPAllocated))
	if condition == nil {
		status.Failure("ClusterGlobalEgressIP %q is missing the %q status condition", clusterGlobalEgressIP.Name,
			submarinerv1.GlobalEgressIPAllocated)
	} else if condition.Status != metav1.ConditionTrue {
		status.Failure("The allocation of global IPs for ClusterGlobalEgressIP %q failed with reason %q and message %q",
			clusterGlobalEgressIP.Name, condition.Reason, condition.Message)
	}
}

func checkGlobalEgressIPs(clusterInfo *cluster.Info, status reporter.Interface) {
	globalEgressIps, err := clusterInfo.ClientProducer.ForSubmariner().SubmarinerV1().GlobalEgressIPs(
		corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.Failure("Error obtaining GlobalEgressIPs resources: %v", err)
		return
	}

	for i := range globalEgressIps.Items {
		gip := globalEgressIps.Items[i]
		numberOfIPs := 1

		if gip.Spec.NumberOfIPs != nil {
			numberOfIPs = *gip.Spec.NumberOfIPs
		}

		if numberOfIPs != len(gip.Status.AllocatedIPs) {
			status.Failure("The number of requested IPs (%d) does not match the number allocated (%d) for GlobalEgressIP %q",
				numberOfIPs, len(gip.Status.AllocatedIPs), gip.Name)
		}

		condition := meta.FindStatusCondition(gip.Status.Conditions, string(submarinerv1.GlobalEgressIPAllocated))
		if condition == nil {
			status.Failure("GlobalEgressIP %q is missing the %q status condition", gip.Name, submarinerv1.GlobalEgressIPAllocated)
			continue
		} else if condition.Status != metav1.ConditionTrue {
			status.Failure("The allocation of global IPs for GlobalEgressIP %q failed with reason %q and message %q",
				gip.Name, condition.Reason, condition.Message)
			continue
		}
	}
}

func checkGlobalIngressIPs(clusterInfo *cluster.Info, status reporter.Interface) {
	serviceExportGVR := &schema.GroupVersionResource{
		Group:    mcsv1a1.GroupVersion.Group,
		Version:  mcsv1a1.GroupVersion.Version,
		Resource: "serviceexports",
	}

	serviceExports, err := clusterInfo.ClientProducer.ForDynamic().Resource(*serviceExportGVR).Namespace(corev1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.Failure("Error listing ServiceExport resources: %v", err)
		return
	}

	for i := range serviceExports.Items {
		ns := serviceExports.Items[i].GetNamespace()
		name := serviceExports.Items[i].GetName()

		svc, err := clusterInfo.ClientProducer.ForKubernetes().CoreV1().Services(ns).Get(context.TODO(), name, metav1.GetOptions{})

		if apierrors.IsNotFound(err) {
			status.Warning("No matching Service resource found for exported service \"%s/%s\"", ns, name)
			continue
		}

		if err != nil {
			status.Failure("Error retrieving Service \"%s/%s\", %v", ns, name, err)
			continue
		}

		if svc.Spec.Type != corev1.ServiceTypeClusterIP {
			continue
		}

		globalIngress, err := clusterInfo.ClientProducer.ForSubmariner().SubmarinerV1().GlobalIngressIPs(ns).Get(context.TODO(),
			name, metav1.GetOptions{})

		if apierrors.IsNotFound(err) {
			status.Failure("No matching GlobalIngressIP resource found for exported service \"%s/%s\"", ns, name)
			continue
		}

		if err != nil {
			status.Failure("Error retrieving GlobalIngressIP for exported service \"%s/%s\": %v", ns, name, err)
			continue
		}

		if globalIngress.Status.AllocatedIP == "" {
			status.Failure("No global IP was allocated for the GlobalIngressIP associated with exported service \"%s/%s\"", ns, name)
			continue
		}

		condition := meta.FindStatusCondition(globalIngress.Status.Conditions, string(submarinerv1.GlobalEgressIPAllocated))
		if condition == nil {
			status.Failure("GlobalIngressIP %q associated with exported service \"%s/%s\" is missing"+
				"the %q status condition", globalIngress.Name, ns, name, submarinerv1.GlobalEgressIPAllocated)
			continue
		} else if condition.Status != metav1.ConditionTrue {
			status.Failure("The allocation of global IPs for GlobalIngressIP %q associated with exported"+
				"service \"%s/%s\" failed with reason %q and message %q",
				globalIngress.Name, ns, name, condition.Reason, condition.Message)
			continue
		}

		verifyInternalService(clusterInfo, status, ns, name, globalIngress)
	}
}

func verifyInternalService(clusterInfo *cluster.Info, status reporter.Interface, ns, name string,
	globalIngress *submarinerv1.GlobalIngressIP,
) {
	svcs, err := clusterInfo.ClientProducer.ForKubernetes().CoreV1().Services(ns).List(
		context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("submariner.io/exportedServiceRef=%s", name)})
	if err != nil {
		status.Failure("Error listing internal Services \"%s/%s\": %v", ns, name, err)
		return
	}

	if len(svcs.Items) == 0 {
		status.Failure("No internal service found for exported service \"%s/%s\"", ns, name)
		return
	}

	if len(svcs.Items) > 1 {
		status.Failure("Found %d internal services for exported service \"%s/%s\" - expected 1", len(svcs.Items), ns, name)
		return
	}

	if svcs.Items[0].Spec.ExternalIPs[0] != globalIngress.Status.AllocatedIP {
		status.Failure(
			"The external IP (%s) for internal service associated with exported service \"%s/%s\" doesn't"+
				"match allocated IP (%s) in GlobalIngressIP %q",
			svcs.Items[0].Spec.ExternalIPs[0], ns, name, globalIngress.Status.AllocatedIP, globalIngress.Name)
	}
}
