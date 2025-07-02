package handlers

import (
	"errors"
	"fmt"

	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"

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

	method := c.Request.Method
	path := c.FullPath()

	if err := c.ShouldBindJSON(&testRun); err != nil {
		utils.Log.Warn(fmt.Sprintf("[REQUEST-ERROR]: Invalid JSON payload for %s at %s: %s", method, path, err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return // Stop further processing if there is a binding error
	}

	// Validate that UUID is provided
	if testRun.TestProjectID == "" {
		utils.Log.Warn(fmt.Sprintf("[REQUEST-ERROR]: Missing TestProjectID for %s at %s", method, path))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project UUID is required"})
		return
	}

	gdb := h.db
	isNewRecord := testRun.ID == 0

	//Check if UUID is already exists and get the Project Name
	projectID, err := getProjectIDByUUID(h.db, testRun.TestProjectID)

	if err != nil || projectID == 0 {
		utils.Log.Warn(fmt.Sprintf("[REQUEST-ERROR]: Project ID %s not found for %s at %s", testRun.TestProjectID, method, path))
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Project ID %s not found", testRun.TestProjectID)})
		return
	}
	testRun.ProjectID = projectID

	// If it's not a new record, try to find it first
	if !isNewRecord {
		if err := gdb.Where("id = ?", testRun.ID).First(&models.TestRun{}).Error; err != nil {
			utils.Log.Warn(fmt.Sprintf("[REQUEST-ERROR]: TestRun with ID %d not found for %s at %s", testRun.ID, method, path))
			c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
			return // Stop further processing if record not found
		}
	}

	// Process tags
	err = ProcessTags(gdb, &testRun)
	if err != nil {
		utils.Log.Error(fmt.Sprintf("[ERROR]: Error processing tags for TestRun ID %d: ", testRun.ID), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error processing tags"})
		return // Stop further processing if tag processing fails
	}

	// Save or update the testRun record in the database
	if err := gdb.Save(&testRun).Error; err != nil {
		utils.Log.Error(fmt.Sprintf("[ERROR]: Error saving TestRun ID %d: ", testRun.ID), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving record"})
		return // Stop further processing if save fails
	}

	utils.Log.Info(fmt.Sprintf("[REQUEST-SUCCESS]: TestRun %d created successfully", testRun.ID))
	c.JSON(http.StatusCreated, &testRun)
}

func getProjectIDByUUID(db *gorm.DB, uuid string) (uint64, error) {
	var project models.ProjectDetails
	if err := db.Where("uuid = ?", uuid).First(&project).Error; err != nil {
		return 0, err
	}
	return project.ID, nil
}

func ProcessTags(db *gorm.DB, testRun *models.TestRun) error {
	for i, suite := range testRun.SuiteRuns {
		for j, spec := range suite.SpecRuns {
			var processedTags []models.Tag
			for _, tag := range spec.Tags {
				var existingTag models.Tag

				// Check if the tag already exists
				result := db.Where("name = ?", tag.Name).First(&existingTag)

				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					// If the tag does not exist, create a new one
					newTag := models.Tag{Name: tag.Name}
					if err := db.Create(&newTag).Error; err != nil {
						return err // Return error if tag creation fails
					}
					processedTags = append(processedTags, newTag)
				} else if result.Error != nil {
					// Return error if there is a problem fetching the tag
					return result.Error
				} else {
					// If the tag exists, use the existing tag
					processedTags = append(processedTags, existingTag)
				}
			}
			// Correctly associate the processed tags with the specific spec run
			testRun.SuiteRuns[i].SpecRuns[j].Tags = processedTags
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
		utils.Log.Warn(fmt.Sprintf("[REQUEST-ERROR]: Invalid JSON payload for TestRun ID %s: %s", id, err.Error()))
	}

	db.Save(&testRun)
	c.JSON(http.StatusOK, &testRun)
}

func (h *Handler) DeleteTestRun(c *gin.Context) {
	method := c.Request.Method
	path := c.FullPath()
	var testRun models.TestRun

	id := c.Param("id")
	if testRunID, err := strconv.Atoi(id); err != nil {
		utils.Log.Warn(fmt.Sprintf("[REQUEST-ERROR]: Invalid TestRunID %s for %s at %s: %s", id, method, path, err.Error()))
		c.AbortWithStatus(http.StatusNotFound)
		return
	} else {
		testRun.ID = uint64(testRunID)
	}

	result := h.db.Delete(&testRun)
	if result.Error != nil {
		// If there was an error during the delete operation
		utils.Log.Error(fmt.Sprintf("[ERROR]: Error deleting TestRun ID %d for %s at %s", testRun.ID, method, path), result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error deleting test run"})
		return
	} else if result.RowsAffected == 0 {
		// If no rows were affected, it means no record was found with the provided ID
		utils.Log.Warn(fmt.Sprintf("[REQUEST-ERROR]: TestRun with ID %d not found for %s at %s", testRun.ID, method, path))
		c.JSON(http.StatusNotFound, gin.H{"error": "test run not found"})
		return
	}

	utils.Log.Info(fmt.Sprintf("[REQUEST-SUCCESS]: TestRun %d deleted successfully for %s at %s", testRun.ID, method, path))
	c.JSON(http.StatusOK, &testRun)
}

func (h *Handler) ReportTestRunAll(c *gin.Context) {
	var testRuns []models.TestRun
	h.db.Preload("SuiteRuns.SpecRuns.Tags").Preload("Project").Find(&testRuns)

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

	method := c.Request.Method
	path := c.FullPath()

	startTime, err := ParseTimeFromStringWithDefault(startTimeInput, time.Now().AddDate(-1, 0, 0))
	if err != nil {
		utils.Log.Warn(fmt.Sprintf("[REQUEST-ERROR]: Invalid startTime parameter for %s at %s", method, path))
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid startTime parameter: %v", err)})
	}
	endTime, err := ParseTimeFromStringWithDefault(endTimeInput, time.Now())
	if err != nil {
		utils.Log.Warn(fmt.Sprintf("[REQUEST-ERROR]: Invalid endTime parameter for %s at %s", method, path))
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
	utils.Log.Info("[PING]: Fern Reporter is running!")
	c.JSON(200, gin.H{
		"message": "Fern Reporter is running!",
	})
}
