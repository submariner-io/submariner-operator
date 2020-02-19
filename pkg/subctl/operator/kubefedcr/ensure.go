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

package kubefedcr

import (
	"fmt"
	"io"
	"os/exec"

	"k8s.io/client-go/rest"
)

const kubefedResource = `
apiVersion: operator.kubefed.io/v1alpha1
kind: KubeFed
metadata:
  name: kubefed-resource
spec:
  scope: Cluster
`

func Ensure(config *rest.Config, namespace string, kubeConfig string, kubeContext string) error {

	args := []string{}
	if kubeConfig != "" {
		args = append(args, "--kubeconfig", kubeConfig)
	}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	args = append(args, "apply", "-n", namespace, "-f", "-")
	cmd := exec.Command("kubectl", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("error setting up kubectl to deploy KubeFed: %s", err)
	}

	go func() {
		defer stdin.Close()
		_, _ = io.WriteString(stdin, kubefedResource)
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running kubectl to deploy KubeFed: %s\n%s", err, out)
	}

	return nil
}
