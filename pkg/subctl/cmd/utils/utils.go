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

package utils

import (
	"fmt"
	"os"

	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/version"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// PanicOnError will print the subctl version and then panic in case of an actual error
func PanicOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "")
		version.PrintSubctlVersion(os.Stderr)
		fmt.Fprintln(os.Stderr, "")
		panic(err.Error())
	}
}

// ExitOnError will print your error nicely and exit in case of error
func ExitOnError(message string, err error) {
	if err != nil {
		ExitWithErrorMsg(fmt.Sprintf("%s: %s", message, err))
	}
}

// ExitWithErrorMsg will print the message and quit the program with an error code
func ExitWithErrorMsg(message string) {
	fmt.Fprintln(os.Stderr, message)
	fmt.Fprintln(os.Stderr, "")
	version.PrintSubctlVersion(os.Stderr)
	fmt.Fprintln(os.Stderr, "")
	os.Exit(1)
}

// GetRestConfig returns a rest.Config to use when communicating with K8s
func GetRestConfig(kubeConfigPath, kubeContext string) (*rest.Config, error) {
	return GetClientConfig(kubeConfigPath, kubeContext).ClientConfig()
}

// GetClientConfig returns a clientcmd.ClientConfig to use when communicating with K8s
func GetClientConfig(kubeConfigPath, kubeContext string) clientcmd.ClientConfig {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = kubeConfigPath

	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	if kubeContext != "" {
		overrides.CurrentContext = kubeContext
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
}
