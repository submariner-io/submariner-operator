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

package cluster

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"
)

const replaceChar = "0"

func IsValidID(clusterID string) error {
	if errs := validation.IsDNS1123Label(clusterID); len(errs) > 0 {
		return errors.Errorf("%s is not a valid ClusterID %v", clusterID, errs)
	}

	return nil
}

func SanitizeClusterID(clusterID string) string {
	var result string
	inputLen := len(clusterID)

	if inputLen > 0 {
		regDNS1123 := regexp.MustCompile("[^a-z0-9-]+")
		result = strings.ToLower(clusterID)
		result = regDNS1123.ReplaceAllString(result, "-")

		if result[0] == '-' {
			result = replaceChar + result[1:]
		}

		if result[inputLen-1:] == "-" {
			result = result[:inputLen-1] + replaceChar
		}
	}

	return result
}
