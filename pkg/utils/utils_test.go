package utils_test

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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

	Describe("CalculateTestMetrics", func() {
		var (
			testRuns []models.TestRun
		)

		Context("when there are no test runs", func() {
			It("should return zeros for all metrics", func() {
				total, executed, passed, failed := utils.CalculateTestMetrics(testRuns)
				Expect(total).To(Equal(0))
				Expect(executed).To(Equal(0))
				Expect(passed).To(Equal(0))
				Expect(failed).To(Equal(0))
			})
		})

		Context("when there are test runs with different statuses", func() {
			BeforeEach(func() {
				testRuns = []models.TestRun{
					{
						ID: 1,
						SuiteRuns: []models.SuiteRun{
							{
								ID: 1,
								SpecRuns: []models.SpecRun{
									{Status: "passed"},
									{Status: "failed"},
									{Status: "skipped"},
								},
							},
						},
					},
				}
			})
			It("should correctly count total, executed, passed, and failed tests", func() {
				total, executed, passed, failed := utils.CalculateTestMetrics(testRuns)
				Expect(total).To(Equal(3))
				Expect(executed).To(Equal(2)) // Skipped is not executed
				Expect(passed).To(Equal(1))
				Expect(failed).To(Equal(1))
			})
		})

		Context("when there are multiple test runs with various statuses", func() {
			BeforeEach(func() {
				testRuns = []models.TestRun{
					{
						ID: 1,
						SuiteRuns: []models.SuiteRun{
							{
								ID: 1,
								SpecRuns: []models.SpecRun{
									{Status: "passed"},
									{Status: "failed"},
									{Status: "skipped"},
								},
							},
						},
					},
					{
						ID: 2,
						SuiteRuns: []models.SuiteRun{
							{
								ID: 2,
								SpecRuns: []models.SpecRun{
									{Status: "passed"},
									{Status: "passed"},
									{Status: "failed"},
								},
							},
						},
					},
				}
			})
			It("should correctly aggregate counts across multiple test runs", func() {
				total, executed, passed, failed := utils.CalculateTestMetrics(testRuns)
				Expect(total).To(Equal(6))    // Total spec runs
				Expect(executed).To(Equal(5)) // Skipped is not executed
				Expect(passed).To(Equal(3))
				Expect(failed).To(Equal(2))
			})
		})

		Context("when there are test runs with no executed specs", func() {
			BeforeEach(func() {
				testRuns = []models.TestRun{
					{
						ID: 1,
						SuiteRuns: []models.SuiteRun{
							{
								ID: 1,
								SpecRuns: []models.SpecRun{
									{Status: "skipped"},
									{Status: "skipped"},
								},
							},
						},
					},
				}
			})
			It("should count total tests but not executed, passed, or failed tests", func() {
				total, executed, passed, failed := utils.CalculateTestMetrics(testRuns)
				Expect(total).To(Equal(2))
				Expect(executed).To(Equal(0))
				Expect(passed).To(Equal(0))
				Expect(failed).To(Equal(0))
			})
		})
	})

	Describe("EncodeCursor", func() {
		Context("when called with a positive offset", func() {
			It("should return the correct base64-encoded string", func() {
				offset := 5
				expected := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", offset)))
				result := utils.EncodeCursor(offset)
				Expect(result).To(Equal(expected))
			})
		})

		Context("when called with zero as the offset", func() {
			It("should return the correct base64-encoded string", func() {
				offset := 0
				expected := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", offset)))
				result := utils.EncodeCursor(offset)
				Expect(result).To(Equal(expected))
			})
		})

		Context("when called with a negative offset", func() {
			It("should return the correct base64-encoded string", func() {
				offset := -10
				expected := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", offset)))
				result := utils.EncodeCursor(offset)
				Expect(result).To(Equal(expected))
			})
		})

		Context("when called with a large offset", func() {
			It("should return the correct base64-encoded string", func() {
				offset := 1234567890
				expected := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", offset)))
				result := utils.EncodeCursor(offset)
				Expect(result).To(Equal(expected))
			})
		})
	})

	Describe("DecodeCursor", func() {
		Context("when called with a nil cursor", func() {
			It("should return 0", func() {
				var cursor *string = nil
				result := utils.DecodeCursor(cursor)
				Expect(result).To(Equal(0))
			})
		})

		Context("when called with a valid base64-encoded cursor", func() {
			It("should return the correct offset", func() {
				offset := 5
				encodedCursor := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor%d", offset)))
				result := utils.DecodeCursor(&encodedCursor)
				Expect(result).To(Equal(offset))
			})
		})

		Context("when called with an invalid base64-encoded cursor", func() {
			It("should return 0", func() {
				invalidCursor := "invalid_base64_string"
				result := utils.DecodeCursor(&invalidCursor)
				Expect(result).To(Equal(0))
			})
		})

		Context("when called with a valid base64 string that doesn't match the expected format", func() {
			It("should return 0", func() {
				encodedCursor := base64.StdEncoding.EncodeToString([]byte("not_a_cursor_format"))
				result := utils.DecodeCursor(&encodedCursor)
				Expect(result).To(Equal(0))
			})
		})
	})

})
