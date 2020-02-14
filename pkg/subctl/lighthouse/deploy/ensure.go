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
	lighthousedns "github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/dns"
	"github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/install"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefed"
)

const (
	DefaultControllerImageName    = "lighthouse-controller"
	DefaultControllerImageRepo    = "quay.io/submariner"
	DefaultControllerImageVersion = "0.1.0"
)

func Ensure(status *cli.Status, config *rest.Config, repo string, version string, isController bool) error {
	repo, version = canonicaliseRepoVersion(repo, version)

	// Ensure DNS
	err := lighthousedns.Ensure(status, config, repo, version)
	if err != nil {
		return fmt.Errorf("error setting DNS up: %s", err)
	}

	// Ensure KubeFed
	err = kubefed.Ensure(status, config, "kubefed-operator", "quay.io/openshift/kubefed-operator:v0.1.0-rc3", isController)
	if err != nil {
		return fmt.Errorf("error deploying KubeFed: %s", err)
	}
	image := ""
	// Ensure lighthouse
	if isController {
		image = generateImageName(repo, DefaultControllerImageName, version)
	}
	return install.Ensure(status, config, image, isController)
}

// canonicaliseRepoVersion returns the canonical repo and version for the given
// repo and version. If the provided version is local, this enforces a local
// image (no repo). Otherwise, empty values are replaced with defaults.
func canonicaliseRepoVersion(repo string, version string) (string, string) {
	if version == "local" {
		return "", version
	}
	if repo == "" {
		repo = DefaultControllerImageRepo
	}
	if repo[len(repo)-1:] != "/" {
		repo = repo + "/"
	}
	if version == "" {
		version = DefaultControllerImageVersion
	}
	return repo, version
}

func generateImageName(repo string, name string, version string) string {
	return repo + name + ":" + version
}
