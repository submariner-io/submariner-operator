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

package ciliumcm

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LocalClusterIdentityFailures returns human-readable problems with the local
// Cilium cluster-id / cluster-name. An empty slice means the identity is usable
// for Submariner's ClusterMesh-shaped publisher.
func LocalClusterIdentityFailures(clusterID, clusterName string) []string {
	var failures []string

	if clusterID == DefaultClusterID {
		failures = append(failures,
			fmt.Sprintf("cilium-config cluster-id %s is reserved for the Submariner ClusterMesh-shaped publisher; use 1..254",
				DefaultClusterID))
	} else if id, err := strconv.Atoi(clusterID); err != nil || id < 1 || id > 254 {
		failures = append(failures,
			fmt.Sprintf("cilium-config cluster-id is %q; set to an integer 1..254 (255 is reserved for Submariner)", clusterID))
	}

	if clusterName == "" || clusterName == "default" {
		failures = append(failures,
			fmt.Sprintf("cilium-config cluster-name is %q; set a non-default name for ClusterMesh", clusterName))
	} else if clusterName == DefaultRemoteName {
		failures = append(failures,
			fmt.Sprintf("cilium-config cluster-name %q is reserved for the Submariner ClusterMesh peer",
				DefaultRemoteName))
	}

	return failures
}

// ValidateLocalClusterIdentity returns an error if cluster-id / cluster-name
// cannot be used with Submariner's ClusterMesh-shaped publisher.
func ValidateLocalClusterIdentity(clusterID, clusterName string) error {
	failures := LocalClusterIdentityFailures(clusterID, clusterName)
	if len(failures) == 0 {
		return nil
	}

	return errors.New(strings.Join(failures, "; "))
}

// LoadAndValidateLocalClusterIdentity reads cilium-config in ciliumNS and
// validates cluster-id / cluster-name.
func LoadAndValidateLocalClusterIdentity(ctx context.Context, client kubernetes.Interface, ciliumNS string) error {
	if ciliumNS == "" {
		return errors.New("cilium namespace is empty; cannot validate cilium-config cluster-id")
	}

	cm, err := client.CoreV1().ConfigMaps(ciliumNS).Get(ctx, CiliumConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return errors.Errorf("ConfigMap %q not found in %q; cannot validate Cilium cluster-id",
				CiliumConfigMapName, ciliumNS)
		}

		return errors.Wrapf(err, "read ConfigMap %q", CiliumConfigMapName)
	}

	return ValidateLocalClusterIdentity(cm.Data["cluster-id"], cm.Data["cluster-name"])
}
