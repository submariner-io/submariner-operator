/*
© 2021 Red Hat, Inc. and others

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
package images

import (
	"fmt"
	"strings"

	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/versions"

	v1 "k8s.io/api/core/v1"
)

func GetImagePath(repo, version, component string, imageOverrides map[string]string) string {
	var path string

	if override, ok := imageOverrides[component]; ok {
		return override
	}

	// If the repository is "local" we don't append it on the front of the image,
	// a local repository is used for development, testing and CI when we inject
	// images in the cluster, for example submariner:local, or submariner-route-agent:local
	if repo == "local" {
		path = component
	} else {
		path = fmt.Sprintf("%s/%s%s%s", repo, names.ImagePrefix, component, names.ImagePostfix)
	}

	path = fmt.Sprintf("%s:%s", path, version)
	return path
}

func GetPullPolicy(version string) v1.PullPolicy {
	if version == "devel" {
		return v1.PullAlways
	} else {
		return v1.PullIfNotPresent
	}
}

func ParseOperatorImage(operatorImage string) (string, string) {
	var repository string
	var version string

	pathParts := strings.SplitN(operatorImage, "/", 3)
	if len(pathParts) == 1 {
		repository = ""
	} else if len(pathParts) < 3 || (!strings.Contains(pathParts[0], ".") &&
		!strings.Contains(pathParts[0], ":") && pathParts[0] != "localhost") {
		repository = pathParts[0]
	} else {
		repository = pathParts[0] + "/" + pathParts[1]
	}

	imageName := strings.Replace(operatorImage, repository, "", 1)
	i := strings.LastIndex(imageName, ":")
	if i == -1 {
		version = versions.DefaultSubmarinerOperatorVersion
	} else {
		version = imageName[i+1:]
	}

	return version, repository
}
