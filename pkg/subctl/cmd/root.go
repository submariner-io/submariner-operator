package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/spf13/cobra"
)

var (
	kubeConfig string
	rootCmd    = &cobra.Command{
		Use:   "subctl",
		Short: "An installer for Submariner",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", kubeConfigFile(), "absolute path(s) to the kubeconfig file(s)")
}

const (
	DefaultOperatorImage = "quay.io/submariner/submariner-operator:0.0.1"
	OperatorNamespace    = "submariner-operator"
)

var operatorImage string

func kubeConfigFile() string {
	var kubeconfig string
	if kubeconfig = os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}
	if home := homeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	} else {
		return ""
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func panicOnError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func getClients() (dynamic.Interface, kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, nil, err
	}
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	return dynClient, clientSet, nil
}

func getRestConfig() (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", kubeConfig)
}

func handleNodeLabels() error {
	_, clientset, err := getClients()
	panicOnError(err)
	// List Submariner-labeled nodes
	const submarinerGatewayLabel = "submariner.io/gateway"
	const trueLabel = "true"
	selector := labels.SelectorFromSet(labels.Set(map[string]string{submarinerGatewayLabel: trueLabel}))
	labeledNodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	if len(labeledNodes.Items) > 0 {
		fmt.Printf("* There are %d labeled nodes in the cluster:\n", len(labeledNodes.Items))
		for _, node := range labeledNodes.Items {
			fmt.Printf("  - %s\n", node.GetName())
		}
	} else {
		answer, err := askForGatewayNode(clientset)
		if err != nil {
			return err
		}
		err = addLabelsToNode(clientset, answer.Node, map[string]string{submarinerGatewayLabel: trueLabel})
		panicOnError(err)

	}
	return nil
}

func askForGatewayNode(clientset kubernetes.Interface) (struct{ Node string }, error) {
	// List all nodes and select one
	allNodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return struct{ Node string }{}, err
	}
	fmt.Printf("There are %d nodes in the cluster\n", len(allNodes.Items))
	allNodeNames := []string{}
	for _, node := range allNodes.Items {
		allNodeNames = append(allNodeNames, node.GetName())
	}
	var qs = []*survey.Question{
		{
			Name: "node",
			Prompt: &survey.Select{
				Message: "Which node should be used as the gateway?",
				Options: allNodeNames},
		},
	}
	answers := struct {
		Node string
	}{}
	err = survey.Ask(qs, &answers)
	if err != nil {
		return struct{ Node string }{}, err
	}
	return answers, nil
}

// this function was sourced from:
// https://github.com/kubernetes/kubernetes/blob/a3ccea9d8743f2ff82e41b6c2af6dc2c41dc7b10/test/utils/density_utils.go#L36
func addLabelsToNode(c kubernetes.Interface, nodeName string, labels map[string]string) error {
	const retries = 5
	const retryInterval = 500 * time.Millisecond

	var tokens []string
	for k, v := range labels {
		tokens = append(tokens, fmt.Sprintf("\"%s\":\"%s\"", k, v))
	}

	labelString := "{" + strings.Join(tokens, ",") + "}"
	patch := fmt.Sprintf(`{"metadata":{"labels":%v}}`, labelString)

	var err error

	// retry is necessary because nodes get updated every 10 seconds, and a patch can happen
	// in the middle of an update
	for attempt := 0; attempt < retries; attempt++ {
		_, err = c.CoreV1().Nodes().Patch(nodeName, types.MergePatchType, []byte(patch))
		if err != nil {
			if !errors.IsConflict(err) {
				return err
			}
		} else {
			break
		}
		time.Sleep(retryInterval)
	}
	return err
}
