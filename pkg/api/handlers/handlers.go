package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/guidewire/fern-reporter/pkg/models"

	"github.com/guidewire/fern-reporter/pkg/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateTestRun(c *gin.Context) {
	var testRun models.TestRun

	if err := c.ShouldBindJSON(&testRun); err != nil {
		fmt.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return // Stop further processing if there is a binding error
	}

	gdb := db.GetDb()
	isNewRecord := testRun.ID == 0

	// If it's not a new record, try to find it first
	if !isNewRecord {
		if err := gdb.Where("id = ?", testRun.ID).First(&models.TestRun{}).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
			return // Stop further processing if record not found
		}
	}

	// Process tags
	err := ProcessTags(gdb, &testRun)
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

func GetTestRunAll(c *gin.Context) {
	var testRuns []models.TestRun
	db.GetDb().Find(&testRuns)
	c.JSON(http.StatusOK, testRuns)
}

func GetTestRunByID(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")
	db.GetDb().Where("id = ?", id).First(&testRun)
	c.JSON(http.StatusOK, testRun)
}

func UpdateTestRun(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")

	db := db.GetDb()
	if err := db.Where("id = ?", id).First(&testRun).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.BindJSON(&testRun)
	db.Save(&testRun)
	c.JSON(http.StatusOK, &testRun)
}

func DeleteTestRun(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")
	if testRunID, err := strconv.Atoi(id); err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	} else {
		testRun.ID = uint64(testRunID)
	}

	result := db.GetDb().Delete(&testRun)
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

func ReportTestRunAll(c *gin.Context) {
	var testRuns []models.TestRun
	db.GetDb().Preload("SuiteRuns.SpecRuns").Find(&testRuns)
	c.HTML(http.StatusOK, "test_runs.html", gin.H{
		"testRuns": testRuns,
	})
}

func ReportTestRunById(c *gin.Context) {
	var testRun models.TestRun
	id := c.Param("id")
	db.GetDb().Preload("SuiteRuns.SpecRuns").Where("id = ?", id).First(&testRun)
	c.HTML(http.StatusOK, "test_runs.html", gin.H{
		"testRuns": []models.TestRun{testRun},
	})
}
