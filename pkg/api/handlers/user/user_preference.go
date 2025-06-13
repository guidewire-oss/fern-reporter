package user

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type FavouriteProjectRequest struct {
	Favourite string `json:"favourite"`
}

type UserPreferenceRequest struct {
	IsDark   bool   `json:"is_dark"`
	Timezone string `json:"timezone"`
}

type ProjectGroupsRequest struct {
	ProjectGroups []struct {
		GroupID      uint64   `json:"group_id"` // will be empty for new group
		GroupName    string   `json:"group_name"`
		ProjectUUIDs []string `json:"project_uuids"` // list of project UUIDs
	} `json:"project_groups"`
}

type DeleteProjectGroupRequest struct {
	ProjectGroup []struct {
		GroupID uint64 `json:"group_id"`
	} `json:"project_group"`
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

type UserHandler struct {
	db *gorm.DB
}

// NewProjectHandler initializes ProjectHandler
func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) SaveFavouriteProject(c *gin.Context) {
	var favouriteRequest FavouriteProjectRequest
	ucookie, _ := c.Cookie(utils.CookieName)

	if err := c.ShouldBindJSON(&favouriteRequest); err != nil {
		fmt.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return // Stop further processing if there is a binding error
	}

	// Check if user exists
	user, err := GetUserObject(h, ucookie)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("User ID not found: %v", err)})
	}

	var project models.ProjectDetails
	if err := h.db.Where("uuid = ?", favouriteRequest.Favourite).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Project id %s not found", favouriteRequest.Favourite)})
		return
	}

	// check if favourite entry exists for the user
	var count int64 = 1
	if err := h.db.Table("preferred_projects").Where("user_id = ? and project_id = ?", user.ID, project.ID).Count(&count).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("favourite project %s already configured for the user", favouriteRequest.Favourite)})
		return
	}

	if count <= 0 {
		userPreferredProject := models.PreferredProject{
			UserID:    user.ID,
			ProjectID: project.ID,
			GroupID:   nil, // ungrouped
			User:      user,
			Project:   project,
		}

		// Save favourite project to DB
		if err := h.db.Omit("User", "Project").Save(&userPreferredProject).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving record or favourite already saved"})
			return
		}
		log.Printf("Saved favourite project %s, for the user cookie %s", project.UUID, ucookie)

	}
	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
	})
}

func (h *UserHandler) DeleteFavouriteProject(c *gin.Context) {
	projectUUID := c.Param("projectUUID")
	ucookie, _ := c.Cookie(utils.CookieName)

	// Check if user exists
	user, err := GetUserObject(h, ucookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("User ID not found: %v", err)})
	}

	var project models.ProjectDetails
	if err := h.db.Where("uuid = ?", projectUUID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Project %s not found", projectUUID)})
		return
	}

	userPreferredProject := models.PreferredProject{
		UserID:    user.ID,
		ProjectID: project.ID,
		GroupID:   nil, // ungrouped
	}

	// Delete favourite from DB
	if err := h.db.Where("user_id = ? and project_id = ?", user.ID, project.ID).Delete(&userPreferredProject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting project"})
		return
	}
	log.Printf("favourite project %s deleted successfully for the user cookie %s", project.UUID, ucookie)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Favourite Project %s deleted successfully", project.UUID)})
}

func (h *UserHandler) GetFavouriteProject(c *gin.Context) {
	ucookie, _ := c.Cookie(utils.CookieName)

	user, err := GetUserObject(h, ucookie)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("User ID not found: %v", err)})
		return
	}

	var uuids []string
	err = h.db.
		Table("preferred_projects").
		Joins("JOIN project_details ON preferred_projects.project_id = project_details.id").
		Where("preferred_projects.user_id = ? AND preferred_projects.group_id IS NULL", user.ID).
		Pluck("project_details.uuid", &uuids).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error fetching favourite project uuids"})
		return
	}

	c.JSON(http.StatusOK, uuids)
}

func (h *UserHandler) SaveUserPreference(c *gin.Context) {
	var preference UserPreferenceRequest
	ucookie, _ := c.Cookie(utils.CookieName)

	if err := c.ShouldBindJSON(&preference); err != nil {
		fmt.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return // Stop further processing if there is a binding error
	}

	// Check if user exists
	_, err := GetUserObject(h, ucookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("User ID not found: %v", err)})
	}

	// Save Preference to DB
	result := h.db.Model(&models.AppUser{}).
		Where("cookie = ?", ucookie).
		Updates(models.AppUser{
			IsDark:   preference.IsDark,
			Timezone: preference.Timezone,
		})

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error updating preference"})
		return
	}

	// If no rows were affected, the cookie didn't exist — optionally create
	if result.RowsAffected == 0 {
		log.Printf("Not updated, user record not exists for the cookie %s", ucookie)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Not updated, user record not exists"})
		return
	}

	log.Printf("user preference updated for the cookie %s", ucookie)
	c.JSON(http.StatusAccepted, gin.H{
		"status": "success",
	})
}

