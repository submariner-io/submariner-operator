package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/broker"

	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

		// Create the CRDs we need
		apiext, err := apiextension.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("Creating the clusters CRD\n")
		_, err = apiext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(broker.NewClustersCRD())
		if err != nil && !apierrors.IsAlreadyExists(err) {
			panic(err.Error())
		}
		fmt.Printf("Creating the endpoints CRD\n")
		_, err = apiext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(broker.NewEndpointsCRD())
		if err != nil && !apierrors.IsAlreadyExists(err) {
			panic(err.Error())
		}

		// Create the namespace
		fmt.Printf("Creating the broker namespace\n")
		_, err = clientset.CoreV1().Namespaces().Create(broker.NewBrokerNamespace())
		if err != nil && !apierrors.IsAlreadyExists(err) {
			panic(err.Error())
		}

		// Create the SA we need for the broker
		fmt.Printf("Creating the broker SA\n")
		_, err = clientset.CoreV1().ServiceAccounts("submariner-k8s-broker").Create(broker.NewBrokerSA())
		if err != nil && !apierrors.IsAlreadyExists(err) {
			panic(err.Error())
		}

		// Create the role
		fmt.Printf("Creating the broker role\n")
		_, err = clientset.RbacV1().Roles("submariner-k8s-broker").Create(broker.NewBrokerRole())
		if err != nil && !apierrors.IsAlreadyExists(err) {
			panic(err.Error())
		}

		// Create the role binding
		fmt.Printf("Creating the broker role binding\n")
		_, err = clientset.RbacV1().RoleBindings("submariner-k8s-broker").Create(broker.NewBrokerRoleBinding())
		if err != nil && !apierrors.IsAlreadyExists(err) {
			panic(err.Error())
		}

		// Generate and store a psk in secret
		pskSecret, err := broker.NewBrokerPSKSecret(IPSECPSKBytes)
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("Creating the broker PSK secret\n")
		_, err = clientset.CoreV1().Secrets("submariner-k8s-broker").Create(pskSecret)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			panic(err.Error())
		}

		// List pods
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	},
}
