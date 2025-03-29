package handlers

import (
	"errors"
	"fmt"

	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"

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
	h.db.Find(&testRuns)
	c.JSON(http.StatusOK, testRuns)
}

func (h *Handler) GetTestRunByID(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")
	h.db.Where("id = ?", id).First(&testRun)
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
		log.Fatalf("error binding json: %v", err)
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
	h.db.Preload("SuiteRuns.SpecRuns.Tags").Find(&testRuns)

	c.JSON(http.StatusOK, gin.H{
		"testRuns":     testRuns,
		"reportHeader": config.GetHeaderName(),
		"total":        len(testRuns),
	})
}

func (h *Handler) ReportTestRunById(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")
	h.db.Preload("SuiteRuns.SpecRuns").Where("id = ?", id).First(&testRun)

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

func (h *Handler) GetProjectAll(c *gin.Context) {
	var projects []struct {
		ID   uint64 `json:"id"`
		Name string `json:"name"`
		UUID string `json:"uuid"`
	}
	h.db.Table("project_details").
		Order("name ASC").
		Find(&projects)

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
	})
}

func (h *Handler) CreateProject(c *gin.Context) {
	var project models.ProjectDetails
	var existingProject models.ProjectDetails

	if err := c.ShouldBindJSON(&project); err != nil {
		fmt.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return // Stop further processing if there is a binding error

	}
	gdb := h.db
	// Check if a project with the same name already exists
	if err := gdb.Where("name = ?", project.Name).First(&existingProject).Error; err == nil {
		// If no error, it means the project exists
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Project name '%s' already exists", project.Name)})
		return
	} else if err := gdb.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving record"})
		return // Stop further processing if save fails
	}

	gdb.First(&project, project.ID)
	c.JSON(http.StatusCreated, &project)
}

func (h *Handler) GetAllProjects(c *gin.Context) {
	var projects []models.ProjectDetails

	gdb := h.db
	if err := gdb.Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching projects"})
		return
	}

	c.JSON(http.StatusOK, projects)
}

func (h *Handler) UpdateProject(c *gin.Context) {
	uuid := c.Param("uuid") // Get UUID from URL parameter
	var project models.ProjectDetails

	gdb := h.db

	// Check if project with given UUID exists
	if err := gdb.Where("uuid = ?", uuid).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Project with UUID '%s' not found", uuid)})
		return
	}

	// Bind request JSON to project struct
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Manually update the updated_at timestamp
	project.UpdatedAt = time.Now()

	// Save updated project details
	if err := gdb.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project updated successfully", "project": project})
}

func (h *Handler) DeleteProject(c *gin.Context) {
	uuid := c.Param("uuid") // Get UUID from URL parameter
	var project models.ProjectDetails

	gdb := h.db

	// Check if project exists
	if err := gdb.Where("uuid = ?", uuid).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Project with UUID '%s' not found", uuid)})
		return
	}

	// Delete project from DB
	if err := gdb.Delete(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Project with UUID '%s' deleted successfully", uuid)})
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
