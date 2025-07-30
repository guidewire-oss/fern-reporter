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

var _ = Describe("GetSummary", func() {
	var (
		db       *gorm.DB
		router   *gin.Engine
		recorder *httptest.ResponseRecorder
		project  models.ProjectDetails
	)

	BeforeEach(func() {
		// Setup DB
		var err error
		db, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
		Expect(err).ToNot(HaveOccurred())

		err = db.AutoMigrate(
			&models.ProjectDetails{},
			&models.TestRun{},
			&models.SuiteRun{},
			&models.SpecRun{},
			&models.Tag{},
		)
		Expect(err).ToNot(HaveOccurred())

		// Seed DB
		project = models.ProjectDetails{
			UUID: uuid.New().String(),
			Name: "Test Project",
		}
		// We have to insert using SQL to avoid GORM ignoring UUID since it is normally managed by postgresql
		// language=SQL
		db.Exec(`INSERT INTO project_details (uuid, name) VALUES (?, ?)`, project.UUID, project.Name)

		row := db.Raw(`SELECT id, uuid, name FROM project_details`).Row()
		var id int
		var uuid, name string
		err = row.Scan(&id, &uuid, &name)
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("Inserted row → ID: %d, UUID: %s, Name: %s\n", id, uuid, name)
		project.ID = uint64(id)

		testRun := models.TestRun{
			TestProjectName: "test-project",
			TestSeed:        1693412583,
			StartTime:       time.Now().Add(-2 * time.Minute),
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

		specRun := models.SpecRun{
			SuiteID:         suiteRun.ID,
			SpecDescription: "Test A",
			Status:          "passed",
			StartTime:       time.Now().Add(-90 * time.Second),
			EndTime:         time.Now().Add(-85 * time.Second),
		}
		Expect(db.Create(&specRun).Error).ToNot(HaveOccurred())

		tags := []models.Tag{
			{Name: "test_type", Value: "acceptance"},
			{Name: "component", Value: "jspolicy"},
			{Name: "owner", Value: "danville"},
			{Name: "category", Value: "infrastructure"},
		}
		for _, tag := range tags {
			Expect(db.Create(&tag).Error).ToNot(HaveOccurred())
		}
		Expect(db.Model(&specRun).Association("Tags").Append(&tags)).To(Succeed())

		// Setup router
		gin.SetMode(gin.TestMode)
		router = gin.Default()

		handler := summary.NewSummaryHandler(db)
		api := router.Group("/api")
		summaryGroup := api.Group("/summary")
		summaryGroup.GET("/:projectUUID", handler.GetSummary)
	})

	It("returns test summary for a valid project and seed", func() {
		url := fmt.Sprintf("/api/summary/%s?seed=1693412583", project.UUID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)

		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))

		var parsed map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &parsed)
		Expect(err).ToNot(HaveOccurred())

		Expect(parsed).To(HaveKey("head"))
		Expect(parsed).To(HaveKey("summary"))

		head := parsed["head"].(map[string]interface{})
		Expect(head["branch"]).To(Equal("main"))
		Expect(head["status"]).To(Equal("passed"))
		Expect(int(head["tests"].(float64))).To(Equal(1))

		summaryArr := parsed["summary"].([]interface{})
		Expect(summaryArr).To(HaveLen(1))

		entry := summaryArr[0].(map[string]interface{})
		Expect(entry["test_type"]).To(Equal("acceptance"))
		Expect(entry["component"]).To(Equal("jspolicy"))
		Expect(entry["owner"]).To(Equal("danville"))
		Expect(entry["passed"]).To(BeNumerically("==", 1))
		Expect(entry["category"]).To(Equal("infrastructure"))
	})
})
