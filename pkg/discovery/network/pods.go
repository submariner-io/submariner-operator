package network

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func findPodCommandParameter(clientSet kubernetes.Interface, labelSelector, parameter string) (string, error) {

	pod, err := findPod(clientSet, labelSelector)

	if err != nil || pod == nil {
		return "", err
	}
	for _, container := range pod.Spec.Containers {
		for _, arg := range container.Command {
			if strings.HasPrefix(arg, parameter) {
				return strings.Split(arg, "=")[1], nil
			}
			// Handling the case where the command is in the form of /bin/sh -c exec ....
			if strings.Contains(arg, " ") {
				for _, subArg := range strings.Split(arg, " ") {
					if strings.HasPrefix(subArg, parameter) {
						return strings.Split(subArg, "=")[1], nil
					}
				}
			}
		}
	}
	return "", nil
}

func findPod(clientSet kubernetes.Interface, labelSelector string) (*v1.Pod, error) {
	pods, err := clientSet.CoreV1().Pods("").List(v1meta.ListOptions{
		LabelSelector: labelSelector,
		Limit:         1,
	})

	if err != nil || len(pods.Items) == 0 {
		return nil, err
	}
	return &pods.Items[0], nil
}
