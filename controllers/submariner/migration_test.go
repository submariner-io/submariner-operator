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

package submariner_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/submariner-io/submariner-operator/controllers/submariner"
	"github.com/submariner-io/submariner/pkg/cni"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Migration tests", func() {
	t := newTestDriver()

	Context("submariner network plugin syncer", func() {
		BeforeEach(func() {
			t.clusterNetwork.NetworkPlugin = cni.OVNKubernetes
		})

		JustBeforeEach(func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)
			t.AssertNoDeployment(ctx, submariner.NetworkPluginSyncerComponent)
		})

		When("the Deployment doesn't exist", func() {
			It("should not create it", func() {
			})
		})

		When("the Deployment does exist", func() {
			BeforeEach(func() {
				t.InitScopedClientObjs = append(t.InitScopedClientObjs,
					t.NewDeployment(submariner.NetworkPluginSyncerComponent),
					&rbacv1.ClusterRole{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: t.Namespace,
							Name:      submariner.NetworkPluginSyncerComponent,
						},
					},
					&rbacv1.ClusterRoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: t.Namespace,
							Name:      submariner.NetworkPluginSyncerComponent,
						},
					},
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: t.Namespace,
							Name:      submariner.NetworkPluginSyncerComponent,
						},
					},
				)
			})

			It("should delete it", func() {
				t.AssertNoResource(&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: submariner.NetworkPluginSyncerComponent,
					},
				})

				t.AssertNoResource(&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: submariner.NetworkPluginSyncerComponent,
					},
				})

				t.AssertNoResource(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name: submariner.NetworkPluginSyncerComponent,
					},
				})
			})
		})
	})
})
