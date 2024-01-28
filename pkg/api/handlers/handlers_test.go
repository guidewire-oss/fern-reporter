package handlers_test

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/guidewire/fern-reporter/pkg/api/handlers"
	"github.com/guidewire/fern-reporter/pkg/models"
)

var (
	db     *sql.DB
	gormDb *gorm.DB
	mock   sqlmock.Sqlmock
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
	db.Close()
})

// Define a custom type that implements the driver.Valuer and sql.Scanner interfaces
type myTime time.Time

// Implement the driver.Valuer interface
func (mt myTime) Value() (driver.Value, error) {
	return time.Time(mt), nil
}

// Implement the sql.Scanner interface
func (mt *myTime) Scan(value interface{}) error {
	*mt = myTime(value.(time.Time))
	return nil
}

var _ = Describe("Handlers", func() {
	Context("when GetTestRunAll handleer is invoked", func() {
		It("should query db to fetch all records", func() {

			rows := sqlmock.NewRows([]string{"ID", "TestProjectName"}).
				AddRow(1, "project 1").
				AddRow(2, "project 2")

			mock.ExpectQuery("SELECT (.+) FROM \"test_runs\"").
				WithoutArgs().
				WillReturnRows(rows)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			handler := handlers.NewHandler(gormDb)

			handler.GetTestRunAll(c)

			Expect(w.Code).To(Equal(200))

			var testRuns []models.TestRun
			if err := json.NewDecoder(w.Body).Decode(&testRuns); err != nil {
				Fail(err.Error())
			}
			Expect(len(testRuns)).To(Equal(2))
			Expect(testRuns[0].TestProjectName).To(Equal("project 1"))
			Expect(testRuns[1].TestProjectName).To(Equal("project 2"))
		})
	})

	Context("When GetTestRunByID handler is invoked", func() {
		It("should query DB with where clause filtering by id", func() {

			rows := sqlmock.NewRows([]string{"ID", "TestProjectName"}).
				AddRow(123, "project 123")

			mock.ExpectQuery("SELECT (.+) FROM \"test_runs\" WHERE id = \\$1").
				WithArgs("123").
				WillReturnRows(rows)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = append(c.Params, gin.Param{Key: "id", Value: "123"})
			handler := handlers.NewHandler(gormDb)
			handler.GetTestRunByID(c)

			Expect(w.Code).To(Equal(200))

			var testRun models.TestRun

			if err := json.NewDecoder(w.Body).Decode(&testRun); err != nil {
				Fail(err.Error())
			}
			Expect(int(testRun.ID)).To(Equal(123))
			Expect(testRun.TestProjectName).To(Equal("project 123"))
		})
	})

	/*Context("When UpdateTestRun handler is invoked", func() {
		It("should update record from DB by id", func() {

			// testRunRow := sqlmock.NewRows([]string{"ID", "TestProjectName"}).
			// 	AddRow(123, "project 123", "321", myTime(time.Now()), time.Now)

			testRunRow := sqlmock.NewRows([]string{"id", "test_project_name"}).
				AddRow(1, "Sample Project")

			// _ := sqlmock.NewRows([]string{"id", "test_run_id", "suite_name", "start_time", "end_time"}).
			// 	AddRow(1, 1, "Sample Suite", myTime(time.Now()), myTime(time.Now()))

			// _ := sqlmock.NewRows([]string{"id", "suite_id", "spec_description", "status", "message", "start_time", "end_time"}).
			// 	AddRow(1, 1, "Sample Spec 1", "passed", "All checks passed", myTime(time.Now()), myTime(time.Now())).
			// 	AddRow(2, 1, "Sample Spec 2", "failed", "Assertion failed", myTime(time.Now()), myTime(time.Now()))

			// mock.ExpectQuery("SELECT (.+) FROM \"test_runs\" WHERE id = \\$1").
			// 	WithArgs("123").
			// 	WillReturnRows(testRunRow)

			fmt.Println(testRunRow)

			mock.ExpectQuery("SELECT (.+) FROM \"test_runs\" WHERE id = \\$1").WithArgs("1").WillReturnRows(testRunRow)
			//mock.ExpectQuery("SELECT (.+) FROM \"suite_runs\" WHERE test_run_id = \\$1").WithArgs("1").WillReturnRows(suiteRunRow)
			//mock.ExpectQuery("SELECT (.+) FROM \"spec_runs\" WHERE suite_id = \\$1").WithArgs("1").WillReturnRows(specRunRow)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = append(c.Params, gin.Param{Key: "id", Value: "1"})
			handler := handlers.NewHandler(gormDb)
			handler.UpdateTestRun(c)

			fmt.Print(w)

			Expect(w.Code).To(Equal(200))

			var testRun models.TestRun

			if err := json.NewDecoder(w.Body).Decode(&testRun); err != nil {
				Fail(err.Error())
			}
			Expect(int(testRun.ID)).To(BeNil())

			Expect(testRun.TestProjectName).To(BeNil())
		})
	})*/

	Context("When DeleteTestRun handler is invoked", func() {
		It("should delete record from DB by id", func() {

			testRunRow := sqlmock.NewResult(1, 1)

			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM \"test_runs\" WHERE \"test_runs\".\"id\" = \\$1").
				WithArgs(123).
				WillReturnResult(testRunRow)
			mock.ExpectCommit()
			mock.ExpectClose()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = append(c.Params, gin.Param{Key: "id", Value: "123"})
			handler := handlers.NewHandler(gormDb)
			handler.DeleteTestRun(c)

			Expect(w.Code).To(Equal(200))

			var testRun models.TestRun

			if err := json.NewDecoder(w.Body).Decode(&testRun); err != nil {
				Fail(err.Error())
			}
			Expect(int(testRun.ID)).To(Equal(123))
		})

		It("should handle error", func() {

			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM \"test_runs\" WHERE \"test_runs\".\"id\" = \\$1").
				WithArgs(123).
				WillReturnError(sql.ErrConnDone)
			mock.ExpectRollback()
			mock.ExpectClose()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = append(c.Params, gin.Param{Key: "id", Value: "123"})
			handler := handlers.NewHandler(gormDb)
			handler.DeleteTestRun(c)

			Expect(w.Code).To(Not(Equal(200)))
			Expect(w.Code).To((Equal(http.StatusInternalServerError)))

			var testRun models.TestRun

			if err := json.NewDecoder(w.Body).Decode(&testRun); err != nil {
				Fail(err.Error())
			}
		})

		It("should handle scenario of no rows affected", func() {

			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM \"test_runs\" WHERE \"test_runs\".\"id\" = \\$1").
				WithArgs(123).
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()
			mock.ExpectClose()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = append(c.Params, gin.Param{Key: "id", Value: "123"})
			handler := handlers.NewHandler(gormDb)
			handler.DeleteTestRun(c)

			Expect(w.Code).To((Equal(http.StatusNotFound)))

			// Ensure you call Result before reading the body
			result := w.Result()

			// Extract the response body as a string
			body, err := io.ReadAll(result.Body)
			if err != nil {
				// Handle the error
				fmt.Printf("Error reading response body: %v", err)
				return
			}

			// Parse the JSON response
			var response map[string]interface{}
			if err := json.Unmarshal(body, &response); err != nil {
				// Handle the error
				fmt.Printf("Error parsing JSON response: %v", err)
				return
			}

			// Extract the error message
			errorMessage, _ := response["error"].(string)
			Expect(errorMessage).To((Equal("test run not found")))

		})

		It("should handle invalid id format", func() {
			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM \"test_runs\" WHERE \"test_runs\".\"id\" = \\$1").
				WithArgs(123).
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()
			mock.ExpectClose()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = append(c.Params, gin.Param{Key: "id", Value: "invalidID"})
			handler := handlers.NewHandler(gormDb)
			handler.DeleteTestRun(c)

			Expect(w.Code).To((Equal(http.StatusNotFound)))

		})

	})
})
