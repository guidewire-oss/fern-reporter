package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/pkg/api/handlers"
	"github.com/guidewire/fern-reporter/pkg/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var _ = Describe("TestRun Handler JSON Response", func() {
	var (
		router  *gin.Engine
		db      *gorm.DB
		testRun models.TestRun
		project models.ProjectDetails
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		db, _ = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		_ = db.AutoMigrate(&models.TestRun{}, &models.ProjectDetails{})

		handler := handlers.NewHandler(db)

		// Seed DB
		project = models.ProjectDetails{
			Name:     "Demo Project",
			TeamName: "Test Team",
			Comment:  "Some notes",
		}
		db.Create(&project)

		testRun = models.TestRun{
			TestSeed:  1234,
			GitBranch: "main",
			GitSha:    "abc123",
			ProjectID: project.ID,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		db.Create(&testRun)

		router = gin.Default()
		api := router.Group("/api")
		testRunGroup := api.Group("/testrun")
		testRunGroup.GET("/", handler.GetTestRunAll)
		testRunGroup.GET("/:id", handler.GetTestRunByID) // <- add this
	})

	It("should return test run with lowercase `project`", func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/testrun/", nil)

		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var parsed []map[string]interface{}
		Expect(json.Unmarshal(w.Body.Bytes(), &parsed)).To(Succeed())
		Expect(parsed).To(HaveLen(1))

		run := parsed[0]

		Expect(run).To(HaveKey("project"))
		project := run["project"].(map[string]interface{})
		Expect(project["name"]).To(Equal("Demo Project"))
		Expect(project["team_name"]).To(Equal("Test Team"))

		Expect(run).ToNot(HaveKey("tags"), "Expected 'tags' to be absent from the response")
	})

	It("should return a single test run by ID with project", func() {
		url := fmt.Sprintf("/api/testrun/%d", testRun.ID)
		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var run map[string]interface{}
		Expect(json.Unmarshal(w.Body.Bytes(), &run)).To(Succeed())

		Expect(run).To(HaveKey("project"))
		project := run["project"].(map[string]interface{})
		Expect(project["name"]).To(Equal("Demo Project"))
		Expect(project["team_name"]).To(Equal("Test Team"))

		Expect(run).ToNot(HaveKey("tags"), "Expected 'tags' to be absent from the response")
	})
})
