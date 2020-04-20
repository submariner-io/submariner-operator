/*
© 2020 Red Hat, Inc. and others.

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

package crds

import (
	"fmt"
	"os/exec"
)

//go:generate go run generators/yamls2go.go

// Copied over from operator/install/crds/ensure.go

//Ensure functions updates or installs the multiclusterservives CRDs in the cluster
func Ensure(kubeConfig string, kubeContext string) (bool, error) {
	args := []string{"enable", "MulticlusterService"}
	if kubeConfig != "" {
		args = append(args, "--kubeconfig", kubeConfig)
	}
	if kubeContext != "" {
		args = append(args, "--host-cluster-context", kubeContext)
	}
	args = append(args, "--kubefed-namespace", "kubefed-operator")
	out, err := exec.Command("kubefedctl", args...).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("error federating MulticlusterService CRD: %s\n%s", err, out)
	}
	return true, nil
}
