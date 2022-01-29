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

package reporter

import "fmt"

type stdout struct{}

func Stdout() Interface {
	return &stdout{}
}

func (s stdout) Start(message string, args ...interface{}) {
	s.Success(message, args...)
}

func (s stdout) End() {
}

func (s stdout) Success(message string, args ...interface{}) {
	fmt.Printf(message+"\n", args...)
}

func (s stdout) Failure(message string, args ...interface{}) {
	fmt.Printf("ERROR: "+message+"\n", args...)
}

func (s stdout) Warning(message string, args ...interface{}) {
	fmt.Printf("WARNING: "+message+"\n", args...)
}

func (s stdout) Error(err error, message string, args ...interface{}) error {
	return HandleError(s, err, message, args...)
}
