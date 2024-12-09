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

package apply_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	submarinerNamespace = "test-ns"
)

var log = logf.Log.WithName("test")

var _ = BeforeSuite(func() {
	Expect(v1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
})

var _ = Describe("", func() {
	kzerolog.InitK8sLogging()
})

func TestApply(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Apply Suite")
}

type testDriver struct {
	client         controllerClient.Client
	initClientObjs []controllerClient.Object
	owner          metav1.Object
}

func newTestDriver() *testDriver {
	t := &testDriver{}

	BeforeEach(func() {
		t.initClientObjs = []controllerClient.Object{}
		t.owner = &v1alpha1.Submariner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "submariner",
				Namespace: submarinerNamespace,
			},
		}
	})

	JustBeforeEach(func() {
		t.client = fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(t.initClientObjs...).Build()
	})

	return t
}

func (t *testDriver) verifyOwnerRef(obj metav1.Object) {
	Expect(obj.GetOwnerReferences()).To(HaveLen(1))
	Expect(obj.GetOwnerReferences()[0].Name).To(Equal(t.owner.GetName()))
}
