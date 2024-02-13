package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
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

var _ = Describe("Handlers", func() {
	Context("when GetTestRunAll handler is invoked", func() {
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

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs("123", 1).
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

	Context("when createTestRun handler is invoked", func() {
		It("and test run doesn't exist, it should create one and return 201 OK", func() {
			expectedTestRun := models.TestRun{
				ID:              0,
				TestProjectName: "TestProject",
				StartTime:       time.Time{},
				EndTime:         time.Time{},
				TestSeed:        0,
				SuiteRuns: []models.SuiteRun{
					{
						ID:        1,
						TestRunID: 1,
						SuiteName: "TestSuite",
						StartTime: time.Now(),
						EndTime:   time.Now(),
						SpecRuns: []models.SpecRun{
							{
								ID:              1,
								SuiteID:         1,
								SpecDescription: "TestSpec",
								Status:          "Passed",
								Message:         "",
								StartTime:       time.Now(),
								EndTime:         time.Now(),
							},
						},
					},
				},
			}

			_, err := json.Marshal(expectedTestRun.SuiteRuns)
			if err != nil {
				fmt.Printf("Error serializing SuiteRuns: %v", err)
				return
			}

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "test_runs" ("test_project_name","test_seed","start_time","end_time") VALUES ($1,$2,$3,$4) RETURNING "id"`)).
				WithArgs(expectedTestRun.TestProjectName, expectedTestRun.TestSeed, expectedTestRun.StartTime, expectedTestRun.EndTime).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectCommit()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create a new request with JSON payload
			jsonStr := []byte(`{"id": 0, "test_project_name":"TestProject"}`)
			req, err := http.NewRequest("POST", "/", bytes.NewBuffer(jsonStr))
			if err != nil {
				fmt.Printf("%v", err)
			}

			// Set the Content-Type header to application/json
			req.Header.Set("Content-Type", "application/json")

			c.Request = req
			handler := handlers.NewHandler(gormDb)
			handler.CreateTestRun(c)

			// Check the response status code
			Expect(w.Code).To(Equal(http.StatusCreated))
			var testRun models.TestRun

			if err := json.NewDecoder(w.Body).Decode(&testRun); err != nil {
				Fail(err.Error())
			}
			Expect(int(testRun.ID)).To(Equal(1))
			Expect(testRun.TestProjectName).To(Equal(expectedTestRun.TestProjectName))
			Expect(testRun.TestSeed).To(Equal(expectedTestRun.TestSeed))
			Expect(testRun.StartTime).To(Equal(expectedTestRun.StartTime))
			Expect(testRun.EndTime).To(Equal(expectedTestRun.EndTime))

		})

		It("with bad POST payload, it should return Bad Request 400 ", func() {

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create a new request with JSON payload
			jsonStr := []byte(`"BAD_PAYLOAD_KEY" "BAD_VALUE"`)
			req, err := http.NewRequest("POST", "/", bytes.NewBuffer(jsonStr))
			if err != nil {
				fmt.Printf("%v", err)
			}

			// Set the Content-Type header to application/json
			req.Header.Set("Content-Type", "application/json")

			c.Request = req
			handler := handlers.NewHandler(gormDb)
			handler.CreateTestRun(c)

			// Check the response status code
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("and test run record exists, it should handle error while finding existing record and return 404 Not Found", func() {
			expectedTestRun := models.TestRun{
				ID:              1,
				TestProjectName: "TestProject",
				StartTime:       time.Time{},
				EndTime:         time.Time{},
				TestSeed:        1,
				SuiteRuns: []models.SuiteRun{
					{
						ID:        1,
						TestRunID: 1,
						SuiteName: "TestSuite",
						StartTime: time.Now(),
						EndTime:   time.Now(),
						SpecRuns: []models.SpecRun{
							{
								ID:              1,
								SuiteID:         1,
								SpecDescription: "TestSpec",
								Status:          "Passed",
								Message:         "",
								StartTime:       time.Now(),
								EndTime:         time.Now(),
							},
						},
					},
				},
			}

			_, err := json.Marshal(expectedTestRun.SuiteRuns)
			if err != nil {
				fmt.Printf("Error serializing SuiteRuns: %v", err)
				return
			}

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs(expectedTestRun.ID, 1).
				WillReturnError(errors.New("Record not found DB error"))

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create a new request with JSON payload
			jsonStr := []byte(`{"id": 1, "test_project_name":"TestProject"}`)
			req, err := http.NewRequest("POST", "/", bytes.NewBuffer(jsonStr))
			if err != nil {
				fmt.Printf("%v", err)
			}

			req.Header.Set("Content-Type", "application/json")

			c.Request = req
			handler := handlers.NewHandler(gormDb)
			handler.CreateTestRun(c)

			// Check the response status code
			Expect(w.Code).To(Equal(http.StatusNotFound))

		})

		It("and error occurs during ProcessTags, it should handle error and return 500 Internal Server Error", func() {
			var testRun = models.TestRun{
				ID:              1,
				TestProjectName: "TestProject",
				StartTime:       time.Time{},
				EndTime:         time.Time{},
				TestSeed:        1,
				SuiteRuns: []models.SuiteRun{
					{
						ID:        1,
						TestRunID: 1,
						SuiteName: "TestSuite",
						StartTime: time.Now(),
						EndTime:   time.Now(),
						SpecRuns: []models.SpecRun{
							{
								ID:              1,
								SuiteID:         1,
								SpecDescription: "TestSpec",
								Status:          "Passed",
								Message:         "",
								StartTime:       time.Now(),
								EndTime:         time.Now(),
								Tags: []models.Tag{
									{
										ID:   1,
										Name: "TagName",
									},
								},
							},
						},
					},
				},
			}

			//mock.ExpectBegin()
			testRuns := sqlmock.NewRows([]string{"id", "TestProjectName"}).
				AddRow(1, "project 1")

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs(testRun.ID, 1).
				WillReturnRows(testRuns)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tags" WHERE id = $1 ORDER BY "tags"."id" LIMIT $2`)).
				WithArgs(1, 1).
				WillReturnError(errors.New("database error"))

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			testRunJson, err := json.Marshal(testRun)
			if err != nil {
				// Handle error
				fmt.Println("Error:", err)
				return
			}

			req, err := http.NewRequest("POST", "/", bytes.NewBuffer(testRunJson))
			if err != nil {
				fmt.Printf("%v", err)
			}

			req.Header.Set("Content-Type", "application/json")

			c.Request = req
			handler := handlers.NewHandler(gormDb)
			handler.CreateTestRun(c)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("error processing tags"))

		})

		It("and error occurs during Save/Update of testRun record, it should handle error and return 500 Internal Server Error", func() {
			var testRun = models.TestRun{
				ID:              1,
				TestProjectName: "TestProject",
				StartTime:       time.Time{},
				EndTime:         time.Time{},
				TestSeed:        1,
				SuiteRuns: []models.SuiteRun{
					{
						ID:        1,
						TestRunID: 1,
						SuiteName: "TestSuite",
						StartTime: time.Now(),
						EndTime:   time.Now(),
						SpecRuns: []models.SpecRun{
							{
								ID:              1,
								SuiteID:         1,
								SpecDescription: "TestSpec",
								Status:          "Passed",
								Message:         "",
								StartTime:       time.Now(),
								EndTime:         time.Now(),
								Tags: []models.Tag{
									{
										ID:   1,
										Name: "TagName",
									},
								},
							},
						},
					},
				},
			}

			testRuns := sqlmock.NewRows([]string{"id", "TestProjectName"}).
				AddRow(1, "project 1")

			//mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs(testRun.ID, 1).
				WillReturnRows(testRuns)

			rows := sqlmock.NewRows([]string{"id"})

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tags" WHERE name = $1 ORDER BY "tags"."id" LIMIT $2`)).WithArgs("TagName", 1).WillReturnRows(rows)

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tags" ("name") VALUES ($1) RETURNING "id"`)).
				WithArgs("TagName").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectCommit()

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "test_runs" SET "test_project_name"=$1,"test_seed"=$2,"start_time"=$3,"end_time"=$4 WHERE "id" = $5`)).
				WithArgs(testRun.TestProjectName, testRun.TestSeed, testRun.StartTime, testRun.EndTime, testRun.ID).
				WillReturnError(errors.New("unable to save record"))
			mock.ExpectRollback()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			testRunJson, err := json.Marshal(testRun)
			if err != nil {
				// Handle error
				fmt.Println("Error:", err)
				return
			}

			req, err := http.NewRequest("POST", "/", bytes.NewBuffer(testRunJson))
			if err != nil {
				fmt.Printf("%v", err)
			}

			req.Header.Set("Content-Type", "application/json")

			c.Request = req
			handler := handlers.NewHandler(gormDb)
			handler.CreateTestRun(c)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("error saving record"))

		})

	})

	Context("When ProcessTags is invoked", func() {
		var testRun = models.TestRun{
			ID:              0,
			TestProjectName: "TestProject",
			StartTime:       time.Time{},
			EndTime:         time.Time{},
			TestSeed:        0,
			SuiteRuns: []models.SuiteRun{
				{
					ID:        1,
					TestRunID: 1,
					SuiteName: "TestSuite",
					StartTime: time.Now(),
					EndTime:   time.Now(),
					SpecRuns: []models.SpecRun{
						{
							ID:              1,
							SuiteID:         1,
							SpecDescription: "TestSpec",
							Status:          "Passed",
							Message:         "",
							StartTime:       time.Now(),
							EndTime:         time.Now(),
						},
					},
				},
			},
		}
		BeforeEach(func() {
			for _, suite := range testRun.SuiteRuns {
				for _, spec := range suite.SpecRuns {
					for _, tag := range spec.Tags {
						mock.ExpectBegin()
						rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
						mock.ExpectQuery("SELECT").WithArgs(tag.Name).WillReturnRows(rows)
						mock.ExpectCommit()
					}
				}
			}
		})

		It("should process tags successfully", func() {

			err := handlers.ProcessTags(gormDb, &testRun)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When ProcessTags is invoked and tag creation fails", func() {
		var testRun = models.TestRun{
			ID:              0,
			TestProjectName: "TestProject",
			StartTime:       time.Time{},
			EndTime:         time.Time{},
			TestSeed:        0,
			SuiteRuns: []models.SuiteRun{
				{
					ID:        1,
					TestRunID: 1,
					SuiteName: "TestSuite",
					StartTime: time.Now(),
					EndTime:   time.Now(),
					SpecRuns: []models.SpecRun{
						{
							ID:              1,
							SuiteID:         1,
							SpecDescription: "TestSpec",
							Status:          "Passed",
							Message:         "",
							StartTime:       time.Now(),
							EndTime:         time.Now(),
							Tags: []models.Tag{
								{
									ID:   1,
									Name: "TagName",
								},
							},
						},
					},
				},
			},
		}
		BeforeEach(func() {
			for _, suite := range testRun.SuiteRuns {
				for _, spec := range suite.SpecRuns {
					for _, tag := range spec.Tags {
						mock.ExpectBegin()
						mock.ExpectQuery("SELECT").WithArgs(tag.Name).WillReturnError(errors.New("database error"))
						mock.ExpectRollback()
					}
				}
			}
		})

		It("should return an error", func() {
			err := handlers.ProcessTags(gormDb, &testRun)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When ProcessTags is invoked and tag already exists in the database", func() {
		var testRun = models.TestRun{
			ID:              0,
			TestProjectName: "TestProject",
			StartTime:       time.Time{},
			EndTime:         time.Time{},
			TestSeed:        0,
			SuiteRuns: []models.SuiteRun{
				{
					ID:        1,
					TestRunID: 1,
					SuiteName: "TestSuite",
					StartTime: time.Now(),
					EndTime:   time.Now(),
					SpecRuns: []models.SpecRun{
						{
							ID:              1,
							SuiteID:         1,
							SpecDescription: "TestSpec",
							Status:          "Passed",
							Message:         "",
							StartTime:       time.Now(),
							EndTime:         time.Now(),
							Tags: []models.Tag{
								{
									ID:   1,
									Name: "TagName",
								},
							},
						},
					},
				},
			},
		}
		BeforeEach(func() {
			for _, suite := range testRun.SuiteRuns {
				for _, spec := range suite.SpecRuns {
					for _, tag := range spec.Tags {
						rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
						mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tags" WHERE name = $1 ORDER BY "tags"."id" LIMIT $2`)).WithArgs(tag.Name, 1).WillReturnRows(rows)
						mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tags" WHERE name = $1 ORDER BY "tags"."id" LIMIT $2`)).WithArgs(tag.Name, 1).WillReturnRows(rows)
						mock.ExpectCommit()
					}
				}
			}
		})

		It("should use existing tag", func() {
			err := handlers.ProcessTags(gormDb, &testRun)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When ProcessTags is invoked and is provided with an empty test run", func() {
		It("should not return an error", func() {
			testRun := &models.TestRun{}
			err := handlers.ProcessTags(gormDb, testRun)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When ProcessTags is invoked and tag is not found", func() {
		It("should create a new tag", func() {
			tag := models.Tag{Name: "NewTag"}
			specRun := models.SpecRun{Tags: []models.Tag{tag}}
			suiteRun := models.SuiteRun{SpecRuns: []models.SpecRun{specRun}}
			testRun := &models.TestRun{SuiteRuns: []models.SuiteRun{suiteRun}}

			rows := sqlmock.NewRows([]string{"id"})

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tags" WHERE name = $1 ORDER BY "tags"."id" LIMIT $2`)).WithArgs(tag.Name, 1).WillReturnRows(rows)
			mock.ExpectBegin()

			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tags" ("name") VALUES ($1) RETURNING "id"`)).
				WithArgs(tag.Name).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectCommit()

			err := handlers.ProcessTags(gormDb, testRun)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When ProcessTags is invoked and tag is not found", func() {
		It("and tag creation has an error, it should return the error", func() {
			tag := models.Tag{Name: "NewTag"}
			specRun := models.SpecRun{Tags: []models.Tag{tag}}
			suiteRun := models.SuiteRun{SpecRuns: []models.SpecRun{specRun}}
			testRun := &models.TestRun{SuiteRuns: []models.SuiteRun{suiteRun}}

			rows := sqlmock.NewRows([]string{"id"})

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tags" WHERE name = $1 ORDER BY "tags"."id" LIMIT $2`)).WithArgs(tag.Name, 1).WillReturnRows(rows)
			mock.ExpectBegin()

			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tags" ("name") VALUES ($1) RETURNING "id"`)).
				WithArgs(tag.Name).WillReturnError(errors.New("database error"))
			mock.ExpectCommit()

			err := handlers.ProcessTags(gormDb, testRun)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When ProcessTags is invoked and tag creation fails due to unknown error", func() {
		var testRun = models.TestRun{
			ID:              0,
			TestProjectName: "TestProject",
			StartTime:       time.Time{},
			EndTime:         time.Time{},
			TestSeed:        0,
			SuiteRuns: []models.SuiteRun{
				{
					ID:        1,
					TestRunID: 1,
					SuiteName: "TestSuite",
					StartTime: time.Now(),
					EndTime:   time.Now(),
					SpecRuns: []models.SpecRun{
						{
							ID:              1,
							SuiteID:         1,
							SpecDescription: "TestSpec",
							Status:          "Passed",
							Message:         "",
							StartTime:       time.Now(),
							EndTime:         time.Now(),
							Tags: []models.Tag{
								{
									ID:   1,
									Name: "TagName",
								},
							},
						},
					},
				},
			},
		}
		BeforeEach(func() {
			for _, suite := range testRun.SuiteRuns {
				for _, spec := range suite.SpecRuns {
					for _, tag := range spec.Tags {
						mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tags" WHERE name = $1 ORDER BY "tags"."id" LIMIT $2`)).WithArgs(tag.Name, 1).WillReturnError(errors.New("unknown error"))
						mock.ExpectRollback()
					}
				}
			}
		})

		It("should return an error", func() {
			err := handlers.ProcessTags(gormDb, &testRun)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown error"))
		})
	})

	Context("when UpdateTestRun handler is invoked", func() {
		It("and test run does not exist, it should return 404", func() {
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName"})
			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"test_runs\" WHERE id = $1 ORDER BY \"test_runs\".\"id\" LIMIT $2")).
				WithArgs("123", 1).
				WillReturnRows(rows)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = append(c.Params, gin.Param{Key: "id", Value: "123"})
			handler := handlers.NewHandler(gormDb)
			handler.UpdateTestRun(c)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("and test run exists, it should return 200 OK", func() {

			expectedTestRun := models.TestRun{
				ID:              1,
				TestProjectName: "TestProject",
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				SuiteRuns: []models.SuiteRun{
					{
						ID:        1,
						TestRunID: 1,
						SuiteName: "TestSuite",
						StartTime: time.Now(),
						EndTime:   time.Now(),
						SpecRuns: []models.SpecRun{
							{
								ID:              1,
								SuiteID:         1,
								SpecDescription: "TestSpec",
								Status:          "Passed",
								Message:         "",
								StartTime:       time.Now(),
								EndTime:         time.Now(),
							},
						},
					},
				},
			}

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs("1", 1).
				WillReturnRows(mock.NewRows([]string{"id", "test_project_name", "test_seed"}).
					AddRow(expectedTestRun.ID, expectedTestRun.TestProjectName, expectedTestRun.TestSeed))

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create a new request with JSON payload
			jsonStr := []byte(`{"id": 1, "test_project_name":"Updated Project"}`)
			req, err := http.NewRequest("PUT", "/endpoint", bytes.NewBuffer(jsonStr))
			if err != nil {
				fmt.Printf("%v", err)
			}

			req.Header.Set("Content-Type", "application/json")

			c.Request = req
			c.Params = append(c.Params, gin.Param{Key: "id", Value: "1"})
			handler := handlers.NewHandler(gormDb)
			handler.UpdateTestRun(c)

			// Check the response status code
			Expect(w.Code).To(Equal(http.StatusOK))

		})

		It("with wrong POST payload, it should return status 200 OK", func() {

			expectedTestRun := models.TestRun{
				ID:              1,
				TestProjectName: "TestProject",
				StartTime:       time.Now(),
				EndTime:         time.Now(),
			}

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs("1", 1).
				WillReturnRows(mock.NewRows([]string{"id", "test_project_name", "test_seed", "start_time", "end_time"}).
					AddRow(expectedTestRun.ID, expectedTestRun.TestProjectName, expectedTestRun.TestSeed, expectedTestRun.StartTime, expectedTestRun.EndTime))

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonStr := []byte(`{"BAD_PAYLOAD_KEY": "BAD_VALUE"}`)

			req, err := http.NewRequest("POST", "/endpoint", bytes.NewBuffer(jsonStr))
			if err != nil {
				fmt.Printf("%v", err)
			}

			req.Header.Set("Content-Type", "application/json")

			c.Request = req
			c.Params = append(c.Params, gin.Param{Key: "id", Value: "1"})
			handler := handlers.NewHandler(gormDb)
			handler.UpdateTestRun(c)

			// Create a map to represent the response
			var responseBody models.TestRun
			err = json.Unmarshal(w.Body.Bytes(), &responseBody)

			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(expectedTestRun.ID).To(Equal(responseBody.ID))
			Expect(expectedTestRun.SuiteRuns).To(Equal(responseBody.SuiteRuns))
			Expect(expectedTestRun.TestProjectName).To(Equal(responseBody.TestProjectName))
			Expect(expectedTestRun.StartTime).To(BeTemporally("==", responseBody.StartTime))
			Expect(expectedTestRun.EndTime).To(BeTemporally("==", responseBody.EndTime))
			Expect(expectedTestRun.TestSeed).To(Equal(responseBody.TestSeed))

		})

		It("with invalid JSON payload, it should return error", func() {

			expectedTestRun := models.TestRun{
				ID:              1,
				TestProjectName: "TestProject",
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				SuiteRuns: []models.SuiteRun{
					{
						ID:        1,
						TestRunID: 1,
						SuiteName: "TestSuite",
						StartTime: time.Now(),
						EndTime:   time.Now(),
						SpecRuns: []models.SpecRun{
							{
								ID:              1,
								SuiteID:         1,
								SpecDescription: "TestSpec",
								Status:          "Passed",
								Message:         "",
								StartTime:       time.Now(),
								EndTime:         time.Now(),
							},
						},
					},
				},
			}

			//mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs("1", 1).WillReturnRows(mock.NewRows([]string{"id", "test_project_name", "test_seed"}).
				AddRow(expectedTestRun.ID, expectedTestRun.TestProjectName, expectedTestRun.TestSeed))
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create a new request with JSON payload
			jsonStr := []byte(`{"id": 1, "test_project_name123":"Updated Project"}`)

			req, err := http.NewRequest("POST", "/endpoint", bytes.NewBuffer(jsonStr))
			if err != nil {
				fmt.Printf("%v", err)
			}

			req.Header.Set("Content-Type", "application/json")

			c.Request = req
			c.Params = append(c.Params, gin.Param{Key: "id", Value: "1"})
			handler := handlers.NewHandler(gormDb)
			handler.UpdateTestRun(c)

			// Create a map to represent the response
			var responseBody models.TestRun
			err = json.Unmarshal(w.Body.Bytes(), &responseBody)

			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

		})
	})

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

	Context("When Ping handler is invoked", func() {
		It("it should return an HTTP status code of 200, indicating that the 'Fern Reporter' service is operational", func() {
			gin.SetMode(gin.TestMode)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			handler := handlers.NewHandler(gormDb)
			r.GET("/ping", handler.Ping)
			c.Request, _ = http.NewRequest(http.MethodGet, "/ping", nil)
			r.ServeHTTP(w, c.Request)

			Expect(w.Code).To(Equal(http.StatusOK))
			expectedBody := `{"message":"Fern Reporter is running!"}`
			Expect(w.Body.String()).To(Equal(expectedBody))
		})

	})
})
