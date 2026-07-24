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

// Package ciliumcm holds helpers for wiring Submariner's Cilium ClusterMesh-shaped
// ipcache publisher (TLS material and peer Secret keys).
package ciliumcm

const (
	// TLSSecretName is the Secret in the operator namespace holding CA/server/client PEM.
	TLSSecretName = "submariner-cilium-cm-tls"

	// ClusterMeshSecretName is Cilium's peer Secret (typically in kube-system).
	ClusterMeshSecretName = "cilium-clustermesh"

	// CiliumConfigMapName is the Cilium agent ConfigMap.
	CiliumConfigMapName = "cilium-config"

	// DefaultRemoteName is the ClusterMesh peer name reserved for Submariner.
	// Only Secret keys for this peer name are written or removed; other ClusterMesh
	// peers in cilium-clustermesh are left untouched.
	DefaultRemoteName = "submariner"

	// DefaultClusterID is the synthetic remote cluster-id published into embed etcd.
	DefaultClusterID = "99"

	// DefaultListenURL is the per-node etcd client URL (TLS).
	DefaultListenURL = "https://127.0.0.1:12379"

	// DefaultPeerURL is the local-only etcd peer URL.
	DefaultPeerURL = "http://127.0.0.1:12380"

	// VolumeName mounted into route-agent.
	VolumeName = "cilium-cm-tls"

	// MountPath inside the route-agent container.
	MountPath = "/var/run/secrets/submariner.io/cilium-cm-tls"

	// CACertKey and the following constants are Secret data keys.
	CACertKey     = "ca.crt"
	CAKeyKey      = "ca.key"
	TLSCertKey    = "tls.crt"
	TLSKeyKey     = "tls.key"
	ClientCertKey = "client.crt"
	ClientKeyKey  = "client.key"
)

// PeerSecretKeys returns the cilium-clustermesh Secret.Data keys owned by Submariner
// for the given ClusterMesh peer name. Callers must only write/delete these keys.
func PeerSecretKeys(remoteName string) []string {
	return []string{
		remoteName,
		remoteName + ".etcd-client-ca.crt",
		remoteName + ".etcd-client.crt",
		remoteName + ".etcd-client.key",
	}
}
