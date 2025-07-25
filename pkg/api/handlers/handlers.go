package handlers

import (
	"errors"
	"fmt"
	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"
	"strings"

	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) CreateTestRun(c *gin.Context) {
	var testRun models.TestRun

	if err := c.ShouldBindJSON(&testRun); err != nil {
		fmt.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return // Stop further processing if there is a binding error
	}

	// Validate that UUID is provided
	if testRun.TestProjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project UUID is required"})
		return
	}

	gdb := h.db
	isNewRecord := testRun.ID == 0

	//Check if UUID is already exists and get the Project Name
	projectID, err := getProjectIDByUUID(h.db, testRun.TestProjectID)

	if err != nil || projectID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Project ID %s not found", testRun.TestProjectID)})
		return
	}
	testRun.ProjectID = projectID

	// If it's not a new record, try to find it first
	if !isNewRecord {
		if err := gdb.Where("id = ?", testRun.ID).First(&models.TestRun{}).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
			return // Stop further processing if record not found
		}
	}

	// Process tags
	err = ProcessTags(gdb, &testRun)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error processing tags"})
		return // Stop further processing if tag processing fails
	}

	// Save or update the testRun record in the database
	if err := gdb.Save(&testRun).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving record"})
		return // Stop further processing if save fails
	}

	c.JSON(http.StatusCreated, &testRun)
}
func getProjectIDByUUID(db *gorm.DB, uuid string) (uint64, error) {
	var project models.ProjectDetails
	if err := db.Where("uuid = ?", uuid).First(&project).Error; err != nil {
		return 0, err
	}
	return project.ID, nil
}

// GetOrCreateTag checks if a tag exists by name, creates it if not, and returns the tag.
func GetOrCreateTag(db *gorm.DB, tagName string) (models.Tag, error) {
	var existingTag models.Tag
	result := db.Where("name = ?", tagName).First(&existingTag)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		newTag := models.Tag{Name: tagName}
		if err := db.Create(&newTag).Error; err != nil {
			log.Printf("failed to create tag %s: %v", tagName, err)
			return models.Tag{}, err
		}
		return newTag, nil
	} else if result.Error != nil {
		return models.Tag{}, result.Error
	}
	return existingTag, nil
}

func ProcessTags(db *gorm.DB, testRun *models.TestRun) error {
	processTagList := func(tags []models.Tag) ([]models.Tag, error) {
		result := make([]models.Tag, 0, len(tags))
		for _, t := range tags {
			tag, err := GetOrCreateTag(db, t.Name)
			if err != nil {
				return nil, err
			}
			result = append(result, tag)
		}
		return result, nil
	}

	for i, suite := range testRun.SuiteRuns {
		// Process suite-level tags
		suiteTags, err := processTagList(suite.Tags)
		if err != nil {
			return err
		}
		testRun.SuiteRuns[i].Tags = suiteTags

		// Process spec-level tags
		for j, spec := range suite.SpecRuns {
			specTags, err := processTagList(spec.Tags)
			if err != nil {
				return err
			}
			testRun.SuiteRuns[i].SpecRuns[j].Tags = specTags
		}
	}
	return nil
}

func (h *Handler) GetTestRunAll(c *gin.Context) {
	var testRuns []models.TestRun
	h.db.Preload("Project").Find(&testRuns)
	c.JSON(http.StatusOK, testRuns)
}

func (h *Handler) GetTestRunByID(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")
	h.db.Preload("Project").Where("id = ?", id).First(&testRun)
	c.JSON(http.StatusOK, testRun)

}

func (h *Handler) UpdateTestRun(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")

	db := h.db
	if err := db.Where("id = ?", id).First(&testRun).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	err := c.BindJSON(&testRun)
	if err != nil {
		log.Printf("error binding json: %v", err)
	}

	db.Save(&testRun)
	c.JSON(http.StatusOK, &testRun)
}

func (h *Handler) DeleteTestRun(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")
	if testRunID, err := strconv.Atoi(id); err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	} else {
		testRun.ID = uint64(testRunID)
	}

	result := h.db.Delete(&testRun)
	if result.Error != nil {
		// If there was an error during the delete operation
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error deleting test run"})
		return
	} else if result.RowsAffected == 0 {
		// If no rows were affected, it means no record was found with the provided ID
		c.JSON(http.StatusNotFound, gin.H{"error": "test run not found"})
		return
	}

	c.JSON(http.StatusOK, &testRun)
}

func (h *Handler) ReportTestRunAll(c *gin.Context) {
	var testRuns []models.TestRun

	filter, err := NewTestRunFilterFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := h.db.Preload("SuiteRuns.SpecRuns.Tags").Preload("Project")

	// Optional Filters
	if filter.ProjectID != "" {
		query = query.Where("project_id = ?", filter.ProjectID)
	}
	if filter.GitBranch != "" {
		query = query.Where("git_branch = ?", filter.GitBranch)
	}
	if filter.GitSha != "" {
		query = query.Where("git_sha LIKE ?", filter.GitSha+"%")
	}
	if filter.StartTime != nil {
		query = query.Where("start_time >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("end_time < ?", *filter.EndTime)
	}

	// Filter by tags, if provided
	if len(filter.Tags) > 0 {
		var testRunIDs []uint
		if err := h.db.
			Table("test_runs").
			Joins("JOIN suite_runs ON suite_runs.test_run_id = test_runs.id").
			Joins("JOIN spec_runs ON spec_runs.suite_id = suite_runs.id").
			Joins("JOIN spec_run_tags ON spec_run_tags.spec_run_id = spec_runs.id").
			Joins("JOIN tags ON tags.id = spec_run_tags.tag_id").
			Where("tags.name IN ?", filter.Tags).
			Distinct("test_runs.id").
			Scan(&testRunIDs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		query = query.Where("test_runs.id IN ?", testRunIDs)
	}

	if err := query.Find(&testRuns).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"testRuns":     testRuns,
		"reportHeader": config.GetHeaderName(),
		"total":        len(testRuns),
	})
}

