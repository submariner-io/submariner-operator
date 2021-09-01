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

package gather

import (
	"fmt"

	"github.com/submariner-io/submariner-operator/pkg/subctl/resource"
	v1 "k8s.io/api/core/v1"
)

const (
	typeIPTables = "iptables"
	typeOvn      = "ovn"
	typeUnknown  = "unknown"
	libreswan    = "libreswan"
)

var systemCmds = map[string]string{
	"ip-a":              "ip -d a",
	"ip-l":              "ip -d l",
	"ip-routes":         "ip route show",
	"ip-rules":          "ip rule list",
	"ip-rules-table150": "ip rule show table 150",
	"sysctl-a":          "sysctl -a",
}

var ipGatewayCmds = map[string]string{
	"ip-routes-table150": "ip route show table 150",
}

var ipTablesCmds = map[string]string{
	"iptables":     "iptables -L -n -v --line-numbers",
	"iptables-nat": "iptables -L -n -v --line-numbers -t nat",
}

var libreswanCmds = map[string]string{
	"ip-xfrm-policy":      "ip xfrm policy",
	"ip-xfrm-state":       "ip xfrm state",
	"ipsec-status":        "ipsec status",
	"ipsec-trafficstatus": "ipsec --trafficstatus",
}

var globalnetCmds = map[string]string{
	"ipset-list": "ipset list",
}

const ovnShowCmd = "ovn-nbctl show"

var ovnCmds = map[string]string{
	"ovn_show":                           ovnShowCmd,
	"ovn_lr_ovn_cluster_router_policies": "ovn-nbctl lr-policy-list ovn_cluster_router",
	"ovn_lr_ovn_cluster_router_routes":   "ovn-nbctl lr-route-list ovn_cluster_router",
	"ovn_lr_submariner_router_routes":    "ovn-nbctl lr-route-list submariner_router",
	"ovn_logical_routers":                "ovn-nbctl list Logical_Router",
	"ovn_lrps":                           "ovn-nbctl list Logical_Router_Port",
	"ovn_logical_switches":               "ovn-nbctl list Logical_Switch",
	"ovn_lsps":                           "ovn-nbctl list Logical_Switch_Port",
	"ovn_routes":                         "ovn-nbctl list Logical_Router_Static_Route",
	"ovn_policies":                       "ovn-nbctl list Logical_Router_Policy",
	"ovn_acls":                           "ovn-nbctl list ACL",
}

var networkPluginCNIType = map[string]string{
	"generic":       typeIPTables,
	"canal-flannel": typeIPTables,
	"weave-net":     typeIPTables,
	"OpenShiftSDN":  typeIPTables,
	"OVNKubernetes": typeOvn,
	"unknown":       typeUnknown,
}

func gatherCNIResources(info Info, networkPlugin string) {
	logPodInfo(info, "CNI data", routeagentPodLabel, func(info Info, pod *v1.Pod) {
		logSystemCmds(info, pod)
		switch networkPluginCNIType[networkPlugin] {
		case typeIPTables:
			logIPTablesCmds(info, pod)
		case typeOvn:
			// no-op. Handled in OVNResources()
		case typeUnknown:
			info.Status.QueueFailureMessage("Unsupported CNI Type")
		}
	})

	logCNIGatewayNodeResources(info)
	logGlobalnetCmds(info)
}

func logCNIGatewayNodeResources(info Info) {
	logPodInfo(info, "CNI data", gatewayPodLabel, logIPGatewayCmds)
}

func logSystemCmds(info Info, pod *v1.Pod) {
	for name, cmd := range systemCmds {
		logCmdOutput(info, pod, cmd, name, false)
	}
}

func logIPGatewayCmds(info Info, pod *v1.Pod) {
	for name, cmd := range ipGatewayCmds {
		logCmdOutput(info, pod, cmd, name, true)
	}
}

func logIPTablesCmds(info Info, pod *v1.Pod) {
	for name, cmd := range ipTablesCmds {
		logCmdOutput(info, pod, cmd, name, false)
	}
}

func logGlobalnetCmds(info Info) {
	logPodInfo(info, "globalnet data", globalnetPodLabel, func(info Info, pod *v1.Pod) {
		for name, cmd := range globalnetCmds {
			logCmdOutput(info, pod, cmd, name, false)
		}
	})
}

func gatherOVNResources(info Info, networkPlugin string) {
	if networkPluginCNIType[networkPlugin] != typeOvn {
		return
	}

	// we check two different labels because OpenShift deploys with a different
	// label compared to ovn-kubernetes upstream
	pods, err := findPods(info.ClientSet, ovnMasterPodLabelOCP)
	if err != nil || pods == nil || len(pods.Items) == 0 {
		pods, err = findPods(info.ClientSet, ovnMasterPodLabelGeneric)
		if err != nil {
			info.Status.QueueFailureMessage("Failed to gather any OVN master pods: " + err.Error())
		} else if pods == nil || len(pods.Items) == 0 {
			info.Status.QueueFailureMessage("Failed to find any OVN master pods")
		}
	}

	var masterPod *v1.Pod
	// ovn-nbctl commands only work on one of the masters, figure out which one
	for i := range pods.Items {
		err = tryCmd(info, &pods.Items[i], ovnShowCmd)
		if err == nil {
			masterPod = &pods.Items[i]
			break
		}
	}

	if masterPod == nil {
		info.Status.QueueFailureMessage(fmt.Sprintf("Failed to exec OVN command in all masters: %s", err))
		return
	}

	info.Status.QueueSuccessMessage(fmt.Sprintf("Gathering OVN data from master pod %q", masterPod.Name))

	for name, command := range ovnCmds {
		logCmdOutput(info, masterPod, command, name, false)
	}
}

func gatherCableDriverResources(info Info, cableDriver string) {
	logPodInfo(info, "cable driver data", gatewayPodLabel, func(info Info, pod *v1.Pod) {
		if cableDriver == libreswan {
			logLibreswanCmds(info, pod)
		}
	})
}

func logLibreswanCmds(info Info, pod *v1.Pod) {
	for name, cmd := range libreswanCmds {
		logCmdOutput(info, pod, cmd, name, true)
	}
}

func execCmdInBash(info Info, pod *v1.Pod, cmd string) (string, string, error) {
	execOptions := resource.ExecOptionsFromPod(pod)
	execConfig := resource.ExecConfig{
		RestConfig: info.RestConfig,
		ClientSet:  info.ClientSet,
	}
	execOptions.Command = []string{"/bin/bash", "-c", cmd}
	return resource.ExecWithOptions(execConfig, execOptions)
}

func logCmdOutput(info Info, pod *v1.Pod, cmd, cmdName string, ignoreError bool) {
	stdOut, _, err := execCmdInBash(info, pod, cmd)
	if err != nil && !ignoreError {
		info.Status.QueueFailureMessage(fmt.Sprintf("Error running %q on pod %q: %v", cmd, pod.Name, err))
		return
	}
	if stdOut != "" {
		// the first line contains the executed command
		stdOut = cmd + "\n" + stdOut
		fileName, err := writeLogToFile(stdOut, pod.Spec.NodeName+"_"+cmdName, info, ".log")
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("Error writing output from command %q on pod %q: %v", cmd, pod.Name, err))
		}
		info.Summary.Resources = append(info.Summary.Resources, ResourceInfo{
			Namespace: pod.Namespace,
			Name:      pod.Spec.NodeName,
			FileName:  fileName,
			Type:      cmdName,
		})
		return
	}
}

func tryCmd(info Info, pod *v1.Pod, cmd string) error {
	_, _, err := execCmdInBash(info, pod, cmd)
	return err
}
