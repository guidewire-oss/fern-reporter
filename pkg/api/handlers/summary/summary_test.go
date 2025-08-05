package summary_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/guidewire/fern-reporter/pkg/api/handlers/summary"
	"github.com/guidewire/fern-reporter/pkg/models"
)

type testEnv struct {
	db      *gorm.DB
	router  *gin.Engine
	project models.ProjectDetails
}

type TaggedSpec struct {
	Description string
	Status      string
	Tags        []models.Tag
}

type TestRunSpecGroup struct {
	TestRunData models.TestRun
	Specs       []TaggedSpec
}

func setupTestEnvWithTaggedSpecs(groups []TestRunSpecGroup) testEnv {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	Expect(err).ToNot(HaveOccurred())

	err = db.AutoMigrate(&models.ProjectDetails{}, &models.TestRun{}, &models.SuiteRun{}, &models.SpecRun{}, &models.Tag{})
	Expect(err).ToNot(HaveOccurred())

	project := models.ProjectDetails{
		UUID: uuid.New().String(),
		Name: "Test Project",
	}
	db.Exec(`INSERT INTO project_details (uuid, name) VALUES (?, ?)`, project.UUID, project.Name)

	row := db.Raw(`SELECT id, uuid, name FROM project_details where uuid=?`, project.UUID).Row()
	var id int
	var uuid, name string
	err = row.Scan(&id, &uuid, &name)
	Expect(err).ToNot(HaveOccurred())
	project.ID = uint64(id)

	for _, group := range groups {
		// Ensure the TestRun is associated with the project
		group.TestRunData.ProjectID = project.ID
		Expect(db.Create(&group.TestRunData).Error).ToNot(HaveOccurred())

		suiteRun := models.SuiteRun{
			TestRunID: group.TestRunData.ID,
			SuiteName: "acceptance",
			StartTime: group.TestRunData.StartTime,
			EndTime:   group.TestRunData.EndTime,
		}
		Expect(db.Create(&suiteRun).Error).ToNot(HaveOccurred())

		for i, taggedSpec := range group.Specs {
			specRun := models.SpecRun{
				SuiteID:         suiteRun.ID,
				SpecDescription: taggedSpec.Description,
				Status:          taggedSpec.Status,
				StartTime:       time.Now().Add(-time.Duration(90-i*30) * time.Second),
				EndTime:         time.Now().Add(-time.Duration(85-i*30) * time.Second),
			}
			Expect(db.Create(&specRun).Error).ToNot(HaveOccurred())

			for _, tag := range taggedSpec.Tags {
				Expect(db.Create(&tag).Error).ToNot(HaveOccurred())
			}

			Expect(db.Model(&specRun).Association("Tags").Append(taggedSpec.Tags)).To(Succeed())
		}
	}

	// Gin setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	handler := summary.NewSummaryHandler(db)
	api := router.Group("/api")
	api.Group("/reports/summary").GET("/project/:projectId/seed/:seed", handler.GetSummary)

	return testEnv{
		db:      db,
		router:  router,
		project: project,
	}
}