func (h *Handler) ReportTestRunById(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")
	h.db.Preload("SuiteRuns.SpecRuns.Tags").Preload("Project").Where("id = ?", id).First(&testRun)

	c.JSON(http.StatusOK, gin.H{
		"reportHeader": config.GetHeaderName(),
		"testRuns":     []models.TestRun{testRun},
	})
}

func (h *Handler) ReportTestRunAllHTML(c *gin.Context) {
	var testRuns []models.TestRun
	h.db.Preload("SuiteRuns.SpecRuns.Tags").Find(&testRuns)
	totalTests, executedTests, passedTests, failedTests := utils.CalculateTestMetrics(testRuns)

	c.HTML(http.StatusOK, "test_runs.html", gin.H{
		"reportHeader":  config.GetHeaderName(),
		"testRuns":      testRuns,
		"totalTests":    totalTests,
		"executedTests": executedTests,
		"passedTests":   passedTests,
		"failedTests":   failedTests,
	})
}

func (h *Handler) ReportTestRunByIdHTML(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")
	h.db.Preload("SuiteRuns.SpecRuns").Where("id = ?", id).First(&testRun)
	testRuns := []models.TestRun{testRun}
	totalTests, executedTests, passedTests, failedTests := utils.CalculateTestMetrics(testRuns)

	c.HTML(http.StatusOK, "test_runs.html", gin.H{
		"reportHeader":  config.GetHeaderName(),
		"testRuns":      []models.TestRun{testRun},
		"totalTests":    totalTests,
		"executedTests": executedTests,
		"passedTests":   passedTests,
		"failedTests":   failedTests,
	})
}

func (h *Handler) ReportTestInsights(c *gin.Context) {
	projectName := c.Param("name")
	startTimeInput := c.Query("startTime")
	endTimeInput := c.Query("endTime")

	startTime, err := ParseTimeFromStringWithDefault(startTimeInput, time.Now().AddDate(-1, 0, 0))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid startTime parameter: %v", err)})
	}
	endTime, err := ParseTimeFromStringWithDefault(endTimeInput, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid endTimeInput parameter: %v", err)})
	}

	longestTestRuns := GetLongestTestRuns(h, projectName, startTime, endTime)
	numTests := len(longestTestRuns)
	if len(longestTestRuns) > 10 {
		longestTestRuns = longestTestRuns[:10] //only send top 10 longest runs to display
	}

	averageDuration := GetAverageDuration(h, projectName, startTime, endTime)
	fmt.Printf("longestTestRuns: %v\n", longestTestRuns)
	fmt.Printf("averageDuration: %v\n", averageDuration)

	c.HTML(http.StatusOK, "insights.html", gin.H{
		"reportHeader":    config.GetHeaderName(),
		"projectName":     projectName,
		"startTime":       startTime,
		"endTime":         endTime,
		"averageDuration": averageDuration,
		"longestTestRuns": longestTestRuns,
		"numTests":        numTests,
	})
}

func (h *Handler) GetTestSummary(c *gin.Context) {
	projectId := c.Param("projectId")
	testSummaries := GetProjectSpecStatistics(h, projectId)

	c.JSON(http.StatusOK, testSummaries)
}

func (h *Handler) Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Fern Reporter is running!",
	})
}

func NewTestRunFilterFromQuery(c *gin.Context) (*models.TestQueryFilter, error) {
	var filter models.TestQueryFilter

	// Allowed query parameters
	allowedParams := map[string]bool{
		"project_id": true,
		"git_branch": true,
		"git_sha":    true,
		"start_time": true,
		"end_time":   true,
		"tags":       true,
	}

	// Validate that no unexpected query params are present
	for key := range c.Request.URL.Query() {
		if !allowedParams[key] {
			return nil, fmt.Errorf("invalid query parameter: '%s'", key)
		}
	}

	filter.ProjectID = c.Query("project_id")
	filter.GitBranch = c.Query("git_branch")
	filter.GitSha = c.Query("git_sha")

	if start := c.Query("start_time"); start != "" {
		t, err := time.Parse("2006-01-02", start)
		if err != nil {
			return nil, fmt.Errorf("invalid start_time format (expected YYYY-MM-DD): %v", err)
		}
		filter.StartTime = &t
	}

	if end := c.Query("end_time"); end != "" {
		t, err := time.Parse("2006-01-02", end)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time format (expected YYYY-MM-DD): %v", err)
		}
		filter.EndTime = &t
	}

	if filter.StartTime != nil && filter.EndTime != nil && filter.StartTime.After(*filter.EndTime) {
		return nil, fmt.Errorf("start_time must be before or equal to end_time")
	}

	if tags := c.Query("tags"); tags != "" {
		filter.Tags = strings.Split(tags, ",")
	}

	return &filter, nil
}
