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

	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

type cliReporter struct {
	status *cli.Status
}

func NewCLIReporter() api.Reporter {
	return &cliReporter{status: cli.NewStatus()}
}

func (r *cliReporter) Started(message string, args ...interface{}) {
	r.status.Start(fmt.Sprintf(message, args...))
}

func (r *cliReporter) Succeeded(message string, args ...interface{}) {
	if message != "" {
		r.status.QueueSuccessMessage(fmt.Sprintf(message, args...))
	}
	r.status.End(cli.Success)
}

func (r *cliReporter) Failed(err ...error) {
	if len(err) > 0 {
		r.status.QueueFailureMessage(err[0].Error())
	}
	r.status.End(cli.Failure)
}