var _ = Describe("GetSummary", func() {

	It("returns test summary for a valid project and seed", func() {

		env := setupTestEnvWithTaggedSpecs([]TestRunSpecGroup{
			{
				TestRunData: models.TestRun{
					TestProjectName: "test-project",
					TestSeed:        1693412583,
					StartTime:       time.Now().Add(-5 * time.Minute),
					EndTime:         time.Now().Add(-4 * time.Minute),
					GitBranch:       "main",
					GitSha:          "deadbeef1234567890",
				},
				Specs: []TaggedSpec{
					{
						Description: "spec-run-a",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "infrastructure"},
						},
					},
					{
						Description: "spec-run-b",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "infrastructure"},
						},
					},
				},
			},
		})

		url := fmt.Sprintf("/api/reports/summary/project/%s/seed/1693412583?group_by=testtype&group_by=component&group_by=owner&group_by=category", env.project.UUID)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			Fail(fmt.Sprintf("Failed to create http request: %v", err))
		}

		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("summary"))

		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(parsed["branch"]).To(Equal("main"))
		Expect(parsed["status"]).To(Equal("passed"))
		Expect(int(parsed["tests"].(float64))).To(Equal(2))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(1))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("jspolicy"))
		Expect(entry["owner"]).To(Equal("capitola"))
		Expect(entry["passed"]).To(BeNumerically("==", 2))
		Expect(entry["category"]).To(Equal("infrastructure"))
	})

	It("returns test summary for multiple components", func() {

		env := setupTestEnvWithTaggedSpecs([]TestRunSpecGroup{
			{
				TestRunData: models.TestRun{
					TestProjectName: "test-project",
					TestSeed:        1693412583,
					StartTime:       time.Now().Add(-5 * time.Minute),
					EndTime:         time.Now().Add(-4 * time.Minute),
					GitBranch:       "main",
					GitSha:          "deadbeef1234567890",
				},
				Specs: []TaggedSpec{
					{
						Description: "spec-run-a",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "infrastructure"},
						},
					},
					{
						Description: "spec-run-b",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "keda"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "infrastructure"},
						},
					},
				},
			},
		})
		///

		url := fmt.Sprintf("/api/reports/summary/project/%s/seed/1693412583?group_by=testtype&group_by=component&group_by=owner&group_by=category", env.project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("summary"))

		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(parsed["branch"]).To(Equal("main"))
		Expect(parsed["status"]).To(Equal("passed"))
		Expect(int(parsed["tests"].(float64))).To(Equal(2))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(2))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("jspolicy"))
		Expect(entry["owner"]).To(Equal("capitola"))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
		Expect(entry["category"]).To(Equal("infrastructure"))

		entry = summaryArr[1].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("keda"))
		Expect(entry["owner"]).To(Equal("capitola"))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
		Expect(entry["category"]).To(Equal("infrastructure"))
	})

	It("returns test summary for failed status", func() {
		env := setupTestEnvWithTaggedSpecs([]TestRunSpecGroup{
			{
				TestRunData: models.TestRun{
					TestProjectName: "test-project",
					TestSeed:        1693412583,
					StartTime:       time.Now().Add(-5 * time.Minute),
					EndTime:         time.Now().Add(-4 * time.Minute),
					GitBranch:       "main",
					GitSha:          "deadbeef1234567890",
				},
				Specs: []TaggedSpec{
					{
						Description: "spec-run-a",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "danville"},
							{Category: "category", Value: "infrastructure"},
						},
					},
					{
						Description: "spec-run-b",
						Status:      "failed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "danville"},
							{Category: "category", Value: "infrastructure"},
						},
					},
				},
			},
		})

		url := fmt.Sprintf("/api/reports/summary/project/%s/seed/1693412583?group_by=testtype&group_by=component&group_by=owner&group_by=category", env.project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("summary"))

		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(parsed["branch"]).To(Equal("main"))
		Expect(parsed["status"]).To(Equal("failed"))
		Expect(int(parsed["tests"].(float64))).To(Equal(2))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(1))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("jspolicy"))
		Expect(entry["owner"]).To(Equal("danville"))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
		Expect(entry["failed"]).To(BeNumerically("==", 1))
		Expect(entry["category"]).To(Equal("infrastructure"))
	})

	It("returns test summary appropriately when grouping by something other than the default tags - eg just test type", func() {

		env := setupTestEnvWithTaggedSpecs([]TestRunSpecGroup{
			{
				TestRunData: models.TestRun{
					TestProjectName: "test-project",
					TestSeed:        1693412583,
					StartTime:       time.Now().Add(-5 * time.Minute),
					EndTime:         time.Now().Add(-4 * time.Minute),
					GitBranch:       "main",
					GitSha:          "deadbeef1234567890",
				},
				Specs: []TaggedSpec{
					{
						Description: "spec-run-a",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "metrics-server"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "helm"},
						},
					},
					{
						Description: "spec-run-b",
						Status:      "failed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "danville"},
							{Category: "category", Value: "infrastructure"},
						},
					},
				},
			},
		})

		url := fmt.Sprintf("/api/reports/summary/project/%s/seed/1693412583?group_by=testtype", env.project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("summary"))

		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(parsed["branch"]).To(Equal("main"))
		Expect(parsed["status"]).To(Equal("failed"))
		Expect(int(parsed["tests"].(float64))).To(Equal(2))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(1))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
		Expect(entry["failed"]).To(BeNumerically("==", 1))
	})

	It("returns test summary appropriately when grouping by two tags, test type and component", func() {
		env := setupTestEnvWithTaggedSpecs([]TestRunSpecGroup{
			{
				TestRunData: models.TestRun{
					TestProjectName: "test-project",
					TestSeed:        1693412583,
					StartTime:       time.Now().Add(-5 * time.Minute),
					EndTime:         time.Now().Add(-4 * time.Minute),
					GitBranch:       "main",
					GitSha:          "deadbeef1234567890",
				},
				Specs: []TaggedSpec{
					{
						Description: "spec-run-a",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "metrics-server"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "helm"},
						},
					},
					{
						Description: "spec-run-b",
						Status:      "failed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "danville1"},
							{Category: "category", Value: "infrastructure"},
						},
					},
					{
						Description: "spec-run-c",
						Status:      "skipped",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "danville2"},
							{Category: "category", Value: "infrastructure"},
						},
					},
					{
						Description: "spec-run-d",
						Status:      "pending",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "keda"},
							{Category: "owner", Value: "danville3"},
							{Category: "category", Value: "infrastructure"},
						},
					},
				},
			},
		})
		url := fmt.Sprintf("/api/reports/summary/project/%s/seed/1693412583?group_by=testtype&group_by=component", env.project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("summary"))

		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(parsed["branch"]).To(Equal("main"))
		Expect(parsed["status"]).To(Equal("failed"))
		Expect(int(parsed["tests"].(float64))).To(Equal(4))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(3))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("jspolicy"))
		Expect(entry["failed"]).To(BeNumerically("==", 1))
		Expect(entry["skipped"]).To(BeNumerically("==", 1))

		entry = summaryArr[1].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("keda"))
		Expect(entry["pending"]).To(BeNumerically("==", 1))

		entry = summaryArr[2].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("metrics-server"))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
	})

	It("behaves logically when you ask it to group by a non-existent tag", func() {

		env := setupTestEnvWithTaggedSpecs([]TestRunSpecGroup{
			{
				TestRunData: models.TestRun{
					TestProjectName: "test-project",
					TestSeed:        1693412583,
					StartTime:       time.Now().Add(-5 * time.Minute),
					EndTime:         time.Now().Add(-4 * time.Minute),
					GitBranch:       "main",
					GitSha:          "deadbeef1234567890",
				},
				Specs: []TaggedSpec{
					{
						Description: "spec-run-a",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "metrics-server"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "helm"},
						},
					},
					{
						Description: "spec-run-b",
						Status:      "failed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "danville"},
							{Category: "category", Value: "infrastructure"},
						},
					},
				},
			},
		})

		url := fmt.Sprintf("/api/reports/summary/project/%s/seed/1693412583?group_by=banana", env.project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("summary"))

		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(parsed["branch"]).To(Equal("main"))
		Expect(parsed["status"]).To(Equal("failed"))
		Expect(int(parsed["tests"].(float64))).To(Equal(2))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(1))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["passed"]).To(BeNumerically("==", 1))
		Expect(entry["failed"]).To(BeNumerically("==", 1))
	})

	It("returns test summary for multiple test runs with same project and seed", func() {

		env := setupTestEnvWithTaggedSpecs([]TestRunSpecGroup{
			{
				TestRunData: models.TestRun{
					TestProjectName: "test-project",
					TestSeed:        1693412583,
					StartTime:       time.Now().Add(-5 * time.Minute),
					EndTime:         time.Now().Add(-4 * time.Minute),
					GitBranch:       "main",
					GitSha:          "deadbeef1234567890",
				},
				Specs: []TaggedSpec{
					{
						Description: "spec-run-a",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "keda"},
							{Category: "owner", Value: "danville"},
							{Category: "category", Value: "infrastructure"},
						},
					},
					{
						Description: "spec-run-b",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "keda"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "infrastructure"},
						},
					},
				},
			},
			{
				TestRunData: models.TestRun{
					TestProjectName: "test-project",
					TestSeed:        1693412583,
					StartTime:       time.Now().Add(-5 * time.Minute),
					EndTime:         time.Now().Add(-4 * time.Minute),
					GitBranch:       "main",
					GitSha:          "deadbeef1234567890",
				},
				Specs: []TaggedSpec{
					{
						Description: "spec-run-c",
						Status:      "failed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "keda"},
							{Category: "owner", Value: "danville"},
							{Category: "category", Value: "infrastructure"},
						},
					},
					{
						Description: "spec-run-d",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "keda"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "infrastructure"},
						},
					},
				},
			},
		})

		url := fmt.Sprintf("/api/reports/summary/project/%s/seed/1693412583?group_by=testtype&group_by=component&group_by=owner&group_by=category", env.project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("summary"))

		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(parsed["branch"]).To(Equal("main"))
		Expect(parsed["status"]).To(Equal("failed"))
		Expect(int(parsed["tests"].(float64))).To(Equal(4))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(2))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("keda"))
		Expect(entry["owner"]).To(Equal("capitola"))
		Expect(entry["passed"]).To(BeNumerically("==", 2))
		Expect(entry["category"]).To(Equal("infrastructure"))

		entry = summaryArr[1].(map[string]interface{})
		Expect(entry["testtype"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("keda"))
		Expect(entry["owner"]).To(Equal("danville"))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
		Expect(entry["failed"]).To(BeNumerically("==", 1))
		Expect(entry["category"]).To(Equal("infrastructure"))

	})

	It("returns test summary appropriately when group_by clause is not specified", func() {
		env := setupTestEnvWithTaggedSpecs([]TestRunSpecGroup{
			{
				TestRunData: models.TestRun{
					TestProjectName: "test-project",
					TestSeed:        1693412583,
					StartTime:       time.Now().Add(-5 * time.Minute),
					EndTime:         time.Now().Add(-4 * time.Minute),
					GitBranch:       "main",
					GitSha:          "deadbeef1234567890",
				},
				Specs: []TaggedSpec{
					{
						Description: "spec-run-a",
						Status:      "passed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "metrics-server"},
							{Category: "owner", Value: "capitola"},
							{Category: "category", Value: "helm"},
						},
					},
					{
						Description: "spec-run-b",
						Status:      "failed",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "danville1"},
							{Category: "category", Value: "infrastructure"},
						},
					},
					{
						Description: "spec-run-c",
						Status:      "skipped",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "jspolicy"},
							{Category: "owner", Value: "danville2"},
							{Category: "category", Value: "infrastructure"},
						},
					},
					{
						Description: "spec-run-d",
						Status:      "pending",
						Tags: []models.Tag{
							{Category: "testtype", Value: "acceptance"},
							{Category: "component", Value: "keda"},
							{Category: "owner", Value: "danville3"},
							{Category: "category", Value: "infrastructure"},
						},
					},
				},
			},
		})
		url := fmt.Sprintf("/api/reports/summary/project/%s/seed/1693412583", env.project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("summary"))

		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(parsed["branch"]).To(Equal("main"))
		Expect(parsed["status"]).To(Equal("failed"))
		Expect(int(parsed["tests"].(float64))).To(Equal(4))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(1))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["failed"]).To(BeNumerically("==", 1))
		Expect(entry["skipped"]).To(BeNumerically("==", 1))
		Expect(entry["pending"]).To(BeNumerically("==", 1))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
	})

})
