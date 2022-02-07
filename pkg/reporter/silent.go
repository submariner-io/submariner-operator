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

type silent struct{}

func Silent() Interface {
	return &silent{}
}

func (s silent) Start(_ string, _ ...interface{}) {
}

func (s silent) End() {
}

func (s silent) Success(_ string, _ ...interface{}) {
}

func (s silent) Failure(_ string, _ ...interface{}) {
}

func (s silent) Warning(_ string, _ ...interface{}) {
}

func (s silent) Error(err error, message string, args ...interface{}) error {
	return HandleError(s, err, message, args...)
}
