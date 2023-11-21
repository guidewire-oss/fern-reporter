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
		testRun.Id = testRunID
	}

	db.GetDb().Delete(&testRun)
	c.JSON(http.StatusOK, &testRun)
}
