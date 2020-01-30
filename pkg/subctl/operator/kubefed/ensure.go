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

package kubefed

import (
	"fmt"

	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedcr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedop"
)

func Ensure(status *cli.Status, config *rest.Config, operatorNamespace string, operatorImage string) error {

	err := kubefedop.Ensure(status, config, "kubefed-operator", "quay.io/openshift/kubefed-operator:v0.1.0-rc3")
	if err != nil {
		return fmt.Errorf("error deploying KubeFed: %s", err)
	}
	err = kubefedcr.Ensure(config, "kubefed-operator")
	if err != nil {
		return fmt.Errorf("error deploying KubeFed: %s", err)
	}

	return nil
}
