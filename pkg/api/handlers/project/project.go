package project

import (
	"errors"
	"fmt"
	"gorm.io/gorm/clause"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/pkg/models"
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

	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	project.Name = strings.TrimSpace(project.Name)

	if err := h.db.Where("name = ?", project.Name).First(&project).Error; err == nil {
		log.Printf("Project %s already exists", project.Name)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project Name already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Failed to query project name: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query project name"})
		return
	}

	if err := h.db.Clauses(clause.Returning{}).Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}
	log.Printf("Created project, Name: %s, UUID: %s", project.Name, project.UUID)
	c.JSON(http.StatusCreated, project)
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	id := c.Param("uuid")
	var existing, project models.ProjectDetails

	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Where("uuid = ?", id).First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}
	project.Name = strings.TrimSpace(project.Name)

	// Check for name uniqueness excluding current UUID
	var count int64
	if err := h.db.Model(&models.ProjectDetails{}).
		Where("name = ? AND uuid != ?", project.Name, id).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while checking for duplicates"})
		return
	}
	if count > 0 {
		log.Printf("Project %s already exists", project.Name)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name already exists"})
		return
	}

	// Copy immutable fields from existing to ensure GORM updates instead of inserts
	project.ID = existing.ID
	project.UUID = existing.UUID
	project.CreatedAt = existing.CreatedAt

	if err := h.db.Save(&project).Error; err != nil {
		log.Printf("Failed to update project: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	log.Printf("Updated project, Name: %s, UUID: %s", project.Name, project.UUID)
	c.JSON(http.StatusOK, project)
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	uuid := c.Param("uuid")

	if err := h.db.Where("uuid = ?", uuid).Delete(&models.ProjectDetails{}).Error; err != nil {
		log.Printf("Failed to delete project: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	log.Printf("Deleted project, UUID: %s", uuid)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Project ID %s deleted", uuid)})
}

func (h *ProjectHandler) GetAllProjects(c *gin.Context) {
	var projects []models.ProjectDetails

	if err := h.db.Order("name ASC").Find(&projects).Error; err != nil {
		log.Printf("Error fetching projects: %s", err.Error())
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
