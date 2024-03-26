package client_test

import (
	"time"

	"github.com/guidewire/fern-reporter/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FernApiClient", func() {
	It("should get a new client", func() {

		fernApiClient := client.New("test")

		Expect(fernApiClient).ToNot(BeNil())

	})

	It("should get a new client with BaseURL", func() {

		fernApiClient := client.New("test", client.WithBaseURL("test URL"))

		Expect(fernApiClient).ToNot(BeNil())

	})

	It("should get a new client with HTTP Client", func() {

		fernApiClient := client.New("test", client.WithHTTPClient(nil))

		Expect(fernApiClient).ToNot(BeNil())

	})

	It("should get a new client with timeout", func() {

		fernApiClient := client.New("test", client.WithTimeout(5*time.Second))

		Expect(fernApiClient).ToNot(BeNil())

	})

})
