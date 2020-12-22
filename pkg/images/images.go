package images

import (
	"fmt"
	"strings"

	"github.com/submariner-io/submariner-operator/pkg/names"
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
	i := strings.LastIndex(operatorImage, ":")
	var repository string
	var version string
	if i == -1 {
		repository = operatorImage
	} else {
		repository = operatorImage[:i]
		version = operatorImage[i+1:]
	}

	suffix := "/" + names.OperatorImage
	j := strings.LastIndex(repository, suffix)
	if j != -1 {
		repository = repository[:j]
	}
	return version, repository
}
