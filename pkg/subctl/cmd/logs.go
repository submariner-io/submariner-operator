package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
)

func getGatewayPodsDetails(clientSet kubernetes.Interface) {
	pods, err := network.FindPod(clientSet, "app=submariner-gateway")

	if err != nil {
		fmt.Println("error getting pods")
	}
	for _, pod := range pods.Items {
		getPodLogs(clientSet, pod.Name)
	}
}

func getPodLogs(clientset kubernetes.Interface, podname string) {
	podLogOpts := corev1.PodLogOptions{}
	logrequest := clientset.CoreV1().Pods(SubmarinerNamespace).GetLogs(podname, &podLogOpts)
	logstream, err := logrequest.Stream()
	if err != nil {
		fmt.Errorf("error opening log stream %s", err)
	}
	defer logstream.Close()

	logs := new(bytes.Buffer)
	_, err = io.Copy(logs, logstream)
	if err != nil {
		fmt.Errorf("error in copy information from logs to buf %s", err)
	}

	dirname := filepath.Join(strings.Split(podname, "-")[1], "logs")
	if _, err := os.Stat(dirname); os.IsNotExist(err) {
		os.MkdirAll(dirname, 0777)
	}

	extension := ".logs"
	logFile := filepath.Join(dirname, podname)
	writeToFile(logFile+extension, logs.String())
}

func writeToFile(filename string, data string) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
	if err != nil {
		fmt.Println("error opening file", err)
	}
	defer f.Close()
	_, err = f.WriteString(data)
	if err != nil {
		fmt.Errorf("error writing to file", err)
	}
}
