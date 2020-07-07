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
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	connectionsLocalClusterLabel   = "local_cluster"
	connectionsLocalHostnameLabel  = "local_hostname"
	connectionsRemoteClusterLabel  = "remote_cluster"
	connectionsRemoteHostnameLabel = "remote_hostname"
	connectionsStatusLabel         = "status"
)

var (
	reconciliationsCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "submariner_reconciliations",
			Help: "Number of reconciliations processed",
		},
	)
	gatewaysGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "submariner_gateways",
			Help: "Number of gateways",
		},
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
	metrics.Registry.MustRegister(reconciliationsCounter, gatewaysGauge, connectionsGauge)
}

func recordReconciliation() {
	reconciliationsCounter.Inc()
}

func recordGateways(count int) {
	gatewaysGauge.Set(float64(count))
}

func recordNoConnections() {
	connectionsGauge.Reset()
}

func recordConnection(localCluster string, localHostname string, remoteCluster string, remoteHostname string, status string) {
	connectionsGauge.With(prometheus.Labels{
		connectionsLocalClusterLabel:   localCluster,
		connectionsLocalHostnameLabel:  localHostname,
		connectionsRemoteClusterLabel:  remoteCluster,
		connectionsRemoteHostnameLabel: remoteHostname,
		connectionsStatusLabel:         status,
	}).Inc()
}
