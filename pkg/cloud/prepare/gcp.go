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
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/cloud"
	"github.com/submariner-io/submariner-operator/pkg/cloud/gcp"
	"golang.org/x/oauth2/google"
)

func GCP(restConfigProducer *restconfig.Producer, ports *cloud.Ports, config *gcp.Config, creds *google.Credentials,
	status reporter.Interface,
) error {
	gwPorts := []api.PortSpec{
		{Port: ports.Natt, Protocol: "udp"},
		{Port: ports.NatDiscovery, Protocol: "udp"},

		// ESP & AH protocols are used for private-ip to private-ip gateway communications
		{Port: 0, Protocol: "esp"},
		{Port: 0, Protocol: "ah"},
	}
	input := api.PrepareForSubmarinerInput{
		InternalPorts: []api.PortSpec{
			{Port: ports.Vxlan, Protocol: "udp"},
			{Port: ports.Metrics, Protocol: "tcp"},
		},
	}

	// nolint:wrapcheck // No need to wrap errors here.
	err := gcp.RunOn(restConfigProducer, config, creds, cli.NewReporter(),
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, status reporter.Interface) error {
			if config.Gateways > 0 {
				gwInput := api.GatewayDeployInput{
					PublicPorts: gwPorts,
					Gateways:    config.Gateways,
				}
				err := gwDeployer.Deploy(gwInput, status)
				if err != nil {
					return err
				}
			}

			return cloud.PrepareForSubmariner(input, status)
		})

	return status.Error(err, "Failed to prepare GCP cloud")
}
