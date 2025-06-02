package datamigrations

import (
	"github.com/guidewire/fern-reporter/pkg/models"
	"gorm.io/gorm"
	"log"
	"strings"
)

func BackfillTestRunStatus(db *gorm.DB) {

	// Short-circuit if all test runs already have status
	var missingStatusCount int64
	db.Model(&models.TestRun{}).Where("status IS NULL OR status = ''").Count(&missingStatusCount)
	if missingStatusCount == 0 {
		log.Println("No test runs missing status. Skipping backfill.")
		return
	}

	const batchSize = 100
	offset := 0

	log.Println("Initiating test run status update .")
	for {
		var testRuns []models.TestRun

		err := db.Preload("SuiteRuns.SpecRuns").
			Where("status IS NULL OR status = ''").
			Limit(batchSize).
			Offset(offset).
			Find(&testRuns).Error

		if err != nil {
			log.Printf("Backfill failed at offset %d: %v\n", offset, err)
			return
		}

		if len(testRuns) == 0 {
			break
		}

		for _, testRun := range testRuns {
			status := "PASSED"

			for _, suite := range testRun.SuiteRuns {
				for _, spec := range suite.SpecRuns {
					if strings.EqualFold(spec.Status, "FAILED") {
						status = "FAILED"
						goto UpdateStatus
					}
					if strings.EqualFold(spec.Status, "SKIPPED") && status != "FAILED" {
						status = "SKIPPED"
					}
				}
			}

		UpdateStatus:
			if err := db.Model(&models.TestRun{}).
				Where("id = ?", testRun.ID).
				Update("status", status).Error; err != nil {
				log.Printf("Failed to update test_run ID %d: %v\n", testRun.ID, err)
			}
		}

		offset += batchSize
	}

	log.Println("Test run status update is complete.")
}
