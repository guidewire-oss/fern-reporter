package handlers_test

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/api/handlers"
	"github.com/guidewire/fern-reporter/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _ = BeforeEach(func() {
	db, mock, _ = sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 db,
		PreferSimpleProtocol: true,
	})
	gormDb, _ = gorm.Open(dialector, &gorm.Config{})
})

var _ = AfterEach(func() {
	err := db.Close()
	if err != nil {
		fmt.Printf("Unable to close the db connection %s", err.Error())
	}
})

var _ = Describe("Insights test", func() {
	Context("When ReportTestInsights is invoked", func() {
		gin.SetMode(gin.TestMode)
		router := gin.Default()
		funcMap := template.FuncMap{
			"CalculateDuration": utils.CalculateDuration,
			"FormatDate":        utils.FormatDate,
		}
		router.SetFuncMap(funcMap)
		router.LoadHTMLGlob("../../views/insights.html")
		timeQueryLayout := "2006-01-02T15:04:05"

		When("given a query for a test project name from a past test run", func() {
			testProjectName := ""
			_, err := config.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			When("a query is made for a time range that overlaps with the tests", func() {
				startTime := time.Date(2024, 4, 19, 0, 0, 0, 0, time.UTC)
				endTime := time.Date(2024, 4, 22, 0, 0, 0, 0, time.UTC)

				It("should return a summary of the test insights", func() {
					rows := sqlmock.NewRows([]string{"id", "test_project_name", "start_time", "end_time", "pass_rate", "duration"}).
						AddRow(1, "TestProject", time.Date(2024, 4, 20, 12, 0, 0, 0, time.UTC),
							time.Date(2024, 4, 20, 12, 1, 0, 0, time.UTC), 100.000, 60).
						AddRow(2, "TestProject", time.Date(2024, 4, 21, 12, 0, 0, 0, time.UTC),
							time.Date(2024, 4, 21, 12, 1, 0, 0, time.UTC), 33.333, 60)

					mock.ExpectQuery(regexp.QuoteMeta(`SELECT suite_runs.id, test_runs.test_project_name, test_runs.start_time, test_runs.end_time,ROUND(AVG(CASE WHEN spec_runs.status = 'passed' THEN 100.0 ELSE 0.0 END), 3) AS pass_rate, (test_runs.end_time - test_runs.start_time) AS duration FROM "test_runs" INNER JOIN suite_runs ON test_runs.id = suite_runs.test_run_id INNER JOIN spec_runs ON suite_runs.id = spec_runs.suite_id WHERE test_runs.start_time >= $1 AND test_runs.start_time <= $2 AND test_project_name = $3 GROUP BY suite_runs.id, test_runs.test_project_name, test_runs.start_time, test_runs.end_time ORDER BY duration DESC`)).
						WithArgs(startTime, endTime, testProjectName).
						WillReturnRows(rows)

					mock.ExpectQuery(regexp.QuoteMeta(`SELECT AVG(EXTRACT(EPOCH FROM (end_time - start_time))) FROM "test_runs" WHERE test_project_name = $1 AND start_time >= $2 AND start_time <= $3`)).
						WithArgs(testProjectName, startTime, endTime).
						WillReturnRows(sqlmock.NewRows([]string{"avg"}).AddRow(60))

					w := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(w)

					c.Request, _ = http.NewRequest("GET", "/insights", nil)
					c.Params = append(c.Params, gin.Param{Key: "name", Value: testProjectName})
					q := c.Request.URL.Query()
					q.Add("startTime", startTime.Format(timeQueryLayout))
					q.Add("endTime", endTime.Format(timeQueryLayout))
					c.Request.URL.RawQuery = q.Encode()

					handler := handlers.NewHandler(gormDb)
					router.GET("/insights", handler.ReportTestInsights)
					router.ServeHTTP(w, c.Request)

					Expect(w.Code).To(Equal(http.StatusOK))

					doc, err := goquery.NewDocumentFromReader(w.Body)
					Expect(err).NotTo(HaveOccurred())

					averageDurationText := strings.TrimSpace(doc.Find("table:nth-of-type(2) tbody tr td:nth-child(2)").Text())
					Expect(averageDurationText).To(Equal("60"))

					testRunsCount := doc.Find("table:nth-of-type(3) tbody tr.test-row").Length()
					Expect(testRunsCount).To(Equal(2))

					specPassRateOne := strings.TrimSpace(doc.Find("table:nth-of-type(3) tbody tr:nth-child(1) td:nth-child(4)").Text())
					Expect(specPassRateOne).To(Equal("100%"))
					specPassRateTwo := strings.TrimSpace(doc.Find("table:nth-of-type(3) tbody tr:nth-child(2) td:nth-child(4)").Text())
					Expect(specPassRateTwo).To(Equal("33.333%"))

					testDurationOne := strings.TrimSpace(doc.Find("table:nth-of-type(3) tbody tr:nth-child(1) td:nth-child(3)").Text())
					Expect(testDurationOne).To(Equal("1m0s"))
					testDurationTwo := strings.TrimSpace(doc.Find("table:nth-of-type(3) tbody tr:nth-child(1) td:nth-child(3)").Text())
					Expect(testDurationTwo).To(Equal("1m0s"))
				})
			})
			When("given a time range that does not include the test projects", func() {
				startTime := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
				endTime := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

				It("should not include insights for any tests and return default empty data", func() {
					rows := sqlmock.NewRows([]string{"id", "test_project_name", "start_time", "end_time", "pass_rate", "duration"})

					mock.ExpectQuery(regexp.QuoteMeta(`SELECT suite_runs.id, test_runs.test_project_name, test_runs.start_time, test_runs.end_time,ROUND(AVG(CASE WHEN spec_runs.status = 'passed' THEN 100.0 ELSE 0.0 END), 3) AS pass_rate, (test_runs.end_time - test_runs.start_time) AS duration FROM "test_runs" INNER JOIN suite_runs ON test_runs.id = suite_runs.test_run_id INNER JOIN spec_runs ON suite_runs.id = spec_runs.suite_id WHERE test_runs.start_time >= $1 AND test_runs.start_time <= $2 AND test_project_name = $3 GROUP BY suite_runs.id, test_runs.test_project_name, test_runs.start_time, test_runs.end_time ORDER BY duration DESC`)).
						WithArgs(startTime, endTime, testProjectName).
						WillReturnRows(rows)

					mock.ExpectQuery(regexp.QuoteMeta(`SELECT AVG(EXTRACT(EPOCH FROM (end_time - start_time))) FROM "test_runs" WHERE test_project_name = $1 AND start_time >= $2 AND start_time <= $3`)).
						WithArgs(testProjectName, startTime, endTime).
						WillReturnRows(sqlmock.NewRows([]string{"avg"}).AddRow(0))

					w := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(w)

					c.Request, _ = http.NewRequest("GET", "/insights", nil)
					c.Params = append(c.Params, gin.Param{Key: "name", Value: testProjectName})
					q := c.Request.URL.Query()
					q.Add("startTime", startTime.Format(timeQueryLayout))
					q.Add("endTime", endTime.Format(timeQueryLayout))
					c.Request.URL.RawQuery = q.Encode()

					router.ServeHTTP(w, c.Request)

					Expect(w.Code).To(Equal(http.StatusOK))

					doc, err := goquery.NewDocumentFromReader(w.Body)
					Expect(err).NotTo(HaveOccurred())

					averageDurationText := strings.TrimSpace(doc.Find("table:nth-of-type(2) tbody tr td:nth-child(2)").Text())
					Expect(averageDurationText).To(Equal("0"))

					testRunsCount := doc.Find("table:nth-of-type(3) tbody tr.test-row").Length()
					Expect(testRunsCount).To(Equal(0))
				})
			})
		})
	})
})
