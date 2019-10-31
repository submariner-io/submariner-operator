package network

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOpenShift4NetworkDiscovery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network discovery")
}
