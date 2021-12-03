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
package utils

import (
	"fmt"
	"os"

	"github.com/submariner-io/submariner-operator/pkg/version"
)

// ExitOnError will print your error nicely and exit in case of error
func ExitOnError(message string, err error) {
	if err != nil {
		ExitWithErrorMsg(fmt.Sprintf("%s: %s", message, err))
	}
}

// ExitWithErrorMsg will print the message and quit the program with an error code
func ExitWithErrorMsg(message string) {
	fmt.Fprintln(os.Stderr, message)
	fmt.Fprintln(os.Stderr, "")
	version.PrintSubctlVersion(os.Stderr)
	fmt.Fprintln(os.Stderr, "")
	os.Exit(1)
}