func (h *UserHandler) GetUserPreference(c *gin.Context) {
	//ucookie := c.Param("ucookie")
	ucookie, _ := c.Cookie(utils.CookieName)
	var user models.AppUser

	if err := h.db.Where("cookie = ?", ucookie).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching User"})
		return
	}
	c.JSON(http.StatusOK, user)
}
func (h *UserHandler) SaveProjectGroups(c *gin.Context) {
	var projectGroupsRequest ProjectGroupsRequest
	ucookie, _ := c.Cookie(utils.CookieName)

	if err := c.ShouldBindJSON(&projectGroupsRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Remove duplicate project entries if there are any
	var groupIDs []uint64
	for i, group := range projectGroupsRequest.ProjectGroups {
		seen := make(map[string]bool)
		var uniqueProjects []string

		for _, projectUUID := range group.ProjectUUIDs {
			if !seen[projectUUID] {
				seen[projectUUID] = true
				uniqueProjects = append(uniqueProjects, projectUUID)
			}
		}
		projectGroupsRequest.ProjectGroups[i].ProjectUUIDs = uniqueProjects
		if group.GroupID != 0 { // Only consider existing groups (non-zero group_id)
			groupIDs = append(groupIDs, group.GroupID)
		}
	}

	// 1. Find the user
	user, err := GetUserObject(h, ucookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("User ID not found: %v", err)})
	}

	// Begin transaction
	tx := h.db.Begin()

	// 2. Delete only PreferredProjects matching user_id and group_id in the request
	if len(groupIDs) > 0 {
		if err := tx.Where("user_id = ? AND group_id IN ?", user.ID, groupIDs).
			Delete(&models.PreferredProject{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear preferences"})
			return
		}
	}

	// 3. Prepare all new preferred entries
	var preferredEntries []models.PreferredProject

	for _, group := range projectGroupsRequest.ProjectGroups {
		var groupModel models.ProjectGroup

		// Try to find the group first
		err := tx.Where("user_id = ? AND group_id = ?", user.ID, group.GroupID).First(&groupModel).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create if it doesn't exist
			groupModel = models.ProjectGroup{
				GroupID:   group.GroupID,
				UserID:    user.ID,
				GroupName: group.GroupName,
			}
			if err := tx.Create(&groupModel).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create group '%s' ", err)})
				return
			}
		} else if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch group"})
			return
		}

		for _, projectUUID := range group.ProjectUUIDs {
			var project models.ProjectDetails
			if err := tx.Where("uuid = ?", projectUUID).First(&project).Error; err != nil {
				// Optionally skip or log; skipping here
				continue
			}

			preferredEntries = append(preferredEntries, models.PreferredProject{
				UserID:    user.ID,
				ProjectID: project.ID,
				GroupID:   &groupModel.GroupID,
			})
		}
	}

	// 4. Bulk insert preferred entries
	if len(preferredEntries) > 0 {
		if err := tx.Create(&preferredEntries).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save preferred entries"})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	log.Printf("ProjectGroups updated for the Group Ids %v", groupIDs)
	c.JSON(http.StatusCreated, gin.H{"status": "success"})
}

func (h *UserHandler) GetProjectGroups(c *gin.Context) {
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

	// 6. Return response
	response := ProjectGroupResponse{
		Cookie:        user.Cookie,
		ProjectGroups: grouped,
	}

	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) getProjectSummaryMapping(preferred []models.PreferredProject, branchFilter string) (map[string]ProjectSummary, error) {
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

func (h *UserHandler) DeleteProjectGroups(c *gin.Context) {
	var req DeleteProjectGroupRequest
	ucookie, _ := c.Cookie(utils.CookieName)

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Find user by cookie
	var user models.AppUser
	if err := h.db.Where("cookie = ?", ucookie).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Collect group IDs to delete
	var groupIDs []uint64
	for _, pref := range req.ProjectGroup {
		if pref.GroupID != 0 {
			groupIDs = append(groupIDs, pref.GroupID)
		}
	}

	if len(groupIDs) > 0 {
		// Begin transaction
		tx := h.db.Begin()
		// Delete preferred projects for the user and specified group IDs
		if err := tx.Where("user_id = ? AND group_id IN ?", user.ID, groupIDs).
			Delete(&models.PreferredProject{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete preferred projects"})
			return
		}

		// Delete the project_groups
		if err := tx.Where("user_id = ? AND group_id IN ?", user.ID, groupIDs).
			Delete(&models.ProjectGroup{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete project group"})
			return
		}
		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
			return
		}
	}

	log.Printf("Deleted preferred project updated for the Group Ids %v", groupIDs)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func GetUserObject(h *UserHandler, cookie string) (models.AppUser, error) {
	var user models.AppUser
	if err := h.db.Where("cookie = ?", cookie).First(&user).Error; err != nil {
		// Add entry if the user not exists
		log.Printf("User not exists for the cookie %s, creating new user", cookie)
		var user = models.AppUser{
			Cookie:   cookie,
			Timezone: "America/Los_Angeles",
		}
		if err := h.db.Create(&user).Error; err != nil {
			return user, err
		}
		return user, nil
	}
	return user, nil
}
