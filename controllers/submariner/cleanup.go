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

package submariner

import (
	"context"

	"github.com/submariner-io/admiral/pkg/finalizer"
	operatorv1alpha1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/resource"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *Reconciler) runComponentCleanup(ctx context.Context, instance *operatorv1alpha1.Submariner) (reconcile.Result, error) {
	err := finalizer.Remove(ctx, resource.ForControllerClient(r.config.Client, instance.Namespace, &operatorv1alpha1.Submariner{}),
		instance, SubmarinerFinalizer)
	return reconcile.Result{}, err // nolint:wrapcheck // No need to wrap
}
