package image

import (
	"fmt"
	"strings"

	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

func Operator(imageVersion, repo string, imageOverrideArr []string) string {
	if imageVersion == "" {
		imageVersion = submariner.DefaultSubmarinerOperatorVersion
	}

	if repo == "" {
		repo = submariner.DefaultRepo
	}

	return images.GetImagePath(repo, imageVersion, names.OperatorImage, names.OperatorComponent, GetOverrides(imageOverrideArr))
}

func GetOverrides(imageOverrideArr []string) map[string]string {
	if len(imageOverrideArr) > 0 {
		imageOverrides := make(map[string]string)
		for _, s := range imageOverrideArr {
			key := strings.Split(s, "=")[0]
			if invalidImageName(key) {
				utils.ExitWithErrorMsg(fmt.Sprintf("Invalid image name %s provided. Please choose from %q", key, names.ValidImageNames))
			}
			value := strings.Split(s, "=")[1]
			imageOverrides[key] = value
		}
		return imageOverrides
	}
	return nil
}

func invalidImageName(key string) bool {
	for _, name := range names.ValidImageNames {
		if key == name {
			return false
		}
	}
	return true
}
