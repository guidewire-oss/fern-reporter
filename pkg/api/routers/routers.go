package routers

import (
	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/api/handlers"
	"github.com/guidewire/fern-reporter/pkg/api/handlers/project"
	"github.com/guidewire/fern-reporter/pkg/api/handlers/user"
	"github.com/guidewire/fern-reporter/pkg/auth"
	"github.com/guidewire/fern-reporter/pkg/db"

	"github.com/gin-gonic/gin"
)

var (
	testRun *gin.RouterGroup
)

func RegisterRouters(router *gin.Engine) {
	handler := handlers.NewHandler(db.GetDb())
	userHandler := user.NewUserHandler(db.GetDb())
	projectHandler := project.NewProjectHandler(db.GetDb())

	authEnabled := config.GetAuth().Enabled

	var api *gin.RouterGroup
	if authEnabled {
		api = router.Group("/api", auth.ScopeMiddleware())
	} else {
		api = router.Group("/api")
	}

	api.Use()
	{
		testRun = api.Group("/testrun/")
		testRun.GET("/", handler.GetTestRunAll)
		testRun.GET("/:id", handler.GetTestRunByID)
		testRun.POST("/", handler.CreateTestRun)
		testRun.PUT("/:id", handler.UpdateTestRun)
		testRun.DELETE("/:id", handler.DeleteTestRun)
		testRun.GET("/project-groups", handler.GetProjectGroupsSummary)

		testReport := api.Group("/reports")
		testReport.GET("/projects/", projectHandler.GetAllProjectsForReport)
		testReport.GET("/summary/:projectId/", handler.GetTestSummary)
		testReport.GET("/testruns/", handler.ReportTestRunAll)
		testReport.GET("/testruns/:id/", handler.ReportTestRunById)

		// Project
		project := api.Group("/project")
		project.GET("", projectHandler.GetAllProjects)
		project.POST("", projectHandler.CreateProject)
		project.PUT("/:uuid", projectHandler.UpdateProject)
		project.DELETE("/:uuid", projectHandler.DeleteProject)

		// User Preference
		user := api.Group("/user")
		user.POST("/favourite", userHandler.SaveFavouriteProject)
		user.DELETE("/favourite/:projectUUID", userHandler.DeleteFavouriteProject)
		user.PUT("/preference", userHandler.SaveUserPreference)
		user.GET("/preference", userHandler.GetUserPreference)
		user.POST("/preferred", userHandler.SavePreferredProject)
		user.GET("/preferred", userHandler.GetPreferredProject)
		user.DELETE("/preferred", userHandler.DeletePreferredProject)
	}

	var reports *gin.RouterGroup
	if authEnabled {
		reports = router.Group("/reports/testruns", auth.ScopeMiddleware())
	} else {
		reports = router.Group("/reports/testruns")
	}

	reports.Use()
	{
		reports.GET("/", handler.ReportTestRunAllHTML)
		reports.GET("/:id", handler.ReportTestRunByIdHTML)
	}

	var ping *gin.RouterGroup
	if authEnabled {
		ping = router.Group("/ping", auth.ScopeMiddleware())
	} else {
		ping = router.Group("/ping")
	}

	ping.Use()
	{
		ping.GET("/", handler.Ping)
	}
	insights := router.Group("/insights")
	{
		insights.GET("/:name", handler.ReportTestInsights)
	}
}
