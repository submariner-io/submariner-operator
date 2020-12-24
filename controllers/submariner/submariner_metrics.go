/*
© 2020 Red Hat, Inc. and others.

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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

const (
	connectionsLocalClusterLabel   = "local_cluster"
	connectionsLocalHostnameLabel  = "local_hostname"
	connectionsRemoteClusterLabel  = "remote_cluster"
	connectionsRemoteHostnameLabel = "remote_hostname"
	connectionsStatusLabel         = "status"
)

var (
	gatewaysGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "submariner_gateways",
			Help: "Number of gateways",
		},
	)
	gatewayCreationTimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "submariner_gateway_creation_timestamp",
			Help: "Timestamp of gateway creation time",
		},
		[]string{
			connectionsLocalClusterLabel,
			connectionsLocalHostnameLabel},
	)
	connectionsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "submariner_connections",
			Help: "Number of connections (by endpoint and status)",
		},
		[]string{
			connectionsLocalClusterLabel,
			connectionsLocalHostnameLabel,
			connectionsRemoteClusterLabel,
			connectionsRemoteHostnameLabel,
			connectionsStatusLabel},
	)
)

func init() {
	metrics.Registry.MustRegister(gatewaysGauge, connectionsGauge, gatewayCreationTimeGauge)
}

func recordGateways(count int) {
	gatewaysGauge.Set(float64(count))
	if count == 0 {
		gatewayCreationTimeGauge.Reset()
	}
}

func recordGatewayCreationTime(localEndpoint submv1.EndpointSpec, upTime time.Time) {
	gatewayCreationTimeGauge.With(prometheus.Labels{
		connectionsLocalClusterLabel:  localEndpoint.ClusterID,
		connectionsLocalHostnameLabel: localEndpoint.Hostname,
	}).Set(float64(upTime.Unix()))
}

func recordNoConnections() {
	connectionsGauge.Reset()
}

func recordConnection(localEndpoint, remoteEndpoint submv1.EndpointSpec, status string) {
	connectionsGauge.With(prometheus.Labels{
		connectionsLocalClusterLabel:   localEndpoint.ClusterID,
		connectionsLocalHostnameLabel:  localEndpoint.Hostname,
		connectionsRemoteClusterLabel:  remoteEndpoint.ClusterID,
		connectionsRemoteHostnameLabel: remoteEndpoint.Hostname,
		connectionsStatusLabel:         status,
	}).Inc()
}
