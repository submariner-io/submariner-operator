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

package prepare

import (
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/releases/projects/cloud-prepare/pkg/k8s"
	"github.com/submariner-io/submariner-operator/pkg/cloud"
	"github.com/submariner-io/submariner-operator/pkg/cloud/gcp"
	"golang.org/x/oauth2/google"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
)

func GCP(port cloud.Port, info *gcp.Options, creds *google.Credentials, restMapper meta.RESTMapper, dynamicClient dynamic.Interface, k8sClient k8s.Interface, reporter api.Reporter) error {
	gwPorts := []api.PortSpec{
		{Port: port.Natt, Protocol: "udp"},
		{Port: port.NatDiscovery, Protocol: "udp"},

		// ESP & AH protocols are used for private-ip to private-ip gateway communications
		{Port: 0, Protocol: "esp"},
		{Port: 0, Protocol: "ah"},
	}
	input := api.PrepareForSubmarinerInput{
		InternalPorts: []api.PortSpec{
			{Port: port.VxLAN, Protocol: "udp"},
			{Port: port.Metrics, Protocol: "tcp"},
		},
	}

	// nolint:wrapcheck // No need to wrap errors here.
	err := gcp.RunOnGCP(info, creds, restMapper, dynamicClient, k8sClient, info.GWInstance,
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error {
			if info.Gateways > 0 {
				gwInput := api.GatewayDeployInput{
					PublicPorts: gwPorts,
					Gateways:    info.Gateways,
				}
				err := gwDeployer.Deploy(gwInput, reporter)
				if err != nil {
					return err
				}
			}

			return cloud.PrepareForSubmariner(input, reporter)
		}, reporter)

	if err != nil {
		return err
	}

	return nil
}
