package resolvers_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestResolversSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resolvers Suite")
}
