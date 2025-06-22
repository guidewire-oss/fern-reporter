package handlers

import (
	"errors"
	"fmt"
	"sort"
	"strings"

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

type ProjectSummary struct {
	UUID        string               `json:"uuid"`
	Name        string               `json:"name"`
	Status      models.TestRunStatus `json:"status"`
	TestCount   uint64               `json:"test_count"`
	TestPassed  uint64               `json:"test_passed"`
	TestFailed  uint64               `json:"test_failed"`
	TestSkipped uint64               `json:"test_skipped"`
	Date        time.Time            `json:"date"`
	GitBranch   string               `json:"git_branch"`
}

type ProjectGroup struct {
	GroupID   uint64           `json:"group_id"`
	GroupName string           `json:"group_name"`
	Projects  []ProjectSummary `json:"projects"`
}

type ProjectGroupResponse struct {
	Cookie        string         `json:"cookie"`
	ProjectGroups []ProjectGroup `json:"project_groups"`
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

	computeTestRunStatus(&testRun)

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

func computeTestRunStatus(testRun *models.TestRun) {
	status := models.StatusPassed

	for _, suite := range testRun.SuiteRuns {
		for _, spec := range suite.SpecRuns {
			if strings.EqualFold(spec.Status, "FAILED") {
				testRun.Status = models.StatusFailed
				return
			}
			if strings.EqualFold(spec.Status, "SKIPPED") {
				status = models.StatusSkipped
			}
		}
	}

	testRun.Status = status
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

func (h *Handler) GetProjectGroups(c *gin.Context) {
	ucookie, _ := c.Cookie(utils.CookieName)

	// 1. Get the user
	var user models.AppUser
	if err := h.db.Where("cookie = ?", ucookie).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error fetching user"})
		}
		return
	}

	groupIDStr := c.Query("group_id")
	branch := c.Query("git_branch")

	// 2. Get preferred projects
	var preferred []models.PreferredProject
	query := h.db.Preload("Project").
		Preload("Group").
		Where("user_id = ?", user.ID)

	if groupIDStr != "" {
		if groupID, err := strconv.ParseUint(groupIDStr, 10, 64); err == nil {
			query = query.Where("group_id = ?", groupID)
		}
	}

	err := query.Find(&preferred).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error fetching preferences"})
		return
	}

	// 3. Get project summaries (filtering by branch if given)
	projectSummaryMap, err := h.getProjectSummaryMapping(preferred, branch)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. Group by group ID
	groupMap := make(map[uint64]*ProjectGroup)
	for _, item := range preferred {
		if item.Group == nil {
			continue // skip ungrouped
		}

		groupID := item.Group.GroupID
		if _, exists := groupMap[groupID]; !exists {
			groupMap[groupID] = &ProjectGroup{
				GroupID:   groupID,
				GroupName: item.Group.GroupName,
				Projects:  []ProjectSummary{},
			}
		}

		if summary, ok := projectSummaryMap[item.Project.UUID]; ok {
			groupMap[groupID].Projects = append(groupMap[groupID].Projects, summary)
		}
	}

	// 5. Convert map to slice
	var grouped []ProjectGroup
	for _, group := range groupMap {
		grouped = append(grouped, *group)
	}

	sort.Slice(grouped, func(i, j int) bool {
		return grouped[i].GroupID < grouped[j].GroupID
	})

	// 6. Return response
	response := ProjectGroupResponse{
		Cookie:        user.Cookie,
		ProjectGroups: grouped,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) getProjectSummaryMapping(preferred []models.PreferredProject, branchFilter string) (map[string]ProjectSummary, error) {
	projectSummaryMap := make(map[string]ProjectSummary)

	for _, pref := range preferred {
		var testRun models.TestRun
		query := h.db.Model(&models.TestRun{})

		query = query.Preload("SuiteRuns.SpecRuns").
			Joins("JOIN project_details ON project_details.id = test_runs.project_id").
			Where("project_details.uuid = ?", pref.Project.UUID)

		if branchFilter != "" {
			query = query.Where("test_runs.git_branch = ?", branchFilter)
		}

		query = query.Order("test_runs.end_time desc")

		if err := query.First(&testRun).Error; err != nil {
			return nil, fmt.Errorf("error fetching test run for project %s: %w", pref.Project.UUID, err)
		}

		summary := ProjectSummary{
			UUID:      pref.Project.UUID,
			Name:      pref.Project.Name,
			Status:    testRun.Status,
			Date:      testRun.EndTime,
			GitBranch: testRun.GitBranch,
		}

		computeTestStatusCount(&summary, testRun)

		projectSummaryMap[pref.Project.UUID] = summary
	}

	return projectSummaryMap, nil
}

func computeTestStatusCount(summary *ProjectSummary, testRun models.TestRun) {
	for _, suite := range testRun.SuiteRuns {
		for _, spec := range suite.SpecRuns {
			switch strings.ToUpper(spec.Status) {
			case "PASSED":
				summary.TestPassed++
			case "SKIPPED":
				summary.TestSkipped++
			case "FAILED":
				summary.TestFailed++
			}
			summary.TestCount++
		}
	}
}

func (h *Handler) Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Fern Reporter is running!",
	})
}
