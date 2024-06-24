package auth_test

import (
	"github.com/guidewire/fern-reporter/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Suite")
}

var _ = BeforeSuite(func() {
	_, err := config.LoadConfig()
	Expect(err).NotTo(HaveOccurred())
	os.Setenv("SCOPE_CLAIM_NAME", "scp")
})
