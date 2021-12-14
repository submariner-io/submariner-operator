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
package cmd

import (
	"bytes"
	"os"

	"github.com/pkg/errors"
)

func CompareFiles(file1, file2 string) (bool, error) {
	first, err := os.ReadFile(file1)
	if err != nil {
		return false, errors.Wrapf(err, "error reading file %q", file1)
	}

	second, err := os.ReadFile(file2)
	if err != nil {
		return false, errors.Wrapf(err, "error reading file %q", file2)
	}

	return bytes.Equal(first, second), nil
}
