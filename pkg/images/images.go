package images

import (
	"fmt"
)

func GetImagePath(repo, version, component string) string {
	var path string

	// If the repository is "local" we don't append it on the front of the image,
	// a local repository is used for development, testing and CI when we inject
	// images in the cluster, for example submariner:local, or submariner-route-agent:local
	if repo == "local" {
		path = component
	} else {
		path = fmt.Sprintf("%s/%s", repo, component)
	}

	path = fmt.Sprintf("%s:%s", path, version)
	return path
}
