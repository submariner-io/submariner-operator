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

package image

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/names"
)

func ForOperator(imageVersion, repo string, imageOverrideArr []string) (string, error) {
	if imageVersion == "" {
		imageVersion = submariner.DefaultSubmarinerOperatorVersion
	}

	if repo == "" {
		repo = submariner.DefaultRepo
	}

	imageOverrides, err := GetOverrides(imageOverrideArr)
	if err != nil {
		return "", errors.Wrap(err, "error overriding Operator image")
	}

	return images.GetImagePath(repo, imageVersion, names.OperatorImage, names.OperatorComponent, imageOverrides), nil
}

func GetOverrides(imageOverrideArr []string) (map[string]string, error) {
	if len(imageOverrideArr) > 0 {
		imageOverrides := make(map[string]string)

		for _, s := range imageOverrideArr {
			key := strings.Split(s, "=")[0]
			if invalidImageName(key) {
				return nil, fmt.Errorf("invalid image name %s provided. Please choose from %q", key, names.ValidImageNames)
			}

			value := strings.Split(s, "=")[1]
			imageOverrides[key] = value
		}

		return imageOverrides, nil
	}

	return map[string]string{}, nil
}

func invalidImageName(key string) bool {
	for _, name := range names.ValidImageNames {
		if key == name {
			return false
		}
	}

	return true
}
