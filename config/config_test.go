package config_test

import (
	"github.com/guidewire/fern-reporter/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Context("LoadConfig", func() {
		It("should load the configuration from a file", func() {

			appConfig, err := config.LoadConfig()
			Expect(err).NotTo(HaveOccurred())

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

})
