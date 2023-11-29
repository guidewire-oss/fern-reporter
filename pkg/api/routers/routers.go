package routers

import (
	"fern-reporter/pkg/api/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterRouters(router *gin.Engine) {
	// router.GET("/", handlers.Home)
	api := router.Group("/api")
	{
		testRun := api.Group("/testrun")
		testRun.GET("/", handlers.GetTestRunAll)
		testRun.GET("/:id", handlers.GetTestRunByID)
		testRun.POST("/", handlers.CreateTestRun)
		testRun.PUT("/:id", handlers.UpdateTestRun)
		testRun.DELETE("/:id", handlers.DeleteTestRun)
	}
	reports := router.Group("/reports/testruns")
	{
		testRunReport := reports.GET("/", handlers.ReportTestRunAll)
		testRunReport.GET("/:id", handlers.ReportTestRunById)
	}
}
