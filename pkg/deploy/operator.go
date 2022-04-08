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

package deploy

import (
	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/image"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
)

func Operator(status reporter.Interface, version, repository string, imageOverrideArr []string, debug bool,
	clientProducer client.Producer,
) error {
	operatorImage, err := image.ForOperator(version, repository, imageOverrideArr)
	if err != nil {
		return errors.Wrap(err, "error overriding Operator Image")
	}

	err = submarinerop.Ensure(status, clientProducer, constants.OperatorNamespace, operatorImage, debug)
	if err != nil {
		return errors.Wrap(err, "error deploying Submariner operator")
	}

	return nil
}
