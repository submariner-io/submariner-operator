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

package gateway

import (
	"context"
	goerrors "errors"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner/deploy/crds"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Ensure ensures that the required resources are deployed on the target system.
// The resources handled here are the gateway CRDs: Cluster and Endpoint.
func Ensure(ctx context.Context, crdUpdater crd.Updater) error {
	var errs []error

	for _, ref := range []struct {
		name string
		crd  []byte
	}{
		{"Cluster", crds.ClustersCRD},
		{"Endpoint", crds.EndpointsCRD},
		{"Gateway", crds.GatewaysCRD},
		{"ClusterGlobalEgressIP", crds.ClusterGlobalEgressIPsCRD},
		{"GlobalEgressIP", crds.GlobalEgressIPsCRD},
		{"GlobalIngressIP", crds.GlobalIngressIPsCRD},
		{"GatewayRoute", crds.GatewayRoutesCRD},
		{"NonGatewayRoute", crds.NonGatewayRoutesCRD},
		{"RouteAgent", crds.RouteAgentsCRD},
	} {
		if _, err := crdUpdater.CreateOrUpdateFromBytes(ctx, ref.crd); err != nil && !apierrors.IsAlreadyExists(err) {
			errs = append(errs, errors.Wrapf(err, "error provisioning the %s CRD", ref.name))
		}
	}

	return goerrors.Join(errs...)
}
