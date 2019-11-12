package cmd

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

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
