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

package images

import (
	"fmt"
	"os"
	"strings"

	apis "github.com/submariner-io/submariner-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func GetImagePath(repo, version, image, component string, imageOverrides map[string]string) string {
	if override, ok := imageOverrides[component]; ok {
		return logIfChanged(repo, version, image, component, override, "Image is overridden")
	}

	if relatedImage, present := os.LookupEnv("RELATED_IMAGE_" + image); present {
		return logIfChanged(repo, version, image, component, relatedImage, "Related image in the environment")
	}

	path := image

	// If the repository is "local" we don't append it on the front of the image,
	// a local repository is used for development, testing and CI when we inject
	// images in the cluster, for example submariner-gateway:local, or submariner-route-agent:local
	if repo != "local" {
		path = fmt.Sprintf("%s/%s", repo, image)
	}

	path = fmt.Sprintf("%s:%s", path, version)

	return logIfChanged(repo, version, image, component, path, "Calculated path")
}

type imageParameters struct {
	repo      string
	version   string
	image     string
	component string
}

var (
	log          = logf.Log.WithName("images")
	loggedImages = make(map[imageParameters]string)
)

func logIfChanged(repo, version, image, component, result, explanation string) string {
	imageParams := imageParameters{
		repo:      repo,
		version:   version,
		image:     image,
		component: component,
	}
	previous, ok := loggedImages[imageParams]

	if !ok || result != previous {
		log.Info("New GetImagePath result", "repo", repo, "version", version, "image", image, "component", component,
			"previous", previous, "result", result, "explanation", explanation)
	}

	loggedImages[imageParams] = result

	return result
}

func GetPullPolicy(version, override string) v1.PullPolicy {
	tags := strings.Split(override, ":")
	if len(tags) > 1 {
		return getPullPolicy(tags[1])
	}

	return getPullPolicy(version)
}

func getPullPolicy(version string) v1.PullPolicy {
	if version == "devel" || version == "local" || strings.HasPrefix(version, "release-") {
		return v1.PullAlways
	}

	return v1.PullIfNotPresent
}

func ParseOperatorImage(operatorImage string) (string, string) {
	var repository string
	var version string

	pathParts := strings.SplitN(operatorImage, "/", 3)
	if len(pathParts) == 1 {
		repository = ""
	} else if len(pathParts) < 3 || !strings.Contains(pathParts[0], ".") &&
		!strings.Contains(pathParts[0], ":") && pathParts[0] != "localhost" {
		repository = pathParts[0]
	} else {
		repository = pathParts[0] + "/" + pathParts[1]
	}

	imageName := strings.Replace(operatorImage, repository, "", 1)
	i := strings.LastIndex(imageName, ":")

	if i == -1 {
		version = apis.DefaultSubmarinerOperatorVersion
	} else {
		version = imageName[i+1:]
	}

	return version, repository
}
