package utils

import (
	"github.com/guidewire/fern-reporter/pkg/models"
	"time"
)

const (
	DateLayoutFormat = "2006-01-02 15:04:05"
	StatusSkipped    = "skipped"
	StatusPassed     = "passed"
	StatusFailed     = "failed"
)

func CalculateDuration(start, end time.Time) string {
	duration := end.Sub(start)
	return duration.String() // or format as needed
}

func FormatDate(t time.Time) string {
	return t.Format(DateLayoutFormat)
}

// Common function to calculate test metrics
func CalculateTestMetrics(testRuns []models.TestRun) (totalTests, executedTests, passedTests, failedTests int) {
	for _, testRun := range testRuns {
		for _, suiteRun := range testRun.SuiteRuns {
			for _, specRun := range suiteRun.SpecRuns {
				totalTests++ // Count each spec run
				if specRun.Status != StatusSkipped {
					executedTests++ // Count only executed spec runs
					if specRun.Status == StatusPassed {
						passedTests++ // Count passed spec runs
					} else if specRun.Status == StatusFailed {
						failedTests++ // Count failed spec runs
					}
				}
			}
		}
	}
	return
}
