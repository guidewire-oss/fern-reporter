package routers_test

import (
	"database/sql"
	"fmt"
	"github.com/guidewire/fern-reporter/pkg/api/handlers/project"
	"os"
	"reflect"
	"runtime"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/api/handlers"
	"github.com/guidewire/fern-reporter/pkg/api/handlers/user"
	"github.com/guidewire/fern-reporter/pkg/api/routers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _ = Describe("RegisterRouters", func() {
	var (
		router *gin.Engine
		gormDb *gorm.DB
		db     *sql.DB
	)

	BeforeEach(func() {
		router = gin.Default()
		db, _, _ = sqlmock.New()

		dialector := postgres.New(postgres.Config{
			DSN:                  "sqlmock_db_0",
			DriverName:           "postgres",
			Conn:                 db,
			PreferSimpleProtocol: true,
		})
		gormDb, _ = gorm.Open(dialector, &gorm.Config{})
	})

	_ = AfterEach(func() {
		err := db.Close()
		if err != nil {
			fmt.Printf("Unable to close the db connection %s", err.Error())
		}
	})

	Context("Registering routes", func() {
		It("should register API routes", func() {
			handler := handlers.NewHandler(gormDb)
			userHandler := user.NewUserHandler(gormDb)
			projectHandler := project.NewProjectHandler(gormDb)

			routers.RegisterRouters(router)

			Expect(router).NotTo(BeNil())

			// Check if API routes are registered correctly
			ExpectRoute(router, "GET", "/api/testrun/", handler.GetTestRunAll)
			ExpectRoute(router, "GET", "/api/testrun/:id", handler.GetTestRunByID)
			ExpectRoute(router, "POST", "/api/testrun/", handler.CreateTestRun)
			ExpectRoute(router, "PUT", "/api/testrun/:id", handler.UpdateTestRun)
			ExpectRoute(router, "DELETE", "/api/testrun/:id", handler.DeleteTestRun)

			ExpectRoute(router, "GET", "/api/project", projectHandler.GetAllProjects)
			ExpectRoute(router, "POST", "/api/project", projectHandler.CreateProject)
			ExpectRoute(router, "PUT", "/api/project/:uuid", projectHandler.UpdateProject)
			ExpectRoute(router, "DELETE", "/api/project/:uuid", projectHandler.DeleteProject)

			ExpectRoute(router, "POST", "/api/user/favourite", userHandler.SaveFavouriteProject)
			ExpectRoute(router, "DELETE", "/api/user/favourite/:projectUUID", userHandler.DeleteFavouriteProject)
			ExpectRoute(router, "GET", "/api/user/favourite", userHandler.GetFavouriteProject)
			ExpectRoute(router, "PUT", "/api/user/preference", userHandler.SaveUserPreference)
			ExpectRoute(router, "GET", "/api/user/preference", userHandler.GetUserPreference)
			ExpectRoute(router, "POST", "/api/user/project-groups", userHandler.SaveProjectGroups)
			ExpectRoute(router, "GET", "/api/user/project-groups", userHandler.GetProjectGroups)
			ExpectRoute(router, "DELETE", "/api/user/project-groups", userHandler.DeleteProjectGroups)
		})

		It("should register report routes", func() {
			handler := handlers.NewHandler(gormDb)

			routers.RegisterRouters(router)

			Expect(router).NotTo(BeNil())

			// Check if report routes are registered correctly
			ExpectRoute(router, "GET", "/reports/testruns/", handler.ReportTestRunAllHTML)
			ExpectRoute(router, "GET", "/reports/testruns/:id", handler.ReportTestRunByIdHTML)
		})
	})

	Context("Registering routes with auth", func() {
		It("should register API routes", func() {
			handler := handlers.NewHandler(gormDb)

			routers.RegisterRouters(router)

			os.Setenv("AUTH_ENABLED", "true") //nolint:all
			_, err := config.LoadConfig()
			Expect(err).NotTo(HaveOccurred())

			Expect(router).NotTo(BeNil())

			// Check if API routes are registered correctly
			ExpectRoute(router, "GET", "/api/testrun/", handler.GetTestRunAll)
			ExpectRoute(router, "GET", "/api/testrun/:id", handler.GetTestRunByID)
			ExpectRoute(router, "POST", "/api/testrun/", handler.CreateTestRun)
			ExpectRoute(router, "PUT", "/api/testrun/:id", handler.UpdateTestRun)
			ExpectRoute(router, "DELETE", "/api/testrun/:id", handler.DeleteTestRun)
		})

		It("should register report routes", func() {
			handler := handlers.NewHandler(gormDb)

			routers.RegisterRouters(router)

			os.Setenv("AUTH_ENABLED", "true") //nolint:all
			_, err := config.LoadConfig()
			Expect(err).NotTo(HaveOccurred())

			Expect(router).NotTo(BeNil())

			// Check if report routes are registered correctly
			ExpectRoute(router, "GET", "/reports/testruns/", handler.ReportTestRunAllHTML)
			ExpectRoute(router, "GET", "/reports/testruns/:id", handler.ReportTestRunByIdHTML)
		})
	})
})

func ExpectRoute(router *gin.Engine, method, path string, handler gin.HandlerFunc) {
	for _, route := range router.Routes() {
		if route.Method == method && route.Path == path {
			if route.HandlerFunc != nil {
				expectedSource := getSourceCode(handler)
				actualSource := getSourceCode(route.HandlerFunc)
				Expect(actualSource).To(Equal(expectedSource), "Handler mismatch for route: %s %s", method, path)
			} else {
				Fail(fmt.Sprintf("Handler mismatch for route: %s %s. Expected gin.HandlerFunc but got %T", method, path, route.Handler))
			}
			return
		}
	}
	Fail(fmt.Sprintf("Route not found: %s %s", method, path))
}

func getSourceCode(handler gin.HandlerFunc) string {
	pc := reflect.ValueOf(handler).Pointer()
	funcName := runtime.FuncForPC(pc).Name()
	file, line := runtime.FuncForPC(pc).FileLine(0)
	return fmt.Sprintf("%s:%d:%s", file, line, funcName)
}
