package summary

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/guidewire/fern-reporter/pkg/models"
)

type SummaryHandler struct {
	db *gorm.DB
}

func NewSummaryHandler(db *gorm.DB) *SummaryHandler {
	return &SummaryHandler{db: db}
}

func (h SummaryHandler) GetSummary(c *gin.Context) {
	projectUUID := c.Param("projectUUID")

	var testRun models.TestRun

	query := h.db.
		Model(&models.TestRun{}). // ← CRITICAL
		Joins("Project").
		Where("Project.uuid = ?", projectUUID).
		Preload("Project").
		Preload("SuiteRuns.SpecRuns.Tags").
		Preload("SuiteRuns.Tags")

	seedParam := c.Query("seed")
	if seedParam != "" {
		seed, err := strconv.ParseUint(seedParam, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid seed"})
			return
		}
		query = query.Where("test_seed = ?", seed)
	} else {
		query = query.Order("start_time desc")
	}

	if err := query.First(&testRun).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "TestRun not found"})
		return
	}

	//sort by spec run id
	for i := range testRun.SuiteRuns {
		sort.Slice(testRun.SuiteRuns[i].SpecRuns, func(a, b int) bool {
			return testRun.SuiteRuns[i].SpecRuns[a].ID < testRun.SuiteRuns[i].SpecRuns[b].ID
		})
	}

	// Sort and print
	pretty, _ := json.MarshalIndent(testRun, "", "  ")
	fmt.Printf("TestRun after sorting:\n%s\n", pretty)

	// Aggregation
	statusCounts := map[string]int{}
	groups := map[string]map[string]map[string]map[string]int{} // test_type → component → owner → category data

	totalTests := 0

	for _, suite := range testRun.SuiteRuns {
		for _, spec := range suite.SpecRuns {
			totalTests++

			var testType, component, owner, category string
			for _, tag := range spec.Tags {
				switch tag.Name {
				case "test_type":
					testType = tag.Value
				case "component":
					component = tag.Value
				case "owner":
					owner = tag.Value
				case "category":
					category = tag.Value
				}
			}

			if testType == "" {
				testType = "unspecified"
			}
			if component == "" {
				component = "unspecified"
			}
			if owner == "" {
				owner = "unspecified"
			}
			if category == "" {
				category = "unspecified"
			}

			if _, ok := groups[testType]; !ok {
				groups[testType] = map[string]map[string]map[string]int{}
			}
			if _, ok := groups[testType][component]; !ok {
				groups[testType][component] = map[string]map[string]int{}
			}
			if _, ok := groups[testType][component][owner]; !ok {
				groups[testType][component][owner] = map[string]int{
					"total":   0,
					"passed":  0,
					"failed":  0,
					"skipped": 0,
					"pending": 0,
				}
			}

			status := spec.Status
			statusCounts[status]++
			groups[testType][component][owner]["total"]++
			groups[testType][component][owner][status]++
			// Save category as "virtual key"
			groups[testType][component][owner]["__category__"] = intFromCategory(category)
		}
	}

	overallStatus := "passed"
	if statusCounts["failed"] > 0 {
		overallStatus = "failed"
	}

	// Construct summary section
	var summary []map[string]interface{}
	for testType, comps := range groups {
		for component, owners := range comps {
			for owner, stats := range owners {
				category := categoryFromInt(stats["__category__"])
				entry := map[string]interface{}{
					"test_type": testType,
					"component": component,
					"owner":     owner,
					"category":  category,
					"total":     stats["total"],
				}
				// only include if > 0
				if stats["passed"] > 0 {
					entry["passed"] = stats["passed"]
				}
				if stats["failed"] > 0 {
					entry["failed"] = stats["failed"]
				}
				if stats["skipped"] > 0 {
					entry["skipped"] = stats["skipped"]
				}
				if stats["pending"] > 0 {
					entry["pending"] = stats["pending"]
				}
				summary = append(summary, entry)
			}
		}
	}

	response := map[string]interface{}{
		"head": map[string]interface{}{
			"project_id": testRun.Project.UUID,
			"seed":       testRun.TestSeed,
			"branch":     testRun.GitBranch,
			"sha":        testRun.GitSha,
			"status":     overallStatus,
			"tests":      totalTests,
			"start_time": testRun.StartTime.Format(time.RFC3339),
			"end_time":   testRun.EndTime.Format(time.RFC3339),
		},
		"summary": summary,
	}

	c.JSON(http.StatusOK, response)
}

// Converts category string to int for map use
func intFromCategory(category string) int {
	// Optional: use a real map for category values
	switch category {
	case "infrastructure":
		return 1
	case "atmos-ng":
		return 2
	default:
		return 0
	}
}

func categoryFromInt(i int) string {
	switch i {
	case 1:
		return "infrastructure"
	case 2:
		return "atmos-ng"
	default:
		return "unspecified"
	}
}
