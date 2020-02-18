package globalnet_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGlobalnet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Globalnet Suite")
}
