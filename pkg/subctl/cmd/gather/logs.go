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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func gatherPodLogs(podLabelSelector string, info *Info) error {
	pods, err := findPods(info.ClientSet, podLabelSelector)

	if err != nil {
		return err
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		err := podLogsToFile(pod, info)
		if err != nil {
			return errors.WithMessagef(err, "error getting logs for pod %q", pod.Name)
		}
	}

	return nil
}

func podLogsToFile(pod *corev1.Pod, info *Info) error {
	logRequest := info.ClientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})

	logs, err := processLogStream(logRequest)
	if err != nil {
		return err
	}

	err = writeLogToFile(logs, pod.Name, info)
	if err != nil {
		return err
	}

	return nil
}

func processLogStream(logrequest *rest.Request) (string, error) {
	logStream, err := logrequest.Stream()
	if err != nil {
		return "", errors.WithMessage(err, "error opening log stream")
	}
	defer logStream.Close()

	logs := new(bytes.Buffer)
	_, err = io.Copy(logs, logStream)
	if err != nil {
		return "", errors.WithMessage(err, "error copying the log stream")
	}

	return logs.String(), nil
}

func writeLogToFile(data, podName string, info *Info) error {
	fileName := filepath.Join(info.DirName, info.ClusterName+"_"+podName+".log")
	f, err := os.Create(fileName)
	if err != nil {
		return errors.WithMessagef(err, "error opening file %s", fileName)
	}
	defer f.Close()

	_, err = f.WriteString(data)
	if err != nil {
		return errors.WithMessagef(err, "error writing to file %s", fileName)
	}

	return nil
}

func findPods(clientSet kubernetes.Interface, byLabelSelector string) (*corev1.PodList, error) {
	pods, err := clientSet.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: byLabelSelector})

	if err != nil {
		return nil, errors.WithMessage(err, "error listing pods")
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods found matching label selector %q", byLabelSelector)
	}

	return pods, nil
}
