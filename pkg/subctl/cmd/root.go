package cmd

import (
	"os"
	"path/filepath"

	"fmt"

	"github.com/AlecAivazis/survey/v2"
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
	// TODO Read from the KUBECONFIG env var
	if home := homeDir(); home != "" {
		rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path(s) to the kubeconfig file(s)")
	} else {
		rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", "", "absolute path(s) to the kubeconfig file(s)")
	}
	rootCmd.PersistentFlags().StringVarP(&operatorImage, "image", "i", DefaultOperatorImage,
		"the operator image you wish to use")
}

const (
	DefaultOperatorImage = "quay.io/submariner/submariner-operator:0.0.1"
	OperatorNamespace    = "submariner-operator"
)

var operatorImage string

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
	selector := labels.SelectorFromSet(labels.Set(map[string]string{"submariner.io/gateway": "true"}))
	labeledNodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	fmt.Printf("There are %d labeled nodes in the cluster\n", len(labeledNodes.Items))
	for _, node := range labeledNodes.Items {
		for _, label := range node.GetLabels() {
			fmt.Printf("Node %s, label %s\n", node.GetName(), label)
		}
	}
	if len(labeledNodes.Items) == 0 {
		// List all nodes and select one
		allNodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return err
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
			return err
		}

		// TODO label the node
	}
	return nil
}
