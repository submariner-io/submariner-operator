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

package webhook_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/submariner-operator/pkg/webhook"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	mcsv1b1 "sigs.k8s.io/mcs-api/pkg/apis/v1beta1"
)

const (
	testClusterID   = "cluster-a"
	otherClusterID  = "other-cluster"
	brokerNamespace = "submariner-k8s-broker"
)

var _ = Describe("BrokerValidator", func() {
	t := newTestDriver()

	DescribeTableSubtree("when the request user name is",
		func(username string) {
			It("should allow", func(ctx context.Context) {
				t.username = username

				resp := t.validator.Handle(ctx, t.createRequest(newSecret(""), nil, admissionv1.Create))
				Expect(resp.Allowed).To(BeTrue())
				Expect(resp.Result.Message).To(Equal(webhook.NotBrokerClientSAMessage))
			})
		},
		Entry("not a broker-client SA", fmt.Sprintf("system:serviceaccount:%s:submariner-k8s-broker-admin", brokerNamespace)),
		Entry("not a SA", "system:user:foo:bar"),
		Entry("a user", "admin"),
		Entry("a SA from another namespace", "system:serviceaccount:attacker-namespace:cluster-malicious"),
	)

	t.testAccess("own certificate Secret", newSecret(testClusterID), AllowOwn)
	t.testAccess("another cluster's certificate Secret", newSecret("cluster-b"), Deny)
	t.testAccess("unlabeled Secret", newSecret(""), Deny)

	t.testAccess("own Endpoint", newEndpoint(testClusterID), AllowOwn)
	t.testAccess("another cluster's Endpoint", newEndpoint("cluster-b"), Deny)

	t.testAccess("own Cluster", newCluster(testClusterID), AllowOwn)
	t.testAccess("another cluster's Cluster", newCluster("cluster-b"), Deny)

	t.testAccess("own EndpointSlice", newEndpointSlice(testClusterID), AllowOwn)
	t.testAccess("another cluster's EndpointSlice", newEndpointSlice("cluster-b"), Deny)
	t.testAccess("unlabeled EndpointSlice", newEndpointSlice(""), Deny)

	t.testAccess("own ServiceImport", newServiceImport(testClusterID), AllowOwn)
	t.testAccess("another cluster's ServiceImport", newServiceImport("cluster-b"), Deny)
	t.testAccess("unlabeled ServiceImport", newServiceImport(""), Deny)

	t.testAccess("an aggregate ServiceImport with only other clusters present", newAggregatedServiceImport(otherClusterID), Deny)

	DescribeTableSubtree("an aggregate ServiceImport with no clusters present",
		func(op admissionv1.Operation, access Access) {
			t.testHandle(op, newAggregatedServiceImport(), access)
		},
		Entry("", admissionv1.Create, Allow),
		Entry("", admissionv1.Update, Deny),
		Entry("", admissionv1.Delete, Allow),
	)

	Context("an aggregate ServiceImport with own cluster present", func() {
		DescribeTableSubtree("and other clusters present",
			func(op admissionv1.Operation, access Access) {
				t.testHandle(op, newAggregatedServiceImport(otherClusterID, testClusterID), access)
			},
			Entry("", admissionv1.Create, Deny),
			Entry("", admissionv1.Update, Allow),
			Entry("", admissionv1.Delete, Deny),
		)

		DescribeTableSubtree("and no other clusters present",
			func(op admissionv1.Operation) {
				t.testHandle(op, newAggregatedServiceImport(testClusterID), Allow)
			},
			Entry("", admissionv1.Create),
			Entry("", admissionv1.Update),
			Entry("", admissionv1.Delete),
		)

		Context("and attempting to add/remove other clusters", func() {
			It("should deny access", func(ctx context.Context) {
				old := newAggregatedServiceImport(testClusterID, otherClusterID)

				Expect(t.validator.Handle(ctx, t.createRequest(newAggregatedServiceImport(testClusterID), old,
					admissionv1.Update)).Allowed).To(BeFalse())

				Expect(t.validator.Handle(ctx, t.createRequest(newAggregatedServiceImport("third", testClusterID, otherClusterID),
					old, admissionv1.Update)).Allowed).To(BeFalse())
			})
		})

		Context("and attempting to replace another cluster's ID", func() {
			It("should deny access", func(ctx context.Context) {
				old := newAggregatedServiceImport(testClusterID, otherClusterID)

				Expect(t.validator.Handle(ctx, t.createRequest(newAggregatedServiceImport(testClusterID, "different-cluster"),
					old, admissionv1.Update)).Allowed).To(BeFalse())
			})
		})

		Context("and attempting to add/remove own cluster", func() {
			It("should allow access", func(ctx context.Context) {
				Expect(t.validator.Handle(ctx, t.createRequest(newAggregatedServiceImport(testClusterID),
					newAggregatedServiceImport(), admissionv1.Update)).Allowed).To(BeTrue())

				Expect(t.validator.Handle(ctx, t.createRequest(newAggregatedServiceImport(),
					newAggregatedServiceImport(testClusterID), admissionv1.Update)).Allowed).To(BeTrue())

				Expect(t.validator.Handle(ctx, t.createRequest(newAggregatedServiceImport(otherClusterID, testClusterID),
					newAggregatedServiceImport(otherClusterID), admissionv1.Update)).Allowed).To(BeTrue())

				Expect(t.validator.Handle(ctx, t.createRequest(newAggregatedServiceImport(otherClusterID),
					newAggregatedServiceImport(otherClusterID, testClusterID), admissionv1.Update)).Allowed).To(BeTrue())
			})
		})
	})
})

