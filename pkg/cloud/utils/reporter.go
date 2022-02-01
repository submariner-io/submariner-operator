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
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
)

type statusReporter struct {
	status reporter.Interface
}

func NewStatusReporter() api.Reporter {
	return &statusReporter{status: cli.NewReporter()}
}

func (s *statusReporter) Started(message string, args ...interface{}) {
	s.status.Start(message, args...)
}

func (s *statusReporter) Succeeded(message string, args ...interface{}) {
	s.status.Success(message, args...)
	s.status.End()
}

func (s *statusReporter) Failed(err ...error) {
	if len(err) > 0 {
		s.status.Failure(err[0].Error())
	}

	s.status.End()
}
