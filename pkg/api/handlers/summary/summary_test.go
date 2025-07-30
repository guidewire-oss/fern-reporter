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

func setupTestEnvWithTaggedSpecs(specsWithTags map[string][]models.Tag) testEnv {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	Expect(err).ToNot(HaveOccurred())

	err = db.AutoMigrate(&models.ProjectDetails{}, &models.TestRun{}, &models.SuiteRun{}, &models.SpecRun{}, &models.Tag{})
	Expect(err).ToNot(HaveOccurred())

	project := models.ProjectDetails{
		UUID: uuid.New().String(),
		Name: "Test Project",
	}
	// We have to insert using SQL to avoid GORM ignoring UUID since it is normally managed by postgresql
	// language=SQL
	db.Exec(`INSERT INTO project_details (uuid, name) VALUES (?, ?)`, project.UUID, project.Name)

	row := db.Raw(`SELECT id, uuid, name FROM project_details where uuid=?`, project.UUID).Row()
	var id int
	var uuid, name string
	err = row.Scan(&id, &uuid, &name)
	Expect(err).ToNot(HaveOccurred())
	fmt.Printf("Inserted row → ID: %d, UUID: %s, Name: %s\n", id, uuid, name)
	project.ID = uint64(id)

	testRun := models.TestRun{
		TestProjectName: "test-project",
		TestSeed:        1693412583,
		StartTime:       time.Now().Add(-3 * time.Minute),
		EndTime:         time.Now(),
		GitBranch:       "main",
		GitSha:          "deadbeef1234567890",
		ProjectID:       project.ID,
	}
	Expect(db.Create(&testRun).Error).ToNot(HaveOccurred())

	suiteRun := models.SuiteRun{
		TestRunID: testRun.ID,
		SuiteName: "acceptance",
		StartTime: testRun.StartTime,
		EndTime:   testRun.EndTime,
	}
	Expect(db.Create(&suiteRun).Error).ToNot(HaveOccurred())

	specIndex := 0
	for desc, tags := range specsWithTags {
		specRun := models.SpecRun{
			SuiteID:         suiteRun.ID,
			SpecDescription: desc,
			Status:          "passed",
			StartTime:       time.Now().Add(-time.Duration(90-specIndex*30) * time.Second),
			EndTime:         time.Now().Add(-time.Duration(85-specIndex*30) * time.Second),
		}
		Expect(db.Create(&specRun).Error).ToNot(HaveOccurred())

		for _, tag := range tags {
			Expect(db.Create(&tag).Error).ToNot(HaveOccurred())
		}

		Expect(db.Model(&specRun).Association("Tags").Append(&tags)).To(Succeed())
		specIndex++
	}

	// Gin setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	handler := summary.NewSummaryHandler(db)
	api := router.Group("/api")
	api.Group("/summary").GET("/:projectUUID", handler.GetSummary)

	return testEnv{
		db:      db,
		router:  router,
		project: project,
	}
}

var _ = Describe("GetSummary", func() {

	It("returns test summary for a valid project and seed", func() {
		// Create records for specs with tags
		specsWithTags := map[string][]models.Tag{
			"spec-run-a": {
				{Name: "test_type", Value: "acceptance"},
				{Name: "component", Value: "jspolicy"},
				{Name: "owner", Value: "danville"},
				{Name: "category", Value: "infrastructure"},
			},
			"spec-run-b": {
				{Name: "test_type", Value: "acceptance"},
				{Name: "component", Value: "jspolicy"},
				{Name: "owner", Value: "danville"},
				{Name: "category", Value: "infrastructure"},
			},
		}

		env := setupTestEnvWithTaggedSpecs(specsWithTags)

		url := fmt.Sprintf("/api/summary/%s?seed=1693412583", env.project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("head"))
		Expect(parsed).To(HaveKey("summary"))

		head := parsed["head"].(map[string]interface{})
		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(head["branch"]).To(Equal("main"))
		Expect(head["status"]).To(Equal("passed"))
		Expect(int(head["tests"].(float64))).To(Equal(2))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(1))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["test_type"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("jspolicy"))
		Expect(entry["owner"]).To(Equal("danville"))
		Expect(entry["passed"]).To(BeNumerically("==", 2))
		Expect(entry["category"]).To(Equal("infrastructure"))
	})

	It("returns test summary for multiple components", func() {
		// Create records for specs with tags
		specsWithTags := map[string][]models.Tag{
			"spec-run-a": {
				{Name: "test_type", Value: "acceptance"},
				{Name: "component", Value: "jspolicy"},
				{Name: "owner", Value: "capitola"},
				{Name: "category", Value: "infrastructure"},
			},
			"spec-run-b": {
				{Name: "test_type", Value: "acceptance"},
				{Name: "component", Value: "keda"},
				{Name: "owner", Value: "capitola"},
				{Name: "category", Value: "infrastructure"},
			},
		}

		env := setupTestEnvWithTaggedSpecs(specsWithTags)

		url := fmt.Sprintf("/api/summary/%s?seed=1693412583", env.project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		env.router.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &parsed)
		fmt.Printf("Raw response body: %s\n", rec.Body.Bytes())
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("head"))
		Expect(parsed).To(HaveKey("summary"))

		head := parsed["head"].(map[string]interface{})
		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Printf("Parsed response (pretty):\n%s\n", pretty)
		Expect(head["branch"]).To(Equal("main"))
		Expect(head["status"]).To(Equal("passed"))
		Expect(int(head["tests"].(float64))).To(Equal(2))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(2))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["test_type"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("jspolicy"))
		Expect(entry["owner"]).To(Equal("capitola"))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
		Expect(entry["category"]).To(Equal("infrastructure"))

		entry = summaryArr[1].(map[string]interface{})
		Expect(entry["test_type"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("keda"))
		Expect(entry["owner"]).To(Equal("capitola"))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
		Expect(entry["category"]).To(Equal("infrastructure"))
	})
})
