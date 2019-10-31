package cmd

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
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

func getClients() (error, dynamic.Interface, kubernetes.Interface) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return err, nil, nil
	}
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err, nil, nil
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err, nil, nil
	}
	return nil, dynClient, clientSet
}
