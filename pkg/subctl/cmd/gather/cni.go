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

var ipCmds = map[string]string{
	"ip-routes":         "ip route show",
	"ip-rules":          "ip rule list",
	"ip-rules-table150": "ip rule show table 150",
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

const ovnShowCmd = "ovn-nbctl show"

var ovnCmds = map[string]string{
	"ovn_show":             ovnShowCmd,
	"ovn_logical_routers":  "ovn-nbctl list Logical_Router",
	"ovn_lrps":             "ovn-nbctl list Logical_Router_Port",
	"ovn_logical_switches": "ovn-nbctl list Logical_Switch",
	"ovn_lsps":             "ovn-nbctl list Logical_Switch_Port",
	"ovn_routes":           "ovn-nbctl list Logical_Router_Static_Route",
	"ovn_policies":         "ovn-nbctl list Logical_Router_Policy",
	"ovn_acls":             "ovn-nbctl list ACL",
}

var networkPluginCNIType = map[string]string{
	"generic":       typeIPTables,
	"canal-flannel": typeIPTables,
	"weave-net":     typeIPTables,
	"OpenShiftSDN":  typeIPTables,
	"OVNKubernetes": typeOvn,
	"unknown":       typeUnknown,
}

func CNIResources(info Info, networkPlugin string) {
	podLabelSelector := routeagentPodLabel
	err := func() error {
		pods, err := findPods(info.ClientSet, podLabelSelector)

		if err != nil {
			return err
		}

		info.Status.QueueSuccessMessage(fmt.Sprintf("Gathering CNI data from %d pods matching label selector %q",
			len(pods.Items), podLabelSelector))

		for i := range pods.Items {
			pod := &pods.Items[i]
			logIPCmds(info, pod)
			switch networkPluginCNIType[networkPlugin] {
			case typeIPTables:
				logIPTablesCmds(info, pod)
			case typeOvn:
				info.Status.QueueWarningMessage("OVN CNI not supported yet")
			case typeUnknown:
				info.Status.QueueFailureMessage("Unsupported CNI Type")
			}
		}
		return nil
	}()
	if err != nil {
		info.Status.QueueFailureMessage(fmt.Sprintf("Failed to gather CNI data from pods matching label selector %q: %s",
			podLabelSelector, err))
	}
	logCNIGatewayNodeResources(info)
}

func logCNIGatewayNodeResources(info Info) {
	podLabelSelector := gatewayPodLabel
	err := func() error {
		pods, err := findPods(info.ClientSet, podLabelSelector)

		if err != nil {
			return err
		}

		info.Status.QueueSuccessMessage(fmt.Sprintf("Gathering CNI data from %d pods matching label selector %q",
			len(pods.Items), podLabelSelector))

		for i := range pods.Items {
			pod := &pods.Items[i]
			logIPGatewayCmds(info, pod)
		}
		return nil
	}()
	if err != nil {
		info.Status.QueueFailureMessage(fmt.Sprintf("Failed to gather CNI data from pods matching label selector %q: %s",
			podLabelSelector, err))
	}
}

func logIPCmds(info Info, pod *v1.Pod) {
	for name, cmd := range ipCmds {
		logCmdOutput(info, pod, cmd, name)
	}
}

func logIPGatewayCmds(info Info, pod *v1.Pod) {
	for name, cmd := range ipGatewayCmds {
		logCmdOutput(info, pod, cmd, name)
	}
}

func logIPTablesCmds(info Info, pod *v1.Pod) {
	for name, cmd := range ipTablesCmds {
		logCmdOutput(info, pod, cmd, name)
	}
}

func OVNResources(info Info, networkPlugin string) {
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
		logCmdOutput(info, masterPod, command, name)
	}
}

func CableDriverResources(info Info, cableDriver string) {
	podLabelSelector := gatewayPodLabel
	err := func() error {
		pods, err := findPods(info.ClientSet, podLabelSelector)

		if err != nil {
			return err
		}

		info.Status.QueueSuccessMessage(fmt.Sprintf("Gathering cable driver data from %d pods matching label selector %q",
			len(pods.Items), podLabelSelector))

		for i := range pods.Items {
			pod := &pods.Items[i]
			if cableDriver == libreswan {
				logLibreswanCmds(info, pod)
			}
		}
		return nil
	}()
	if err != nil {
		info.Status.QueueFailureMessage(fmt.Sprintf("Failed to gather CNI data from pods matching label selector %q: %s",
			podLabelSelector, err))
	}
}

func logLibreswanCmds(info Info, pod *v1.Pod) {
	for name, cmd := range libreswanCmds {
		logCmdOutput(info, pod, cmd, name)
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

func logCmdOutput(info Info, pod *v1.Pod, cmd, cmdName string) {
	stdOut, _, err := execCmdInBash(info, pod, cmd)
	if err != nil {
		info.Status.QueueFailureMessage(fmt.Sprintf("Error running %q on pod %q: %v", cmd, pod.Name, err))
		return
	}
	if stdOut != "" {
		// the first line contains the executed command
		stdOut = cmd + "\n" + stdOut
		err := writeLogToFile(stdOut, pod.Spec.NodeName+"_"+cmdName, info)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("Error writing output from command %q on pod %q: %v", cmd, pod.Name, err))
		}
		return
	}
}

func tryCmd(info Info, pod *v1.Pod, cmd string) error {
	_, _, err := execCmdInBash(info, pod, cmd)
	return err
}
