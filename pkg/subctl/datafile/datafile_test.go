package datafile

import (
	"encoding/base64"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const testBrokerUrl = "https://my-broker-url:8443"

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
			data := &SubctlData{BrokerURL: testBrokerUrl}
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
})

func TestDataFile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subctl datafile")
}
