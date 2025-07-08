package project_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/pkg/api/handlers/project"
	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db     *sql.DB
	gormDb *gorm.DB
	mock   sqlmock.Sqlmock
	err    error
)

var _ = BeforeEach(func() {
	db, mock, err = sqlmock.New()
	Expect(err).NotTo(HaveOccurred())

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 db,
		PreferSimpleProtocol: true,
	})
	gormDb, err = gorm.Open(dialector, &gorm.Config{})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterEach(func() {
	mock.ExpectClose()
	err := db.Close()
	if err != nil {
		utils.GetLogger().Error("[TEST-ERROR]: Unable to close the db connection: ", err)
	}
})

var _ = Describe("Project Handlers", func() {
	Context("When the report project handler is invoked", func() {
		It("should return all project names in ascending order", func() {
			projectRows := sqlmock.NewRows([]string{"id", "name", "uuid"}).
				AddRow(1, "ProjectA", "uuid-1").
				AddRow(2, "ProjectF", "uuid-2").
				AddRow(3, "ProjectZ", "uuid-3")

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" ORDER BY name ASC`)).
				WillReturnRows(projectRows)

			gin.SetMode(gin.TestMode)
			router := gin.Default()
			handler := project.NewProjectHandler(gormDb)
			router.GET("/api/reports/projects", handler.GetAllProjectsForReport)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/reports/projects", nil)
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			expectedJSON := `{"projects":[{"id":1,"name":"ProjectA","uuid":"uuid-1"},{"id":2,"name":"ProjectF","uuid":"uuid-2"},{"id":3,"name":"ProjectZ","uuid":"uuid-3"}]}`
			Expect(w.Body.String()).To(MatchJSON(expectedJSON))
		})
	})

	Context("when save project is invoked", func() {
		projectID := "96ad860-2a9a-504f-8861-aeafd0b2ae29"
		projRequest := models.ProjectDetails{
			Name:     "First Project",
			TeamName: "Team A",
			Comment:  "Sample Comment",
		}
		It("with proper details, it should create one and return project object back", func() {

			reqBody, err := json.Marshal(projRequest)
			if err != nil {
				utils.GetLogger().Error("[TEST-ERROR]: Error Marshaling project request: ", err)
				return
			}
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE name = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(projRequest.Name, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "project_details" ("name","team_name","comment","updated_at") VALUES ($1,$2,$3,$4) RETURNING *`)).
				WithArgs(projRequest.Name, projRequest.TeamName, projRequest.Comment, sqlmock.AnyArg()).
				WillReturnRows(
					sqlmock.NewRows([]string{"id", "uuid", "name", "team_name", "comment", "created_at", "updated_at"}).
						AddRow(1, projectID, projRequest.Name, projRequest.TeamName, projRequest.Comment, time.Now(), time.Now()),
				)

			mock.ExpectCommit()

			req := httptest.NewRequest(http.MethodPost, "/api/project", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			handler := project.NewProjectHandler(gormDb)

			handler.CreateProject(c)
			Expect(w.Code).To(Equal(201))

			var project models.ProjectDetails
			if err := json.NewDecoder(w.Body).Decode(&project); err != nil {
				Fail(err.Error())
			}

			Expect(project.UUID).To(Equal(projectID))
			Expect(project.Name).To(Equal(projRequest.Name))
			Expect(project.TeamName).To(Equal(projRequest.TeamName))
			Expect(project.Comment).To(Equal(projRequest.Comment))
		})
		It("with duplicate project name, it should return error", func() {
			reqBody, err := json.Marshal(projRequest)
			if err != nil {
				utils.GetLogger().Error("[TEST-ERROR]: Error Marshaling project request: ", err)
				return
			}
			projectRows := sqlmock.NewRows([]string{"id", "uuid", "name", "team_name", "comment"}).
				AddRow(1, projectID, projRequest.Name, projRequest.TeamName, projRequest.Comment)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE name = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(projRequest.Name, 1).
				WillReturnRows(projectRows)

			req := httptest.NewRequest(http.MethodPost, "/api/project", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			handler := project.NewProjectHandler(gormDb)

			handler.CreateProject(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response["error"]).To(Equal("Project Name already exists"))
		})
	})

	Context("when update project is invoked", func() {
		projectID := "96ad860-2a9a-504f-8861-aeafd0b2ae29"
		projRequest := models.ProjectDetails{
			ID:       1,
			UUID:     projectID,
			Name:     "First Project",
			TeamName: "Team A",
			Comment:  "Sample Comment",
		}
		It("with proper details, it should update one and return project object back", func() {

			projectRows := sqlmock.NewRows([]string{"id", "uuid", "name", "team_name", "comment"}).
				AddRow(1, projectID, projRequest.Name, projRequest.TeamName, projRequest.Comment)

			reqBody, err := json.Marshal(projRequest)
			if err != nil {
				utils.GetLogger().Error("[TEST-ERROR]: Error Marshaling project request: ", err)
				return
			}

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(projectID, 1).
				WillReturnRows(projectRows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "project_details" WHERE name = $1 AND uuid != $2`)).
				WithArgs(projRequest.Name, projRequest.UUID).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(`UPDATE "project_details" SET "name"=$1,"team_name"=$2,"comment"=$3,"created_at"=$4,"updated_at"=$5 WHERE "id" = $6`)).
				WithArgs(projRequest.Name, projRequest.TeamName, projRequest.Comment, sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			req := httptest.NewRequest(http.MethodPost, "/api/project", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = append(c.Params, gin.Param{Key: "uuid", Value: projectID})
			c.Request = req
			handler := project.NewProjectHandler(gormDb)

			handler.UpdateProject(c)
			Expect(w.Code).To(Equal(200))

			var project models.ProjectDetails
			if err := json.NewDecoder(w.Body).Decode(&project); err != nil {
				Fail(err.Error())
			}
			Expect(project.Name).To(Equal(projRequest.Name))
			Expect(project.TeamName).To(Equal(projRequest.TeamName))
			Expect(project.Comment).To(Equal(projRequest.Comment))
		})
		It("with duplicate project name, it should not update the details and return 400", func() {

			projectRows := sqlmock.NewRows([]string{"id", "uuid", "name", "team_name", "comment"}).
				AddRow(1, projectID, projRequest.Name, projRequest.TeamName, projRequest.Comment)

			reqBody, err := json.Marshal(projRequest)
			if err != nil {
				utils.GetLogger().Error("[TEST-ERROR]: Error Marshaling project request: ", err)
				return
			}

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(projectID, 1).
				WillReturnRows(projectRows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "project_details" WHERE name = $1 AND uuid != $2`)).
				WithArgs(projRequest.Name, projRequest.UUID).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			req := httptest.NewRequest(http.MethodPost, "/api/project", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = append(c.Params, gin.Param{Key: "uuid", Value: projectID})
			c.Request = req
			handler := project.NewProjectHandler(gormDb)

			handler.UpdateProject(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response["error"]).To(Equal("Project name already exists"))
		})
		It("for invalid project UUID, it should not update return 404", func() {

			reqBody, err := json.Marshal(projRequest)
			if err != nil {
				utils.GetLogger().Error("[TEST-ERROR]: Error Marshaling project request: ", err)
				return
			}

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(projectID, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			req := httptest.NewRequest(http.MethodPost, "/api/project", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = append(c.Params, gin.Param{Key: "uuid", Value: projectID})
			c.Request = req
			handler := project.NewProjectHandler(gormDb)

			handler.UpdateProject(c)
			Expect(w.Code).To(Equal(404))
			Expect(w.Body.String()).To(Equal(`{"error":"Project not found"}`))
		})
	})

	Context("when delete project is invoked", func() {
		projectID := "96ad860-2a9a-504f-8861-aeafd0b2ae29"
		projRequest := models.ProjectDetails{
			ID:       1,
			UUID:     projectID,
			Name:     "First Project",
			TeamName: "Team A",
			Comment:  "Sample Comment",
		}
		It("with proper details, it should delete 200", func() {

			reqBody, err := json.Marshal(projRequest)
			if err != nil {
				utils.GetLogger().Error("[TEST-ERROR]: Error Marshaling project request: ", err)
				return
			}

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "project_details" WHERE uuid = $1`)).
				WithArgs(projectID).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			req := httptest.NewRequest(http.MethodPost, "/api/project", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = append(c.Params, gin.Param{Key: "uuid", Value: projectID})
			c.Request = req
			handler := project.NewProjectHandler(gormDb)

			handler.DeleteProject(c)
			Expect(w.Code).To(Equal(200))
			Expect(w.Body.String()).To(Equal(`{"message":"Project ID 96ad860-2a9a-504f-8861-aeafd0b2ae29 deleted"}`))
		})
	})
})
