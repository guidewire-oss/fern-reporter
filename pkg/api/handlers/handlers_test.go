package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/driver/sqlite"
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

var _ = Describe("ParseTagName", func() {
	It("parses category and value from tagName with colon", func() {
		tag := handlers.ParseTagName("priority:high")
		Expect(tag.Name).To(Equal("priority:high"))
		Expect(tag.Category).To(Equal("priority"))
		Expect(tag.Value).To(Equal("high"))
	})

	It("sets only name if no colon is present", func() {
		tag := handlers.ParseTagName("smoke")
		Expect(tag.Name).To(Equal("smoke"))
		Expect(tag.Category).To(BeEmpty())
		Expect(tag.Value).To(Equal("smoke"))
	})

	It("handles empty tagName", func() {
		tag := handlers.ParseTagName("")
		Expect(tag.Name).To(Equal(""))
		Expect(tag.Category).To(BeEmpty())
		Expect(tag.Value).To(BeEmpty())
	})

	It("handles multiple colons, splitting only on the first", func() {
		tag := handlers.ParseTagName("env:prod:us")
		Expect(tag.Name).To(Equal("env:prod:us"))
		Expect(tag.Category).To(Equal("env"))
		Expect(tag.Value).To(Equal("prod:us"))
	})
})

type TestDataSetup struct {
	DB *gorm.DB
}

