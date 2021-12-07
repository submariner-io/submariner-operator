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

package datafile

import (
	"encoding/base64"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testBrokerURL             = "https://my-broker-url:8443"
	testSASecret              = "submariner-k8s-broker-admin-token-abcdef" // #nosec G101 Test secret only
	testToken                 = "i-am-a-token"
	SubmarinerBrokerNamespace = "submariner-k8s-broker"
	BrokerSA                  = "submariner-k8s-broker-admin"
)

var _ = Describe("datafile", func() {
	When("Doing basic encoding to string", func() {
		It("Should generate data", func() {
			data := &SubctlData{}
			str, err := data.ToString()
			Expect(err).NotTo(HaveOccurred())
			Expect(str).NotTo(BeEmpty())
		})

		It("Should generate base64", func() {
			data := &SubctlData{}
			str, err := data.ToString()
			Expect(err).NotTo(HaveOccurred())
			_, err = base64.URLEncoding.DecodeString(str)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("Doing decoding from string", func() {
		It("Should recover the data", func() {
			data := &SubctlData{BrokerURL: testBrokerURL}
			str, _ := data.ToString()
			newData, err := NewFromString(str)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(newData.BrokerURL).To(Equal(data.BrokerURL))
		})

		It("Should fail on bad data", func() {
			_, err := NewFromString("badstring")
			Expect(err).Should(HaveOccurred())
		})
	})

	When("Getting data from cluster", func() {

		var clientSet *fake.Clientset
		BeforeEach(func() {
			pskSecret, _ := newIPSECPSKSecret()
			pskSecret.Namespace = SubmarinerBrokerNamespace

			sa := broker.NewBrokerSA(BrokerSA)
			sa.Namespace = SubmarinerBrokerNamespace
			sa.Secrets = []v1.ObjectReference{{
				Name: testSASecret,
			}}

			saSecret := &v1.Secret{}
			saSecret.Name = testSASecret
			saSecret.Namespace = SubmarinerBrokerNamespace
			saSecret.Data = map[string][]byte{
				"ca.crt": []byte("i-am-a-cert"),
				"token":  []byte(testToken)}
			clientSet = fake.NewSimpleClientset(pskSecret, sa, saSecret)
		})

		It("Should produce a valid structure", func() {
			subCtlData, err := newFromCluster(clientSet, SubmarinerBrokerNamespace, "")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(subCtlData.IPSecPSK.Name).To(Equal("submariner-ipsec-psk"))
			Expect(subCtlData.ClientToken.Name).To(Equal(testSASecret))
		})
	})
})

func TestDataFile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subctl datafile")
}
