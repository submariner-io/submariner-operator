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

package exit

import (
	"fmt"
	"os"

	"github.com/submariner-io/submariner-operator/pkg/version"
)

// OnError exits in case of error.
func OnError(err error) {
	if err != nil {
		printVersion()
		os.Exit(1)
	}
}

// WithMessage will print the message and quit the program with an error code.
func WithMessage(message string) {
	fmt.Fprintln(os.Stderr, message)
	printVersion()
	os.Exit(1)
}

// OnErrorWithMessage will print the message and quit the program with an error code.
func OnErrorWithMessage(err error, message string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", message, err)
		fmt.Fprintln(os.Stderr, "")
		printVersion()
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Fprintln(os.Stderr, "")
	version.PrintSubctlVersion(os.Stderr)
	fmt.Fprintln(os.Stderr, "")
}
