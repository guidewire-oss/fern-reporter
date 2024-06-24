package config_test

import (
	"github.com/guidewire/fern-reporter/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"os"
)

// Mock the file reading function
type MockFileReader struct {
	mock.Mock
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	args := m.Called(filename)
	return args.Get(0).([]byte), args.Error(1)
}

var _ = Describe("When LoadConfig is invoked", func() {

	Context("LoadConfig", func() {
		It("should load the configuration from a file", func() {

			appConfig, err := config.LoadConfig()
			Expect(err).NotTo(HaveOccurred())

			Expect(appConfig.Auth.KeysEndpoint).To(Equal(""))
			Expect(appConfig.Server.Port).To(Equal(":8080"))
			Expect(appConfig.Db.Driver).To(Equal("postgres"))
			Expect(appConfig.Db.Host).To(Equal("localhost"))
			Expect(appConfig.Db.Port).To(Equal("5432"))
			Expect(appConfig.Db.Database).To(Equal("fern"))
			Expect(appConfig.Db.Username).To(Equal("fern"))
			Expect(appConfig.Db.Password).To(Equal("fern"))
			Expect(appConfig.Db.DetailLog).To(BeTrue())
			Expect(appConfig.Db.MaxOpenConns).To(Equal(100))
			Expect(appConfig.Db.MaxIdleConns).To(Equal(10))
			Expect(appConfig.Header).To(Equal("Fern Acceptance Test Report"))
		})

		It("should get non-nil DB", func() {

			_, err := config.LoadConfig()
			Expect(err).NotTo(HaveOccurred())

			Expect(config.GetDb()).ToNot(BeNil())
		})

		It("should get non-nil Server", func() {

			_, err := config.LoadConfig()
			Expect(err).NotTo(HaveOccurred())

			Expect(config.GetServer()).ToNot(BeNil())
		})

	})

	It("should override configuration with environment variables", func() {

		os.Setenv("AUTH_KEYS_ENDPOINT", "https://test-idp-base-url.com/oauth2/abc123/v1/keys")
		os.Setenv("FERN_USERNAME", "fern")
		os.Setenv("FERN_PASSWORD", "fern")
		os.Setenv("FERN_HOST", "localhost")
		os.Setenv("FERN_PORT", "5432")
		os.Setenv("FERN_DATABASE", "fern")
		os.Setenv("FERN_HEADER_NAME", "Custom Fern Report Header")

		//v := viper.New()
		result, err := config.LoadConfig()

		Expect(err).To(BeNil())
		Expect(result).ToNot(BeNil())
		Expect(result.Db.Username).To(Equal("fern"))
		Expect(result.Db.Password).To(Equal("fern"))
		Expect(result.Db.Host).To(Equal("localhost"))
		Expect(result.Db.Port).To(Equal("5432"))
		Expect(result.Db.Database).To(Equal("fern"))
		Expect(result.Auth.KeysEndpoint).To(Equal("https://test-idp-base-url.com/oauth2/abc123/v1/keys"))
		Expect(result.Header).To(Equal("Custom Fern Report Header"))
	})

})
