package routers_test

import (
	"github.com/guidewire/fern-reporter/config"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRouters(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Routers Suite")
}

var _ = BeforeSuite(func() {
	_, err := config.LoadConfig()
	Expect(err).NotTo(HaveOccurred())
})
