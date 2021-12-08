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
package version

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	Version = "devel"
)

const (
	minK8sMajor = 1 // We need K8s 1.17 for endpoint slices
	minK8sMinor = 17
)

// PrintSubctlVersion will print the version subctl was compiled under.
func PrintSubctlVersion(w io.Writer) {
	fmt.Fprintf(w, "subctl version: %s\n", Version)
}

func CheckRequirements(config *rest.Config) (string, []string, error) {
	failedRequirements := []string{}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", failedRequirements, errors.WithMessage(err, "error creating API server client")
	}
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", failedRequirements, errors.WithMessage(err, "error obtaining API server version")
	}
	major, err := strconv.Atoi(serverVersion.Major)
	if err != nil {
		return serverVersion.String(), failedRequirements,
			errors.WithMessagef(err, "error parsing API server major version %v", serverVersion.Major)
	}
	var minor int
	if strings.HasSuffix(serverVersion.Minor, "+") {
		minor, err = strconv.Atoi(serverVersion.Minor[0 : len(serverVersion.Minor)-1])
	} else {
		minor, err = strconv.Atoi(serverVersion.Minor)
	}
	if err != nil {
		return serverVersion.String(), failedRequirements,
			errors.WithMessagef(err, "error parsing API server minor version %v", serverVersion.Minor)
	}
	if major < minK8sMajor || (major == minK8sMajor && minor < minK8sMinor) {
		failedRequirements = append(failedRequirements,
			fmt.Sprintf("Submariner requires Kubernetes %d.%d; your cluster is running %s.%s",
				minK8sMajor, minK8sMinor, serverVersion.Major, serverVersion.Minor))
	}
	return serverVersion.String(), failedRequirements, nil
}
