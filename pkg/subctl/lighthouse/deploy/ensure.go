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
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	"github.com/kubernetes-sigs/kubefed/pkg/version"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	lighthousedns "github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/dns"
	"github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/install"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefed"
	"github.com/submariner-io/submariner-operator/pkg/versions"
)

const (
	kubefedctlVersionPrefix = "kubefedctl version: version.Info"
)

var (
	serviceDiscovery bool
	imageRepo        string
	imageVersion     string
)

func AddFlags(cmd *cobra.Command, prefix string) {
	cmd.PersistentFlags().BoolVar(&serviceDiscovery, prefix, false,
		"Enable Multi Cluster Service Discovery")
	cmd.PersistentFlags().StringVar(&imageRepo, prefix+"-repo", versions.DefaultSubmarinerRepo,
		"Service Discovery Image repository")
	cmd.PersistentFlags().StringVar(&imageVersion, prefix+"-version", versions.DefaultLighthouseVersion,
		"Service Discovery Image version")
}

func Validate() error {
	if serviceDiscovery {
		// Check we can run kubefedctl, and it’s the right version
		out, err := exec.Command("kubefedctl", "version").CombinedOutput()
		if err != nil {
			return err
		}
		output := string(out)
		if !strings.HasPrefix(output, kubefedctlVersionPrefix) {
			return fmt.Errorf("unable to determine kubefedctl version, please use %s", versions.KubeFedVersion)
		}
		// We need to tweak the output to make it valid JSON again...
		output = output[len(kubefedctlVersionPrefix):]
		output = strings.Replace(output, "Version", "\"gitVersion\"", 1)
		output = strings.Replace(output, "GitCommit", "\"gitCommit\"", 1)
		output = strings.Replace(output, "GitTreeState", "\"gitTreeState\"", 1)
		output = strings.Replace(output, "BuildDate", "\"buildDate\"", 1)
		output = strings.Replace(output, "GoVersion", "\"goVersion\"", 1)
		output = strings.Replace(output, "Compiler", "\"compiler\"", 1)
		output = strings.Replace(output, "Platform", "\"platform\"", 1)
		v := version.Info{}
		err = json.Unmarshal([]byte(output), &v)
		if err != nil {
			return err
		}
		if v.Version != versions.KubeFedVersion {
			return fmt.Errorf("invalid kubefedctl version %s, please use %s", v.Version,
				versions.KubeFedVersion)
		}
	}
	return nil
}

func Ensure(status *cli.Status, config *rest.Config, repo string, version string, isController bool) error {
	repo, version = canonicaliseRepoVersion(repo, version)

	// Ensure DNS
	err := lighthousedns.Ensure(status, config, repo, version)
	if err != nil {
		return fmt.Errorf("error setting DNS up: %s", err)
	}

	// Ensure KubeFed
	err = kubefed.Ensure(status, config, "kubefed-operator",
		"quay.io/openshift/kubefed-operator:"+versions.KubeFedVersion, isController)
	if err != nil {
		return fmt.Errorf("error deploying KubeFed: %s", err)
	}
	image := ""
	// Ensure lighthouse
	if isController {
		image = generateImageName(repo, versions.DefaultLighthouseVersion, version)
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
		repo = versions.DefaultSubmarinerRepo
	}
	if repo[len(repo)-1:] != "/" {
		repo = repo + "/"
	}
	if version == "" {
		version = versions.DefaultLighthouseVersion
	}
	return repo, version
}

func generateImageName(repo string, name string, version string) string {
	return repo + name + ":" + version
}