type testDriver struct {
	validator  *webhook.BrokerValidator
	serializer runtime.Serializer
	username   string
}

func newTestDriver() *testDriver {
	t := &testDriver{}

	BeforeEach(func() {
		t.username = fmt.Sprintf("system:serviceaccount:%s:cluster-%s", brokerNamespace, testClusterID)

		t.validator = webhook.NewBrokerValidator()

		scheme := webhook.NewScheme()
		t.serializer = json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{})
	})

	return t
}

type Access struct {
	value   bool
	message string
}

var (
	AllowOwn = Access{value: true, message: "accessing own"}
	Allow    = Access{value: true}
	Deny     = Access{value: false}
)

func (a Access) String() string {
	if a.value {
		return "allow"
	}

	return "deny"
}

func (t *testDriver) testAccess(desc string, obj runtime.Object, writeAccess Access) {
	DescribeTableSubtree("with "+desc, t.testHandle,
		Entry("", admissionv1.Create, obj, writeAccess),
		Entry("", admissionv1.Update, obj, writeAccess),
		Entry("", admissionv1.Delete, obj, writeAccess),
	)
}

func (t *testDriver) testHandle(op admissionv1.Operation, obj runtime.Object, access Access) {
	It(fmt.Sprintf("should %s %s access", access, op), func(ctx context.Context) {
		resp := t.validator.Handle(ctx, t.createRequest(obj, nil, op))
		Expect(resp.Allowed).To(Equal(access.value))

		if access.message != "" {
			Expect(resp.Result.Message).To(ContainSubstring(access.message))
		}
	})
}

func (t *testDriver) createRequest(obj, old runtime.Object, op admissionv1.Operation) admission.Request {
	objBytes, err := runtime.Encode(t.serializer, obj)
	Expect(err).NotTo(HaveOccurred())

	namespace := resource.MustToMeta(obj).GetNamespace()
	if namespace == "" {
		namespace = brokerNamespace
	}

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Operation: op,
			UserInfo: authenticationv1.UserInfo{
				Username: t.username,
			},
			Kind:      metav1.GroupVersionKind(obj.GetObjectKind().GroupVersionKind()),
			Name:      resource.MustToMeta(obj).GetName(),
			Namespace: namespace,
		},
	}

	if op == admissionv1.Delete {
		req.OldObject = runtime.RawExtension{
			Raw: objBytes,
		}
	} else {
		req.Object = runtime.RawExtension{
			Raw: objBytes,
		}
	}

	if op == admissionv1.Update {
		if old == nil {
			old = obj
		}

		objBytes, err = runtime.Encode(t.serializer, old)
		Expect(err).NotTo(HaveOccurred())

		req.OldObject = runtime.RawExtension{
			Raw: objBytes,
		}
	}

	return req
}

func newSecret(clusterID string) *corev1.Secret {
	s := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-secret",
		},
	}

	if clusterID != "" {
		s.Labels = map[string]string{webhook.SigningRequestLabelKey: clusterID}
	}

	return s
}

func newCluster(clusterID string) *submarinerv1.Cluster {
	return &submarinerv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: submarinerv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterID,
		},
		Spec: submarinerv1.ClusterSpec{
			ClusterID: clusterID,
		},
	}
}

func newEndpoint(clusterID string) *submarinerv1.Endpoint {
	return &submarinerv1.Endpoint{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Endpoint",
			APIVersion: submarinerv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-endpoint",
		},
		Spec: submarinerv1.EndpointSpec{
			ClusterID: clusterID,
		},
	}
}

func newEndpointSlice(clusterID string) *discoveryv1.EndpointSlice {
	s := &discoveryv1.EndpointSlice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EndpointSlice",
			APIVersion: discoveryv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-eps",
		},
	}

	if clusterID != "" {
		s.Labels = map[string]string{mcsv1b1.LabelSourceCluster: clusterID}
	}

	return s
}

func newServiceImport(clusterID string) *mcsv1b1.ServiceImport {
	s := &mcsv1b1.ServiceImport{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceImport",
			APIVersion: mcsv1b1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-si",
			Labels: map[string]string{mcsv1b1.LabelServiceName: "nginx"},
		},
	}

	if clusterID != "" {
		s.Labels[mcsv1b1.LabelSourceCluster] = clusterID
	}

	return s
}

func newAggregatedServiceImport(clusterIDs ...string) *mcsv1b1.ServiceImport {
	s := &mcsv1b1.ServiceImport{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceImport",
			APIVersion: mcsv1b1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-si",
			Annotations: map[string]string{mcsv1b1.LabelServiceName: "nginx"},
		},
	}

	for _, clusterID := range clusterIDs {
		s.Status.Clusters = append(s.Status.Clusters, mcsv1b1.ClusterStatus{Cluster: clusterID})
	}

	return s
}
