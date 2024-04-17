package routers

import (
	"github.com/guidewire/fern-reporter/pkg/api/handlers"
	"github.com/guidewire/fern-reporter/pkg/db"

	"github.com/gin-gonic/gin"
)

func RegisterRouters(router *gin.Engine) {
	// router.GET("/", handlers.Home)
	handler := handlers.NewHandler(db.GetDb())

	api := router.Group("/api")
	{
		testRun := api.Group("/testrun/")
		testRun.GET("/", handler.GetTestRunAll)
		testRun.GET("/:id", handler.GetTestRunByID)
		testRun.POST("/", handler.CreateTestRun)
		testRun.PUT("/:id", handler.UpdateTestRun)
		testRun.DELETE("/:id", handler.DeleteTestRun)
	}
	reports := router.Group("/reports/testruns")
	{
		testRunReport := reports.GET("/", handler.ReportTestRunAll)
		testRunReport.GET("/:id", handler.ReportTestRunById)
	}
	ping := router.Group("/ping")
	{
		ping.GET("/", handler.Ping)
	}
}
