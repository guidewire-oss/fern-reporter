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

var _ = Describe("ReportTestRunAll Handler", func() {
	var (
		router *gin.Engine
		db     *gorm.DB
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)

		db, _ = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		Expect(db.AutoMigrate(&models.TestRun{}, &models.ProjectDetails{}, &models.SuiteRun{}, &models.SpecRun{}, &models.Tag{})).To(Succeed())

		handler := handlers.NewHandler(db)

		// Seed Project
		project := models.ProjectDetails{
			Name:     "Demo Project",
			TeamName: "QA",
			Comment:  "Seed project",
		}
		Expect(db.Create(&project).Error).NotTo(HaveOccurred(), "failed to create seed project")

		// Seed TestRun
		testRun := models.TestRun{
			TestSeed:  5678,
			GitBranch: "main",
			GitSha:    "abc123",
			ProjectID: project.ID,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		Expect(db.Create(&testRun).Error).NotTo(HaveOccurred(), "failed to create test run")

		// Seed SuiteRun
		suite := models.SuiteRun{
			TestRunID: testRun.ID,
			SuiteName: "Suite A",
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		Expect(db.Create(&suite).Error).NotTo(HaveOccurred(), "failed to create suite")
		// Seed SpecRun
		spec := models.SpecRun{
			SuiteID:         suite.ID,
			SpecDescription: "should pass",
			Status:          "passed",
			StartTime:       time.Now(),
			EndTime:         time.Now(),
		}
		Expect(db.Create(&spec).Error).NotTo(HaveOccurred(), "failed to create spec")

		// Seed Tag
		tag := models.Tag{Name: "env"}
		Expect(db.Create(&tag).Error).NotTo(HaveOccurred())

		err := db.Model(&spec).Association("Tags").Append(&tag)
		Expect(err).NotTo(HaveOccurred(), "failed to associate tag with specrun")

		// Router
		router = gin.Default()
		api := router.Group("/api")
		report := api.Group("/reports")
		report.GET("/testruns/", handler.ReportTestRunAll)
	})

	It("should return a test run with project and tags", func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/reports/testruns/", nil)

		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response struct {
			TestRuns     []map[string]interface{} `json:"testRuns"`
			ReportHeader string                   `json:"reportHeader"`
			Total        int                      `json:"total"`
		}
		Expect(json.Unmarshal(w.Body.Bytes(), &response)).To(Succeed())

		Expect(response.Total).To(Equal(1))
		Expect(response.TestRuns).To(HaveLen(1))

		run := response.TestRuns[0]
		Expect(run).To(HaveKey("project"))
		project := run["project"].(map[string]interface{})
		Expect(project["name"]).To(Equal("Demo Project"))

		Expect(run).To(HaveKey("suite_runs"))
		suites := run["suite_runs"].([]interface{})
		Expect(suites).ToNot(BeEmpty())

		specRuns := suites[0].(map[string]interface{})["spec_runs"].([]interface{})
		Expect(specRuns).ToNot(BeEmpty())

		spec := specRuns[0].(map[string]interface{})
		Expect(spec).To(HaveKey("tags"))

		tags := spec["tags"].([]interface{})
		Expect(tags).ToNot(BeEmpty())
		Expect(tags[0].(map[string]interface{})["name"]).To(Equal("env"))
	})

	It("should return test run by ID with project and tags", func() {
		// Create a fresh recorder and request for /:id
		var testRun models.TestRun
		db.First(&testRun) // get the ID we created in BeforeEach

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/reports/testruns/"+fmt.Sprint(testRun.ID)+"/", nil)

		// Add route to router
		router.GET("/api/reports/testruns/:id/", handlers.NewHandler(db).ReportTestRunById)

		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response struct {
			TestRuns     []map[string]interface{} `json:"testRuns"`
			ReportHeader string                   `json:"reportHeader"`
		}
		Expect(json.Unmarshal(w.Body.Bytes(), &response)).To(Succeed())
		Expect(response.TestRuns).To(HaveLen(1))

		run := response.TestRuns[0]
		Expect(run).To(HaveKey("project"))
		project := run["project"].(map[string]interface{})
		Expect(project["name"]).To(Equal("Demo Project"))

		Expect(run).To(HaveKey("suite_runs"))
		suites := run["suite_runs"].([]interface{})
		Expect(suites).ToNot(BeEmpty())

		specRuns := suites[0].(map[string]interface{})["spec_runs"].([]interface{})
		Expect(specRuns).ToNot(BeEmpty())

		spec := specRuns[0].(map[string]interface{})
		Expect(spec).To(HaveKey("tags"))

		tags := spec["tags"].([]interface{})
		Expect(tags).ToNot(BeEmpty())
		Expect(tags[0].(map[string]interface{})["name"]).To(Equal("env"))
	})

})
