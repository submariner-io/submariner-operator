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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func gatherPodLogs(podLabelSelector string, info Info) {
	gatherPodLogsByContainer(podLabelSelector, "", info)
}

func gatherPodLogsByContainer(podLabelSelector, container string, info Info) {
	err := func() error {
		pods, err := findPods(info.ClientSet, podLabelSelector)

		if err != nil {
			return err
		}

		info.Status.QueueSuccessMessage(fmt.Sprintf("Found %d pods matching label selector %q", len(pods.Items), podLabelSelector))

		podLogOptions := corev1.PodLogOptions{}
		podLogOptions.Container = container
		for i := range pods.Items {
			outputPodLogs(&pods.Items[i], podLogOptions, info)
		}

		return nil
	}()

	if err != nil {
		info.Status.QueueFailureMessage(fmt.Sprintf("Failed to gather logs for pods matching label selector %q: %s",
			podLabelSelector, err))
	}
}

func outputPodLogs(pod *corev1.Pod, podLogOptions corev1.PodLogOptions, info Info) {
	err := outputPreviousPodLog(pod, podLogOptions, info)
	if err != nil {
		info.Status.QueueFailureMessage(fmt.Sprintf("Error outputting previous log for pod %q: %v", pod.Name, err))
	}

	err = outputCurrentPodLog(pod, podLogOptions, info)
	if err != nil {
		info.Status.QueueFailureMessage(fmt.Sprintf("Error outputting current log for pod %q: %v", pod.Name, err))
	}
}

func writePodLogToFile(logStream io.ReadCloser, info Info, podName, fileExtension string) error {
	logs, err := getLogFromStream(logStream)
	if err != nil {
		return err
	}

	logs = scrubSensitiveData(info, logs)
	err = writeLogToFile(logs, podName, info, fileExtension)
	if err != nil {
		return err
	}

	return nil
}

func getLogFromStream(logStream io.ReadCloser) (string, error) {
	logs := new(bytes.Buffer)
	_, err := io.Copy(logs, logStream)
	if err != nil {
		return "", errors.WithMessage(err, "error copying the log stream")
	}

	return logs.String(), nil
}

func writeLogToFile(data, podName string, info Info, fileExtension string) error {
	fileName := filepath.Join(info.DirName, escapeFileName(info.ClusterName+"_"+podName)+fileExtension)
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
	pods, err := clientSet.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{LabelSelector: byLabelSelector})

	if err != nil {
		return nil, errors.WithMessage(err, "error listing pods")
	}

	return pods, nil
}

func outputPreviousPodLog(pod *corev1.Pod, podLogOptions corev1.PodLogOptions, info Info) error {
	podLogOptions.Previous = true
	logRequest := info.ClientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
	logStream, _ := logRequest.Stream(context.TODO())

	// TODO: Check for error other than "no previous pods found"

	// if no previous pods found, logstream == nil, ignore it
	if logStream != nil {
		info.Status.QueueWarningMessage(fmt.Sprintf("Found logs for previous instances of pod %s", pod.Name))
		err := writePodLogToFile(logStream, info, pod.Name, ".log.prev")
		if err != nil {
			return err
		}
		defer logStream.Close()
	}
	return nil
}

func outputCurrentPodLog(pod *corev1.Pod, podLogOptions corev1.PodLogOptions, info Info) error {
	// Running with Previous = false on the same pod
	podLogOptions.Previous = false
	logRequest := info.ClientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
	logStream, err := logRequest.Stream(context.TODO())
	if err != nil {
		return errors.WithMessage(err, "error opening log stream")
	}
	defer logStream.Close()

	err = writePodLogToFile(logStream, info, pod.Name, ".log")
	if err != nil {
		return err
	}
	return nil
}
