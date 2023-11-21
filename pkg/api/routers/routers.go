package routers

import (
	"fern-reporter/pkg/api/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterRouters(router *gin.Engine) {
	router.GET("/", handlers.Home)
	router.GET("/testrun", handlers.TestRunView)
	router.GET("/testrun/:id", handlers.TestRunView)

	api := router.Group("/api")
	{
		person := api.Group("/person")
		person.GET("/", handlers.GetTestRunAll)
		person.GET("/:id", handlers.GetTestRunByID)
		person.POST("/", handlers.CreateTestRun)
		person.PUT("/:id", handlers.UpdateTestRun)
		person.DELETE("/:id", handlers.DeleteTestRun)
	}
}
