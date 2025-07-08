package project

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"
	"gorm.io/gorm"
)

type ProjectHandler struct {
	db *gorm.DB
}

// NewProjectHandler initializes ProjectHandler
func NewProjectHandler(db *gorm.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var project models.ProjectDetails

	method := c.Request.Method
	path := c.FullPath()

	if err := c.ShouldBindJSON(&project); err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("[REQUEST-ERROR]: Invalid JSON payload for %s at %s", method, path))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	project.Name = strings.TrimSpace(project.Name)

	if err := h.db.Where("name = ?", project.Name).First(&project).Error; err == nil {
		utils.GetLogger().Warn(fmt.Sprintf("[REQUEST-ERROR]: Project Name %s already exists for %s at %s", project.Name, method, path))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project Name already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		utils.GetLogger().Error(fmt.Sprintf("[ERROR]: Failed to query project name for %s at %s", method, path), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query project name"})
		return
	}

	if err := h.db.Clauses(clause.Returning{}).Create(&project).Error; err != nil {
		utils.GetLogger().Error(fmt.Sprintf("[ERROR]: Failed to create project %s for %s at %s", project.UUID, method, path), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}
	utils.GetLogger().Info(fmt.Sprintf("[REQUEST-SUCCESS]: Project %s created successfully for %s at %s", project.UUID, method, path))
	c.JSON(http.StatusCreated, project)
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	id := c.Param("uuid")
	var existing, project models.ProjectDetails

	method := c.Request.Method
	path := c.FullPath()

	if err := c.ShouldBindJSON(&project); err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("[REQUEST-ERROR]: Invalid JSON payload for %s at %s", method, path))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Where("uuid = ?", id).First(&existing).Error; err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("[REQUEST-ERROR]: Project with UUID %s not found for %s at %s", id, method, path))
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}
	project.Name = strings.TrimSpace(project.Name)

	// Check for name uniqueness excluding current UUID
	var count int64
	if err := h.db.Model(&models.ProjectDetails{}).
		Where("name = ? AND uuid != ?", project.Name, id).
		Count(&count).Error; err != nil {
		utils.GetLogger().Error(fmt.Sprintf("[ERROR]: Database error for Project %s while checking for duplicates for %s at %s", project.UUID, method, path), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while checking for duplicates"})
		return
	}
	if count > 0 {
		utils.GetLogger().Warn(fmt.Sprintf("[REQUEST-ERROR]: Project %s already exists for %s at %s", project.UUID, method, path))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name already exists"})
		return
	}

	// Copy immutable fields from existing to ensure GORM updates instead of inserts
	project.ID = existing.ID
	project.UUID = existing.UUID
	project.CreatedAt = existing.CreatedAt

	if err := h.db.Save(&project).Error; err != nil {
		utils.GetLogger().Error(fmt.Sprintf("[ERROR]: Failed to update project %s for %s at %s", project.UUID, method, path), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	utils.GetLogger().Info(fmt.Sprintf("[REQUEST-SUCCESS]: Project %s updated successfully for %s at %s", project.UUID, method, path))
	c.JSON(http.StatusOK, project)
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	uuid := c.Param("uuid")

	method := c.Request.Method
	path := c.FullPath()

	if err := h.db.Where("uuid = ?", uuid).Delete(&models.ProjectDetails{}).Error; err != nil {
		utils.GetLogger().Error(fmt.Sprintf("[ERROR]: Failed to delete project %s for %s at %s", uuid, method, path), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	utils.GetLogger().Info(fmt.Sprintf("[REQUEST-SUCCESS]: Project %s deleted successfully for %s at %s", uuid, method, path))
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Project ID %s deleted", uuid)})
}

func (h *ProjectHandler) GetAllProjects(c *gin.Context) {
	var projects []models.ProjectDetails

	method := c.Request.Method
	path := c.FullPath()

	if err := h.db.Order("name ASC").Find(&projects).Error; err != nil {
		utils.GetLogger().Error(fmt.Sprintf("[ERROR]: Failed to fetch projects for %s at %s", method, path), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching projects"})
		return
	}

	c.JSON(http.StatusOK, projects)
}

func (h *ProjectHandler) GetAllProjectsForReport(c *gin.Context) {
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
