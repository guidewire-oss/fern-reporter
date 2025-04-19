package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/pkg/models"
	"net/http"
	"strconv"
)

type FavouriteRequest struct {
	Cookie    string   `json:"cookie"`
	Favourite []uint64 `json:"favourite"`
}

type PreferenceRequest struct {
	Cookie   string `json:"cookie"`
	IsDark   bool   `json:"is_dark"`
	Timezone string `json:"timezone"`
}

func (h *Handler) SaveFavourite(c *gin.Context) {
	var favouriteRequest FavouriteRequest

	if err := c.ShouldBindJSON(&favouriteRequest); err != nil {
		fmt.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return // Stop further processing if there is a binding error
	}

	// Check if user exists
	user, err := GetUserObject(h, favouriteRequest.Cookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("User ID not found: %v", err)})
	}

	// check if favourite entry exists for the user
	var count int64
	if err := h.db.Where("user_id = ? and project_id = ?", user.ID, favouriteRequest.Favourite[0]).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "favourite already exists for the user"})
		return
	}

	userPreferredProject := models.PreferredProject{
		UserID:    user.ID,
		ProjectID: favouriteRequest.Favourite[0],
		GroupID:   nil, // ungrouped
	}

	// Save favourite project to DB
	if err := h.db.Save(&userPreferredProject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving record or favourite already saved"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
	})
}

func (h *Handler) DeleteFavourite(c *gin.Context) {
	projectid := c.Param("projectid")
	ucookie := c.Param("ucookie")

	// Check if user exists
	user, err := GetUserObject(h, ucookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("User ID not found: %v", err)})
	}

	projID, err := strconv.ParseUint(projectid, 10, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	userPreferredProject := models.PreferredProject{
		UserID:    user.ID,
		ProjectID: projID,
		GroupID:   nil, // ungrouped
	}

	// Delete favourite from DB
	if err := h.db.Where("user_id = ? and project_id = ?", user.ID, projID).Delete(&userPreferredProject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Favourite Project ID '%s' deleted successfully", projectid)})
}

func (h *Handler) SavePreference(c *gin.Context) {
	var preference PreferenceRequest

	if err := c.ShouldBindJSON(&preference); err != nil {
		fmt.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return // Stop further processing if there is a binding error
	}

	// Save Preference to DB
	result := h.db.Model(&models.AppUser{}).
		Where("cookie = ?", preference.Cookie).
		Updates(models.AppUser{
			IsDark:   &preference.IsDark,
			Timezone: preference.Timezone,
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error updating preference"})
		return
	}

	// If no rows were affected, the cookie didn't exist — optionally create
	if result.RowsAffected == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Not updated, user record not exists"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status": "success",
	})
}

func (h *Handler) GetPreference(c *gin.Context) {
	ucookie := c.Param("ucookie")
	var user models.AppUser

	if err := h.db.Where("cookie = ?", ucookie).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching User"})
		return
	}
	c.JSON(http.StatusOK, user)
}
