package broker

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBrokerSetup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker setup handling")
}
