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

	"github.com/pkg/errors"
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

var networkPluginCNIType = map[string]string{
	"generic":       typeIPTables,
	"canal-flannel": typeIPTables,
	"weave-net":     typeIPTables,
	"OpenShiftSDN":  typeIPTables,
	"OVNKubernetes": typeOvn,
	"unknown":       typeUnknown,
}

func CNIResources(info Info) {
	podLabelSelector := RouteagentPodLabel
	err := func() error {
		pods, err := findPods(info.ClientSet, podLabelSelector)

		if err != nil {
			return err
		}

		info.Status.QueueSuccessMessage(fmt.Sprintf("Found %d pods matching label selector %q", len(pods.Items), podLabelSelector))

		for i := range pods.Items {
			pod := &pods.Items[i]
			logIPCmds(info, pod)
			switch networkPluginCNIType[info.Submariner.Status.NetworkPlugin] {
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
}

func logIPCmds(info Info, pod *v1.Pod) {
	for name, cmd := range ipCmds {
		err := logCmdOutput(info, pod, cmd, name)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("%q", err))
		}
	}
}

func logIPTablesCmds(info Info, pod *v1.Pod) {
	for name, cmd := range ipTablesCmds {
		err := logCmdOutput(info, pod, cmd, name)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("%q", err))
		}
	}
}

func CableDriverResources(info Info) {
	podLabelSelector := gatewayPodLabel
	err := func() error {
		pods, err := findPods(info.ClientSet, podLabelSelector)

		if err != nil {
			return err
		}

		info.Status.QueueSuccessMessage(fmt.Sprintf("Found %d pods matching label selector %q", len(pods.Items), podLabelSelector))

		for i := range pods.Items {
			pod := &pods.Items[i]
			if info.Submariner.Spec.CableDriver == libreswan {
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
		err := logCmdOutput(info, pod, cmd, name)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("%q", err))
		}
	}
}

func logCmdOutput(info Info, pod *v1.Pod, cmd, cmdName string) error {
	execOptions := resource.ExecOptionsFromPod(pod)
	execConfig := resource.ExecConfig{
		RestConfig: info.RestConfig,
		ClientSet:  info.ClientSet,
	}
	execOptions.Command = []string{"/bin/bash", "-c", cmd}
	stdOut, _, err := resource.ExecWithOptions(execConfig, execOptions)
	if err != nil {
		return errors.WithMessagef(err, "error running %q on pod %s : %q", cmd, pod.Name, err)
	}
	if stdOut != "" {
		// the first line contains the executed command
		stdOut = cmd + "\n" + stdOut
		return writeLogToFile(stdOut, pod.Spec.NodeName+"_"+cmdName, info)
	}
	return nil
}
