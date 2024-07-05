package utils_test

import (
	"github.com/guidewire/fern-reporter/pkg/utils"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Suite")
}

var _ = Describe("Utils Test", func() {
	Describe("CalculateDuration", func() {
		It("should correctly calculate the duration between two times", func() {
			start := time.Date(2024, 4, 20, 12, 34, 56, 0, time.UTC)
			end := time.Date(2024, 4, 21, 12, 34, 56, 0, time.UTC)
			expectedDuration := "24h0m0s"

			duration := utils.CalculateDuration(start, end)
			Expect(duration).To(Equal(expectedDuration))
		})
	})

	Describe("FormatDate", func() {
		It("should format the date correctly according to the layout format", func() {
			t := time.Date(2024, 4, 20, 12, 34, 56, 0, time.UTC)
			expectedFormattedDate := "2024-04-20 12:34:56"

			formattedDate := utils.FormatDate(t)
			Expect(formattedDate).To(Equal(expectedFormattedDate))
		})
	})
})
