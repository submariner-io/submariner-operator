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

package lighthouse

import (
	"fmt"

	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/install"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefed"
)

const (
	DefaultControllerImageName    = "lighthouse-controller"
	DefaultControllerImageRepo    = "quay.io/submariner"
	DefaultControllerImageVersion = "0.0.1"
)

func Ensure(status *cli.Status, config *rest.Config, repo string, version string, isController bool) error {
	image := ""
	// Ensure KubeFed
	err := kubefed.Ensure(status, config, "kubefed-operator", "quay.io/openshift/kubefed-operator:v0.1.0-rc3")
	if err != nil {
		return fmt.Errorf("error deploying KubeFed: %s", err)
	}
	// Ensure lighthouse
	if isController {
		image = generateImageName(repo, DefaultControllerImageName, version)
	}
	return install.Ensure(status, config, image, isController)
}

func generateImageName(repo string, name string, version string) string {
	if version == "local" || repo == "" {
		return name + ":local"
	}
	if repo[len(repo)-1:] != "/" {
		repo = repo + "/"
	}
	return repo + name + ":" + version
}
