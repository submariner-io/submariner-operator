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

type Tracker struct {
	Interface
	hasFailures bool
	hasWarnings bool
}

func NewTracker(forReporter Interface) *Tracker {
	return &Tracker{
		Interface: forReporter,
	}
}

func (t *Tracker) Warning(message string, args ...interface{}) {
	t.hasWarnings = true
	t.Interface.Warning(message, args)
}

func (t *Tracker) Failure(message string, args ...interface{}) {
	t.hasFailures = true
	t.Interface.Failure(message, args)
}

func (t *Tracker) Start(message string, args ...interface{}) {
	t.hasWarnings = false
	t.hasFailures = false
	t.Interface.Start(message, args)
}

func (t *Tracker) HasWarnings() bool {
	return t.hasWarnings
}

func (t *Tracker) HasFailures() bool {
	return t.hasFailures
}
