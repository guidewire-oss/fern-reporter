package summary

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/guidewire/fern-reporter/pkg/models"
	"sort"
)

type SummaryHandler struct {
	db *gorm.DB
}

func NewSummaryHandler(db *gorm.DB) *SummaryHandler {
	return &SummaryHandler{db: db}
}

func (h SummaryHandler) GetSummary(c *gin.Context) {
	projectUUID := c.Param("projectUUID")

	groupBy := c.QueryArray("group_by")
	if len(groupBy) == 0 {
		groupBy = []string{"test_type", "component", "owner", "category"} // default fallback
	}

	var testRun models.TestRun

	query := h.db.
		Model(&models.TestRun{}).
		Joins("Project").
		Where("Project.uuid = ?", projectUUID).
		Preload("Project").
		Preload("SuiteRuns.SpecRuns.Tags").
		Preload("SuiteRuns.Tags")

	seedParam := c.Query("seed")
	if seedParam != "" {
		query = query.Where("test_seed = ?", seedParam)
	} else {
		query = query.Order("start_time desc")
	}

	if err := query.First(&testRun).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "TestRun not found"})
		return
	}

	// Flexible aggregation
	groupCounts := map[string]map[string]int{}
	groupKeyMap := map[string]map[string]string{} // compositeKey -> map of grouping keys

	totalTests := 0
	statusCounts := map[string]int{}

	for _, suite := range testRun.SuiteRuns {
		for _, spec := range suite.SpecRuns {
			totalTests++

			// Build tag map for this spec
			tagMap := map[string]string{}
			for _, tag := range spec.Tags {
				tagMap[tag.Name] = tag.Value
			}

			// Compose dynamic key
			keyParts := []string{}
			keyKV := map[string]string{}
			for _, key := range groupBy {
				value := tagMap[key]
				if value == "" {
					value = "unspecified"
				}
				keyParts = append(keyParts, value)
				keyKV[key] = value
			}
			compositeKey := strings.Join(keyParts, "|")

			if _, ok := groupCounts[compositeKey]; !ok {
				groupCounts[compositeKey] = map[string]int{
					"total":   0,
					"passed":  0,
					"failed":  0,
					"skipped": 0,
					"pending": 0,
				}
				groupKeyMap[compositeKey] = keyKV
			}

			status := spec.Status
			statusCounts[status]++
			groupCounts[compositeKey]["total"]++
			groupCounts[compositeKey][status]++
		}
	}

	overallStatus := "passed"
	if statusCounts["failed"] > 0 {
		overallStatus = "failed"
	}

	// Build summary response
	summary := []map[string]interface{}{}
	for key, counts := range groupCounts {
		entry := map[string]interface{}{}
		for _, tag := range groupBy {
			entry[tag] = groupKeyMap[key][tag]
		}
		for k, v := range counts {
			if v > 0 {
				entry[k] = v
			}
		}
		summary = append(summary, entry)
	}

	// Sorts the summary by the groupBy keys to ensure consistent output order.
	sort.Slice(summary, func(i, j int) bool {
		a, b := summary[i], summary[j]
		for _, key := range groupBy {
			va, oka := a[key].(string)
			vb, okb := b[key].(string)
			if !oka || !okb {
				continue
			}
			if va != vb {
				return va < vb
			}
		}
		// as a final tie‐breaker, sort by JSON encoding
		sa, _ := json.Marshal(a)
		sb, _ := json.Marshal(b)
		return string(sa) < string(sb)
	})

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
