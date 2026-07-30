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

package webhook

import (
	"context"
	"fmt"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	operatorv1alpha1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/names"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcsv1a1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var _ = Describe("Broker webhook", func() {
	var (
		generalClient      client.Client
		impersonatedClient client.Client
		otherClusterID     string
		submariner         *operatorv1alpha1.Submariner
		scheme             *runtime.Scheme
		brokerCluster      framework.ClusterIndex
		brokerNamespace    string
	)

	BeforeEach(func(ctx context.Context) {
		var err error

		brokerCluster, brokerNamespace = findBrokerCluster(ctx)
		framework.By(fmt.Sprintf("Found broker on cluster %q, namespace %q",
			framework.TestContext.ClusterIDs[brokerCluster], brokerNamespace))

		otherClusterIndex := framework.FindOtherClusterIndex(int(brokerCluster))
		Expect(otherClusterIndex).NotTo(Equal(-1))

		otherClusterID = framework.TestContext.ClusterIDs[otherClusterIndex]

		framework.By(fmt.Sprintf("Impersonating SA for cluster %q", framework.TestContext.ClusterIDs[brokerCluster]))

		// Create impersonated client for ClusterA's broker client service account
		impersonatedConfig := rest.CopyConfig(framework.RestConfigs[brokerCluster])
		impersonatedConfig.Impersonate = rest.ImpersonationConfig{
			UserName: "system:serviceaccount:" + brokerNamespace + ":" +
				names.ForClusterSA(framework.TestContext.ClusterIDs[brokerCluster]),
		}

		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(rbacv1.AddToScheme(scheme)).To(Succeed())
		Expect(discoveryv1.AddToScheme(scheme)).To(Succeed())
		Expect(mcsv1a1.Install(scheme)).To(Succeed())
		Expect(submarinerv1.AddToScheme(scheme)).To(Succeed())
		Expect(operatorv1alpha1.AddToScheme(scheme)).To(Succeed())

		impersonatedClient, err = client.New(impersonatedConfig, client.Options{Scheme: scheme})
		Expect(err).NotTo(HaveOccurred())

		generalClient, err = client.New(framework.RestConfigs[brokerCluster], client.Options{Scheme: scheme})
		Expect(err).To(Succeed())

		submariner = &operatorv1alpha1.Submariner{}
		err = generalClient.Get(ctx, client.ObjectKey{
			Namespace: framework.TestContext.SubmarinerNamespace,
			Name:      names.SubmarinerCrName,
		}, submariner)
		Expect(err).To(Succeed())
	})

	It("should deny all write access to another cluster's Endpoint", func(ctx context.Context) {
		newEndpoint := &submarinerv1.Endpoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "forbidden-endpoint",
				Namespace: brokerNamespace,
			},
			Spec: submarinerv1.EndpointSpec{
				ClusterID: otherClusterID,
				CableName: "submariner-cable-other-192-168-1-200",
				Subnets:   []string{"10.245.0.0/16"},
			},
		}

		framework.By(fmt.Sprintf("Attempting Endpoint create/update/delete for cluster %q", otherClusterID))

		Eventually(func(g Gomega) {
			existing := findEndpoint(ctx, g, impersonatedClient, otherClusterID, brokerNamespace)

			expectForbidden(g, impersonatedClient.Create(ctx, newEndpoint))
			expectForbidden(g, impersonatedClient.Update(ctx, existing))
			expectForbidden(g, impersonatedClient.Delete(ctx, existing))
		}).Within(time.Minute).ProbeEvery(time.Second).Should(Succeed())
	})

	It("should deny all write access to another cluster's Cluster", func(ctx context.Context) {
		newCluster := &submarinerv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "forbidden-cluster",
				Namespace: brokerNamespace,
			},
			Spec: submarinerv1.ClusterSpec{
				ClusterID:   "forbidden-cluster",
				ClusterCIDR: []string{"10.86.0.0/16"},
				ServiceCIDR: []string{"10.140.0.0/16"},
				GlobalCIDR:  []string{},
			},
		}

		framework.By(fmt.Sprintf("Attempting Cluster create/update/delete for cluster %q", otherClusterID))

		Eventually(func(g Gomega) {
			existing := &submarinerv1.Cluster{}
			g.Expect(impersonatedClient.Get(ctx, client.ObjectKey{Name: otherClusterID, Namespace: brokerNamespace},
				existing)).To(Succeed())

			expectForbidden(g, impersonatedClient.Create(ctx, newCluster))
			expectForbidden(g, impersonatedClient.Update(ctx, existing))
			expectForbidden(g, impersonatedClient.Delete(ctx, existing))
		}).Within(time.Second * 2).ProbeEvery(time.Second).Should(Succeed())
	})

	DescribeTableSubtree("",
		func(typeStr string, newObj func() client.Object) {
			It("should deny create access to another cluster's "+typeStr, func(ctx context.Context) {
				obj := newObj()
				framework.By(fmt.Sprintf("Attempting to create %T for cluster %q: %s", obj, otherClusterID,
					resource.ToJSON(obj)))

				Eventually(func(g Gomega) {
					expectForbidden(g, impersonatedClient.Create(ctx, obj))
				}).Within(time.Second * 5).ProbeEvery(time.Second).Should(Succeed())
			})
		},
		Entry("", "Secret", func() client.Object {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "forbidden-secret",
					Namespace: brokerNamespace,
					Labels:    map[string]string{"submariner.io/csr-request": otherClusterID},
				},
			}
		}),
		Entry("", "EndpointSlice", func() client.Object {
			return &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "forbidden-eps",
					Namespace: brokerNamespace,
					Labels:    map[string]string{mcsv1a1.LabelSourceCluster: otherClusterID},
				},
				AddressType: discoveryv1.AddressTypeIPv4,
			}
		}),
		Entry("", "ServiceImport", func() client.Object {
			return &mcsv1a1.ServiceImport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "forbidden-si",
					Namespace: brokerNamespace,
					Labels: map[string]string{
						mcsv1a1.LabelServiceName:   "nginx",
						mcsv1a1.LabelSourceCluster: otherClusterID,
					},
				},
				Spec: mcsv1a1.ServiceImportSpec{
					Type:  mcsv1a1.ClusterSetIP,
					Ports: []mcsv1a1.ServicePort{},
				},
			}
		}),
	)

	Context("for aggregate ServiceImports", func() {
		BeforeEach(func() {
			if !submariner.Spec.ServiceDiscoveryEnabled {
				Skip("Service Discovery is not enabled")
			}

			err := generalClient.Create(context.TODO(), &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      names.ForClusterSA(otherClusterID) + "-submariner-k8s-broker-cluster",
					Namespace: brokerNamespace,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "submariner-k8s-broker-cluster",
				},
				Subjects: []rbacv1.Subject{
					{
						Namespace: brokerNamespace,
						Name:      names.ForClusterSA(otherClusterID),
						Kind:      "ServiceAccount",
					},
				},
			})
			if !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should only allow access to participating clusters", func(ctx context.Context) {
			serviceName := "test-service"
			clusterName := framework.TestContext.ClusterIDs[brokerCluster]

			framework.By(fmt.Sprintf("Creating test namespace on %q", clusterName))

			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-webhook-",
				},
			}
			Expect(generalClient.Create(ctx, ns)).To(Succeed())

			serviceNamespace := ns.Name

			DeferCleanup(func(ctx context.Context) {
				Expect(generalClient.Delete(ctx, ns)).To(Succeed())
			})

			framework.By(fmt.Sprintf("Creating test service on %q", clusterName))

			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: serviceNamespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:     "http",
							Protocol: corev1.ProtocolTCP,
							Port:     80,
						},
					},
					Selector: map[string]string{
						"app": "test",
					},
				},
			}
			Expect(generalClient.Create(ctx, svc)).To(Succeed())

			framework.By(fmt.Sprintf("Exporting the service on %q", clusterName))

			serviceExport := &mcsv1a1.ServiceExport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: serviceNamespace,
				},
			}
			Expect(generalClient.Create(ctx, serviceExport)).To(Succeed())

			framework.By("Awaiting aggregated ServiceImport")

			serviceImport := &mcsv1a1.ServiceImport{}

			Eventually(func(g Gomega) {
				g.Expect(generalClient.Get(ctx, client.ObjectKey{
					Namespace: brokerNamespace,
					Name:      fmt.Sprintf("%s-%s", serviceName, serviceNamespace),
				}, serviceImport)).To(Succeed())
			}).Within(5 * time.Second).To(Succeed())

			impersonatedConfig := rest.CopyConfig(framework.RestConfigs[brokerCluster])
			impersonatedConfig.Impersonate = rest.ImpersonationConfig{
				UserName: "system:serviceaccount:" + brokerNamespace + ":" + names.ForClusterSA(otherClusterID),
			}

			otherClusterClient, err := client.New(impersonatedConfig, client.Options{Scheme: scheme})
			Expect(err).NotTo(HaveOccurred())

			framework.By(fmt.Sprintf("Attempting access to ServiceImport impersonating cluster %q - should be denied", otherClusterID))

			expectForbidden(Default, otherClusterClient.Delete(ctx, serviceImport))

			resourceClient := resource.ForControllerClient(otherClusterClient, brokerNamespace, &mcsv1a1.ServiceImport{})

			Eventually(func(g Gomega) {
				expectForbidden(g, util.MustUpdate(ctx, resourceClient, serviceImport, util.Replace(serviceImport)))
			}).Within(5 * time.Second).To(Succeed())

			framework.By(fmt.Sprintf("Attempting to add cluster %q to the Status impersonating cluster %q - should be allowed",
				otherClusterID, otherClusterID))

			Eventually(func(g Gomega) {
				g.Expect(util.MustUpdate(ctx, resourceClient, serviceImport, func(si *mcsv1a1.ServiceImport) (*mcsv1a1.ServiceImport, error) {
					si.Status.Clusters = append(si.Status.Clusters, mcsv1a1.ClusterStatus{Cluster: otherClusterID})
					return si, nil
				})).To(Succeed())
			}).Within(5 * time.Second).To(Succeed())

			framework.By(fmt.Sprintf("Attempting to remove cluster %q from the Status impersonating cluster %q - should be denied",
				clusterName, otherClusterID))

			Eventually(func(g Gomega) {
				expectForbidden(g, util.MustUpdate(ctx, resourceClient, serviceImport, func(si *mcsv1a1.ServiceImport) (*mcsv1a1.ServiceImport, error) {
					si.Status.Clusters = slices.DeleteFunc(si.Status.Clusters, func(c mcsv1a1.ClusterStatus) bool {
						return c.Cluster == clusterName
					})

					return si, nil
				}))
			}).Within(5 * time.Second).To(Succeed())

			framework.By(fmt.Sprintf("Attempting to remove cluster %q from the Status impersonating cluster %q - should be allowed",
				otherClusterID, otherClusterID))

			Eventually(func(g Gomega) {
				g.Expect(util.MustUpdate(ctx, resourceClient, serviceImport, func(si *mcsv1a1.ServiceImport) (*mcsv1a1.ServiceImport, error) {
					si.Status.Clusters = slices.DeleteFunc(si.Status.Clusters, func(c mcsv1a1.ClusterStatus) bool {
						return c.Cluster == otherClusterID
					})

					return si, nil
				})).To(Succeed())
			}).Within(5 * time.Second).To(Succeed())

			framework.By("Deleting ServiceExport")

			Expect(generalClient.Delete(ctx, serviceExport)).To(Succeed())

			Eventually(func(g Gomega) {
				err := generalClient.Get(ctx, client.ObjectKey{
					Namespace: brokerNamespace,
					Name:      fmt.Sprintf("%s-%s", serviceName, serviceNamespace),
				}, serviceImport)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "ServiceImport should be deleted")
			}).Within(10 * time.Second).To(Succeed())
		})
	})
})

func expectForbidden(g Gomega, err error) {
	g.Expect(apierrors.IsForbidden(err)).To(BeTrue(), "Expected Forbidden error but got %q, %T, %s",
		err, err, resource.ToJSON(err))
}

func findEndpoint(ctx context.Context, g Gomega, c client.Client, clusterID, brokerNamespace string) *submarinerv1.Endpoint {
	endpointList := &submarinerv1.EndpointList{}
	g.Expect(c.List(ctx, endpointList, client.InNamespace(brokerNamespace))).To(Succeed())

	index := slices.IndexFunc(endpointList.Items, func(e submarinerv1.Endpoint) bool {
		return e.Spec.ClusterID == clusterID
	})

	g.Expect(index).NotTo(Equal(-1))

	return &endpointList.Items[index]
}