var _ = Describe("Handlers", func() {
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
		err := db.Close()
		if err != nil {
			fmt.Printf("Unable to close the db connection %s", err.Error())
		}
	})

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
		expectedProject := models.ProjectDetails{
			ID:       1,
			UUID:     "996ad860-2a9a-504f-8861-aeafd0b2ae29",
			Name:     "Polaris Unit Test",
			TeamName: "Nova",
			Comment:  "",
		}
		It("and test run doesn't exist, it should create one and return 201 OK", func() {
			expectedTestRun := models.TestRun{
				ID:                0,
				TestProjectName:   "TestProject",
				TestProjectID:     "996ad860-2a9a-504f-8861-aeafd0b2ae29",
				StartTime:         time.Time{},
				EndTime:           time.Time{},
				GitBranch:         "main",
				GitSha:            "abcdef",
				BuildTriggerActor: "Actor Name",
				BuildUrl:          "https://someurl.com",
				TestSeed:          0,
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

			rows := sqlmock.NewRows([]string{"ID", "UUID"}).
				AddRow(expectedProject.ID, expectedProject.UUID)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs("996ad860-2a9a-504f-8861-aeafd0b2ae29", 1).
				WillReturnRows(rows)

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "test_runs" ("test_project_name","project_id","test_seed","start_time","end_time","git_branch","git_sha","build_trigger_actor","build_url") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING "id"`)).
				WithArgs(expectedTestRun.TestProjectName, expectedProject.ID, expectedTestRun.TestSeed, expectedTestRun.StartTime, expectedTestRun.EndTime, expectedTestRun.GitBranch, expectedTestRun.GitSha, expectedTestRun.BuildTriggerActor, expectedTestRun.BuildUrl).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectCommit()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create a new request with JSON payload
			jsonStr := fmt.Sprintf(`{"id": 0, "test_project_name":"TestProject", "test_project_id":"996ad860-2a9a-504f-8861-aeafd0b2ae29", "git_branch": "%s", "git_sha": "%s", "build_trigger_actor": "%s", "build_url": "%s"}`, expectedTestRun.GitBranch, expectedTestRun.GitSha, expectedTestRun.BuildTriggerActor, expectedTestRun.BuildUrl)
			req, err := http.NewRequest("POST", "/", bytes.NewBuffer([]byte(jsonStr)))
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
			Expect(testRun.GitBranch).To(Equal(expectedTestRun.GitBranch))
			Expect(testRun.GitSha).To(Equal(expectedTestRun.GitSha))
			Expect(testRun.BuildTriggerActor).To(Equal(expectedTestRun.BuildTriggerActor))
			Expect(testRun.BuildUrl).To(Equal(expectedTestRun.BuildUrl))
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
				ID:                1,
				TestProjectName:   "TestProject",
				StartTime:         time.Time{},
				EndTime:           time.Time{},
				GitBranch:         "main",
				GitSha:            "abcdef",
				BuildTriggerActor: "Actor Name",
				BuildUrl:          "https://someurl.com",
				TestSeed:          1,
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

			rows := sqlmock.NewRows([]string{"ID", "UUID"}).
				AddRow(expectedProject.ID, expectedProject.UUID)
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs("996ad860-2a9a-504f-8861-aeafd0b2ae29", 1).
				WillReturnRows(rows)

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs(expectedTestRun.ID, 1).
				WillReturnError(errors.New("Record not found DB error"))

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create a new request with JSON payload
			jsonStr := fmt.Sprintf(`{"id": 1, "test_project_name":"TestProject", "test_project_id":"996ad860-2a9a-504f-8861-aeafd0b2ae29", "git_branch": "%s", "git_sha": "%s", "build_trigger_actor": "%s", "build_url": "%s"}`, expectedTestRun.GitBranch, expectedTestRun.GitSha, expectedTestRun.BuildTriggerActor, expectedTestRun.BuildUrl)
			req, err := http.NewRequest("POST", "/", bytes.NewBuffer([]byte(jsonStr)))
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
			testRun := models.TestRun{
				ID:                1,
				TestProjectName:   "TestProject",
				TestProjectID:     "996ad860-2a9a-504f-8861-aeafd0b2ae29",
				StartTime:         time.Time{},
				EndTime:           time.Time{},
				GitBranch:         "main",
				GitSha:            "abcdef",
				BuildTriggerActor: "Actor Name",
				BuildUrl:          "https://someurl.com",
				TestSeed:          1,
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

			// mock.ExpectBegin()
			testRuns := sqlmock.NewRows([]string{"id", "TestProjectName"}).
				AddRow(1, "project 1")

			rows := sqlmock.NewRows([]string{"ID", "UUID"}).
				AddRow(expectedProject.ID, expectedProject.UUID)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs("996ad860-2a9a-504f-8861-aeafd0b2ae29", 1).
				WillReturnRows(rows)

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
			testRun := models.TestRun{
				ID:                1,
				TestProjectName:   "TestProject",
				TestProjectID:     "996ad860-2a9a-504f-8861-aeafd0b2ae29",
				StartTime:         time.Time{},
				EndTime:           time.Time{},
				GitBranch:         "main",
				GitSha:            "abcdef",
				BuildTriggerActor: "Actor Name",
				BuildUrl:          "https://someurl.com",
				TestSeed:          1,
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

			projectRows := sqlmock.NewRows([]string{"ID", "UUID"}).
				AddRow(expectedProject.ID, expectedProject.UUID)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs("996ad860-2a9a-504f-8861-aeafd0b2ae29", 1).
				WillReturnRows(projectRows)

			// mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs(testRun.ID, 1).
				WillReturnRows(testRuns)

			rows := sqlmock.NewRows([]string{"id"})

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tags" WHERE name = $1 ORDER BY "tags"."id" LIMIT $2`)).WithArgs("TagName", 1).WillReturnRows(rows)

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tags" ("name","category","value") VALUES ($1,$2,$3) RETURNING "id"`)).
				WithArgs("TagName", "", "TagName").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectCommit()

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(`UPDATE "test_runs" SET "test_project_name"=$1,"project_id"=$2,"test_seed"=$3,"start_time"=$4,"end_time"=$5,"git_branch"=$6,"git_sha"=$7,"build_trigger_actor"=$8,"build_url"=$9 WHERE "id" = $10`)).
				WithArgs(testRun.TestProjectName, expectedProject.ID, testRun.TestSeed, testRun.StartTime, testRun.EndTime, testRun.GitBranch, testRun.GitSha, testRun.BuildTriggerActor, testRun.BuildUrl, testRun.ID).
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
		testRun := models.TestRun{
			ID:                0,
			TestProjectName:   "TestProject",
			StartTime:         time.Time{},
			EndTime:           time.Time{},
			GitBranch:         "main",
			GitSha:            "abcdef",
			BuildTriggerActor: "Actor Name",
			BuildUrl:          "https://someurl.com",
			TestSeed:          0,
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
		testRun := models.TestRun{
			ID:                0,
			TestProjectName:   "TestProject",
			StartTime:         time.Time{},
			EndTime:           time.Time{},
			GitBranch:         "main",
			GitSha:            "abcdef",
			BuildTriggerActor: "Actor Name",
			BuildUrl:          "https://someurl.com",
			TestSeed:          0,
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
		testRun := models.TestRun{
			ID:                0,
			TestProjectName:   "TestProject",
			StartTime:         time.Time{},
			EndTime:           time.Time{},
			GitBranch:         "main",
			GitSha:            "abcdef",
			BuildTriggerActor: "Actor Name",
			BuildUrl:          "https://someurl.com",
			TestSeed:          0,
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
			tag := models.Tag{Name: "NewTag", Category: "", Value: "NewTag"}
			specRun := models.SpecRun{Tags: []models.Tag{tag}}
			suiteRun := models.SuiteRun{SpecRuns: []models.SpecRun{specRun}}
			testRun := &models.TestRun{SuiteRuns: []models.SuiteRun{suiteRun}}

			rows := sqlmock.NewRows([]string{"id"})

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tags" WHERE name = $1 ORDER BY "tags"."id" LIMIT $2`)).WithArgs(tag.Name, 1).WillReturnRows(rows)
			mock.ExpectBegin()

			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tags" ("name","category","value") VALUES ($1,$2,$3) RETURNING "id"`)).
				WithArgs(tag.Name, tag.Category, tag.Value).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
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
		testRun := models.TestRun{
			ID:                0,
			TestProjectName:   "TestProject",
			StartTime:         time.Time{},
			EndTime:           time.Time{},
			GitBranch:         "main",
			GitSha:            "abcdef",
			BuildTriggerActor: "Actor Name",
			BuildUrl:          "https://someurl.com",
			TestSeed:          0,
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
				ID:                1,
				TestProjectName:   "TestProject",
				StartTime:         time.Now(),
				EndTime:           time.Now(),
				GitBranch:         "main",
				GitSha:            "abcdef",
				BuildTriggerActor: "Actor Name",
				BuildUrl:          "https://someurl.com",
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
				WillReturnRows(mock.NewRows([]string{"id", "test_project_name", "test_seed", "git_branch", "git_sha", "build_trigger_actor", "build_url"}).
					AddRow(expectedTestRun.ID, expectedTestRun.TestProjectName, expectedTestRun.TestSeed, expectedTestRun.GitBranch, expectedTestRun.GitSha, expectedTestRun.BuildTriggerActor, expectedTestRun.BuildUrl))

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

			// mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs("1", 1).
				WillReturnRows(mock.NewRows([]string{"id", "test_project_name", "test_seed"}).
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

	Context("When a request is made to get a test summary by project name", func() {
		It("should return the summary of test runs for the project", func() {
			projectID := "1"
			rows := sqlmock.NewRows([]string{"suite_run_id", "suite_name", "test_project_name", "start_time", "total_passed_spec_runs", "total_skipped_spec_runs", "total_spec_runs"}).
				AddRow(1, "TestSuite1", "TestProject", time.Date(2024, 4, 20, 12, 0, 0, 0, time.UTC), 5, 1, 10).
				AddRow(2, "TestSuite2", "TestProject", time.Date(2024, 4, 21, 12, 0, 0, 0, time.UTC), 7, 2, 12)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT suite_runs.id AS suite_run_id, 
			suite_runs.suite_name,
            test_runs.start_time, 
            COUNT(spec_runs.id) FILTER (WHERE spec_runs.status = 'passed') AS total_passed_spec_runs, 
			COUNT(spec_runs.id) FILTER (WHERE spec_runs.status = 'skipped') AS total_skipped_spec_runs, 
            COUNT(spec_runs.id) AS total_spec_runs FROM "test_runs" 
                INNER JOIN suite_runs ON test_runs.id = suite_runs.test_run_id 
                INNER JOIN spec_runs ON suite_runs.id = spec_runs.suite_id 
                WHERE test_runs.project_id = $1 
                GROUP BY suite_runs.id, test_runs.start_time ORDER BY test_runs.start_time`)).WithArgs(projectID).WillReturnRows(rows)

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			handler := handlers.NewHandler(gormDb)

			c.Request, _ = http.NewRequest("GET", "/summary/1/", nil)
			router.GET("/summary/:projectId/", handler.GetTestSummary)
			router.ServeHTTP(w, c.Request)

			Expect(w.Code).To(Equal(http.StatusOK))

			expectedJSON := `[{
				"SuiteRunID": 1,
				"SuiteName": "TestSuite1",
				"TestProjectName": "TestProject",
				"StartTime": "2024-04-20T12:00:00Z",
				"TotalPassedSpecRuns": 5,
				"TotalSkippedSpecRuns": 1,
				"TotalSpecRuns": 10
			}, {
				"SuiteRunID": 2,
				"SuiteName": "TestSuite2",
				"TestProjectName": "TestProject",
				"StartTime": "2024-04-21T12:00:00Z",
				"TotalPassedSpecRuns": 7,
				"TotalSkippedSpecRuns": 2,
				"TotalSpecRuns": 12
			}]`
			Expect(w.Body.String()).To(MatchJSON(expectedJSON))
		})
	})
})

type response struct {
	TestRuns     []models.TestRun `json:"testRuns"`
	ReportHeader string           `json:"reportHeader"`
	Total        int              `json:"total"`
}

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	if err := db.AutoMigrate(&models.ProjectDetails{}, &models.TestRun{}, &models.SuiteRun{}, &models.SpecRun{}, &models.Tag{}); err != nil {
		panic("failed to migrate database: " + err.Error())
	}
	return db
}

func (s *TestDataSetup) InsertTestData(
	projectID uint64, projectName, projectUUID string,
	testRunID uint64, gitBranch, gitSha, testProjectName string, startTime time.Time,
	suiteRunID uint64, suiteName string,
	tagNames []string,
	specDescription, status, message string,
) {
	project := models.ProjectDetails{
		ID:   projectID,
		UUID: projectUUID,
	}

	s.DB.FirstOrCreate(&project, models.ProjectDetails{
		ID: projectID,
	})

	testRun := models.TestRun{
		ID:              testRunID,
		ProjectID:       projectID,
		GitBranch:       gitBranch,
		GitSha:          gitSha,
		StartTime:       startTime,
		TestProjectName: testProjectName,
	}
	s.DB.Create(&testRun)

	suiteRun := models.SuiteRun{
		ID:        suiteRunID,
		TestRunID: testRunID,
		SuiteName: suiteName,
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}
	s.DB.Create(&suiteRun)

	// Create tag records and collect references
	var tags []models.Tag
	for _, name := range tagNames {
		tag := models.Tag{Name: name}
		s.DB.FirstOrCreate(&tag, models.Tag{Name: name}) // avoids duplicate tags if reused
		tags = append(tags, tag)
	}

	specRun := models.SpecRun{
		SuiteID:         suiteRunID,
		SpecDescription: specDescription,
		Status:          status,
		Message:         message,
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		Tags:            tags,
	}
	s.DB.Create(&specRun)

	// Establish many2many relationship
	for i := range tags {
		err := s.DB.Model(&specRun).Association("Tags").Append(tags[i])
		if err != nil {
			return
		}
	}
}

var _ = Describe("ReportTestRunAll", func() {
	var (
		db *gorm.DB
		h  *handlers.Handler
		r  *gin.Engine
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		db = setupTestDB()
		setup := &TestDataSetup{DB: db}
		setup.InsertTestData(
			1, "Test Project 1", "uuid-123",
			1, "main", "abcdef1234567890", "Smoke Project", time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
			1, "suite1",
			[]string{"smoke", "regression"},
			"Test spec", "passed", "",
		)
		setup.InsertTestData(
			2, "Test Project 2", "uuid-234",
			2, "chore/test", "abcdef1234567890", "Smoke Project", time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
			2, "suite2",
			[]string{"smoke", "regression"},
			"Test spec", "passed", "",
		)
		setup.InsertTestData(
			1, "Test Project 1", "uuid-123",
			3, "main", "abcdef1sdaadw", "Smoke Project", time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
			3, "suite3",
			[]string{"acceptance", "regression"},
			"Test spec", "passed", "",
		)

		h = handlers.NewHandler(db)
		r = gin.Default()
		r.GET("/reports/testruns", h.ReportTestRunAll)
	})

	It("returns all test runs with no filters", func() {
		req, _ := http.NewRequest("GET", "/reports/testruns", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		var resp response
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Total).To(Equal(3))
		Expect(resp.TestRuns[0].GitBranch).To(Equal("main"))
	})

	It("filters test runs by project_id", func() {
		req, _ := http.NewRequest("GET", "/reports/testruns?project_id=1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		var resp response
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Total).To(Equal(2))
		Expect(resp.TestRuns[0].ProjectID).To(Equal(uint64(1)))
		Expect(resp.TestRuns[1].ProjectID).To(Equal(uint64(1)))
	})

	It("returns no test runs for non-matching project_id", func() {
		req, _ := http.NewRequest("GET", "/reports/testruns?project_id=999", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		var resp response
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Total).To(Equal(0))
	})

	It("filters test runs by tag name", func() {
		req, _ := http.NewRequest("GET", "/reports/testruns?tags=smoke", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var resp response
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Total).To(Equal(2))
		Expect(resp.TestRuns[0].GitSha).To(Equal("abcdef1234567890"))
		Expect(resp.TestRuns[0].ProjectID).To(Equal(uint64(1)))
		Expect(resp.TestRuns[1].ProjectID).To(Equal(uint64(2)))
	})

	It("filters test runs by multiple query filters", func() {
		startTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
		req, _ := http.NewRequest("GET", fmt.Sprintf("/reports/testruns?project_id=1&git_sha=abcdef1234567890&start_time=%s&tags=smoke", startTime.Format("2006-01-02")), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var resp response
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Total).To(Equal(1))
		Expect(resp.TestRuns[0].GitSha).To(Equal("abcdef1234567890"))
	})

	It("should give an error when filters test runs by start time and end time with invalid range", func() {
		startTime := time.Now()
		endTime := time.Now().Add(-48 * time.Hour)
		req, _ := http.NewRequest("GET", fmt.Sprintf("/reports/testruns?start_time=%s&end_time=%s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02")), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusBadRequest))

		var resp response
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Total).To(Equal(0))
	})

	It("filters test runs by multiple query filters and multiple tags", func() {
		req, _ := http.NewRequest("GET", "/reports/testruns?git_sha=abcdef1234567890&tags=smoke,regression", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var resp response
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Total).To(Equal(2))
		Expect(resp.TestRuns[0].GitSha).To(Equal("abcdef1234567890"))
		Expect(resp.TestRuns[0].ProjectID).To(Equal(uint64(1)))
		Expect(resp.TestRuns[1].ProjectID).To(Equal(uint64(2)))
	})

	It("should give an error when filtering test runs by a non existent filter ", func() {
		req, _ := http.NewRequest("GET", "/reports/testruns?nonexistentfilter=abc", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusBadRequest))

		var resp response
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Total).To(Equal(0))
	})
})
