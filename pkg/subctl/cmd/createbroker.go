package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	createCmd.AddCommand(createBrokerCmd)
}

const IPSECPSKBytes = 48 // using base64 this results on a 64 character password

var createBrokerCmd = &cobra.Command{
	Use:   "broker",
	Short: "set the broker up",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			panic(err.Error())
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		// List Submariner-labeled nodes
		selector := labels.SelectorFromSet(labels.Set(map[string]string{"submariner.io/gateway": "true"}))
		labeledNodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d labeled nodes in the cluster\n", len(labeledNodes.Items))
		for _, node := range labeledNodes.Items {
			for _, label := range node.GetLabels() {
				fmt.Printf("Node %s, label %s\n", node.GetName(), label)
			}
		}

		fmt.Printf("* Deploying broker\n")
		if err = broker.Ensure(config, IPSECPSKBytes); err != nil {
			panic(err)
		}

		fmt.Printf("* Deploying the submariner operator\n")
		if err := install.Ensure(config, OperatorNamespace, operatorImage); err != nil {
			panic(err)
		}

		// List pods
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	},
}
