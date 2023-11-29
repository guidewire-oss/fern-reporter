package handlers

import (
	"fern-reporter/pkg/models"
	"net/http"
	"strconv"

	"fern-reporter/pkg/db"

	"github.com/gin-gonic/gin"
)

func CreateTestRun(c *gin.Context) {
	var testRun models.TestRun

	if err := c.ShouldBindJSON(&testRun); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return // Stop further processing if there is a binding error
	}

	gdb := db.GetDb()
	if testRun.ID != 0 {
		// Check if a record with the given ID already exists
		if err := gdb.Where("id = ?", testRun.ID).First(&testRun).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
			return // Stop further processing if record not found
		}
	}

	// Save the testRun record to the database
	if err := gdb.Save(&testRun).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving record"})
		return // Stop further processing if save fails
	}

	c.JSON(http.StatusCreated, &testRun) // Send successful response
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
