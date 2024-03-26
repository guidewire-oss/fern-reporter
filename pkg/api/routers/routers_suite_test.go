package routers_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRouters(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Routers Suite")
}
