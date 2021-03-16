package gather

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type GatherParams struct {
	ClusterName string
	DirName     string
	PodName     string
}

const submarinerNamespace = "submariner-operator"

func getPodLogs(clientset kubernetes.Interface, params GatherParams) error {
	logrequest := clientset.CoreV1().Pods(submarinerNamespace).GetLogs(params.PodName, &corev1.PodLogOptions{})

	logs, err := processLogStream(logrequest)
	if err != nil {
		return err
	}

	err = writeToFile(logs, params)
	if err != nil {
		return err
	}
	return nil
}

func processLogStream(logrequest *rest.Request) (string, error) {
	logstream, err := logrequest.Stream()
	if err != nil {
		return "", errors.WithMessage(err, "error opening log stream")
	}
	defer logstream.Close()

	logs := new(bytes.Buffer)
	_, err = io.Copy(logs, logstream)
	if err != nil {
		return "", errors.WithMessage(err, "error converting log stream to string")
	}
	return logs.String(), nil
}

func writeToFile(data string, params GatherParams) error {
	filename := filepath.Join(params.DirName, params.ClusterName+"-"+params.PodName+".log")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
	if err != nil {
		return errors.WithMessagef(err, "error opening file %s", filename)
	}
	defer f.Close()
	_, err = f.WriteString(data)
	if err != nil {
		return errors.WithMessagef(err, "error writing to file %s", filename)
	}
	return nil
}

func findPods(clientSet kubernetes.Interface, options v1meta.ListOptions) (*corev1.PodList, error) {
	pods, err := clientSet.CoreV1().Pods("").List(options)

	if err != nil {
		return nil, errors.WithMessagef(err, "error listing Pods by label selector %q", options.LabelSelector)
	}

	if len(pods.Items) == 0 {
		return nil, errors.New("No pods found matching the Labelselector")
	}

	return pods, nil
}
