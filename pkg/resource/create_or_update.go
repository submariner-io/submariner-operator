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

package resource

import (
	"context"

	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
)

func CreateOrUpdate(ctx context.Context, client resource.Interface, obj runtime.Object) (bool, error) {
	result, err := util.CreateOrUpdate(ctx, client, obj, util.Replace(obj))
	return result == util.OperationResultCreated, err // nolint:wrapcheck // No need to wrap.
}
