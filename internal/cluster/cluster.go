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

const rfc1123Compliant = "0"

func IsValidID(clusterID string) error {
	if errs := validation.IsDNS1123Label(clusterID); len(errs) > 0 {
		return errors.Errorf("%s is not a valid ClusterID %v", clusterID, errs)
	}

	return nil
}

func SanitizeID(clusterID string) string {
	if clusterID == "" {
		return ""
	}

	regDNS1123 := regexp.MustCompile("[^a-z0-9-]+")
	result := regDNS1123.ReplaceAllString(strings.ToLower(clusterID), "-")

	if result[0] == '-' {
		result = rfc1123Compliant + result[1:]
	}

	resultLen := len(result)
	if result[resultLen-1] == '-' {
		result = result[:resultLen-1] + rfc1123Compliant
	}

	return result
}
