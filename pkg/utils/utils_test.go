package utils_test

import (
	"github.com/guidewire/fern-reporter/pkg/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"

	"github.com/guidewire/fern-reporter/pkg/utils"
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
})
