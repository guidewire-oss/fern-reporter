package config_test

import (
	"github.com/guidewire/fern-reporter/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
)

var tempConfigFile string

var _ = BeforeSuite(func() {
	// Create a temporary config file for testing
	configContent := `
db:
  driver:   postgres
  host:     localhost
  port:     5432
  database: fern
  username: testuser
  password: testpass
  detail-log: true
  max-open-conns: 100
  max-idle-conns: 10
server:
  port: :8080
`
	tempFile, err := os.CreateTemp("", "config_test*.yaml")
	Expect(err).NotTo(HaveOccurred())
	tempConfigFile = tempFile.Name()
	err = os.WriteFile(tempConfigFile, []byte(configContent), 0644)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	// Cleanup: remove the temporary config file
	err := os.Remove(tempConfigFile)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Config", func() {
	Context("LoadConfig", func() {
		It("should load the configuration from a file", func() {
			configPath, err := filepath.Abs(tempConfigFile)
			Expect(err).NotTo(HaveOccurred())

			appConfig, err := config.LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(appConfig.Server.Port).To(Equal(":8080"))
			Expect(appConfig.Db.Driver).To(Equal("postgres"))
			Expect(appConfig.Db.Host).To(Equal("localhost"))
			Expect(appConfig.Db.Port).To(Equal("5432"))
			Expect(appConfig.Db.Database).To(Equal("fern"))
			Expect(appConfig.Db.Username).To(Equal("testuser"))
			Expect(appConfig.Db.Password).To(Equal("testpass"))
			Expect(appConfig.Db.DetailLog).To(BeTrue())
			Expect(appConfig.Db.MaxOpenConns).To(Equal(100))
			Expect(appConfig.Db.MaxIdleConns).To(Equal(10))
		})

		It("should return an error if the config file is not found", func() {
			_, err := config.LoadConfig("nonexistent_config.yaml")
			Expect(err).To(HaveOccurred())
		})
	})

})
