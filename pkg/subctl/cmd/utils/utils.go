/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	subOperatorClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	"github.com/submariner-io/submariner-operator/pkg/version"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1opts "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// PanicOnError will print the subctl version and then panic in case of an actual error.
func PanicOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "")
		version.PrintSubctlVersion(os.Stderr)
		fmt.Fprintln(os.Stderr, "")
		panic(err.Error())
	}
}

// ExitOnError will print your error nicely and exit in case of error.
func ExitOnError(message string, err error) {
	if err != nil {
		ExitWithErrorMsg(fmt.Sprintf("%s: %s", message, err))
	}
}

// ExitWithErrorMsg will print the message and quit the program with an error code.
func ExitWithErrorMsg(message string) {
	fmt.Fprintln(os.Stderr, message)
	fmt.Fprintln(os.Stderr, "")
	version.PrintSubctlVersion(os.Stderr)
	fmt.Fprintln(os.Stderr, "")
	os.Exit(1)
}

// ExpectFlag exits with an error if the flag value is empty.
func ExpectFlag(flag, value string) {
	if value == "" {
		ExitWithErrorMsg(fmt.Sprintf("You must specify the %v flag", flag))
	}
}

func GetSubmarinerResourceWithError(config *rest.Config) (*v1alpha1.Submariner, error) {
	submarinerClient, err := subOperatorClientset.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "error creating clientset")
	}

	// TODO skitt namespace constant
	submariner, err := submarinerClient.SubmarinerV1alpha1().Submariners("submariner-operator").
		Get(context.TODO(), submarinercr.SubmarinerName, v1opts.GetOptions{})
	if err != nil {
		return nil, errors.WithMessagef(err, "error retrieving Submariner object %s", submarinercr.SubmarinerName)
	}

	return submariner, nil
}

func GetSubmarinerResource(config *rest.Config) *v1alpha1.Submariner {
	submariner, err := GetSubmarinerResourceWithError(config)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		ExitOnError("Error obtaining the Submariner resource", err)
	}

	return submariner
}
