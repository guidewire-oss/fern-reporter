package user_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/pkg/api/handlers/user"
	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"regexp"
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
		utils.Log.Error("[TEST-ERROR]: Unable to close the db connection: ", err)
	}
})

var _ = Describe("User Preference Handlers", Ordered, func() {
	projectId := "96ad860-2a9a-504f-8861-aeafd0b2ae29"
	ucookie := "5c0fc06d-26d9-4202-a1f3-2d024e957171"

	Context("when save favourite project is invoked", func() {

		var favRequest = user.FavouriteProjectRequest{
			Favourite: projectId,
		}

		It("and save the favorite project, it should create one and return 201 OK", func() {
			reqBody, err := json.Marshal(favRequest)
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal favourite project request", err)
				return
			}

			project_rows := sqlmock.NewRows([]string{"ID", "UUID"}).
				AddRow(1, projectId)

			user_rows := sqlmock.NewRows([]string{"ID", "Cookie"}).
				AddRow(1, ucookie)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(favRequest.Favourite, 1).
				WillReturnRows(project_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "preferred_projects" WHERE user_id = $1 and project_id = $2`)).
				WithArgs(1, 1).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "preferred_projects" ("user_id","project_id","group_id") VALUES ($1,$2,$3) RETURNING "id"`)).
				WithArgs(1, 1, nil).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectCommit()

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/user/preference", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)

			handler.SaveFavouriteProject(c)
			Expect(w.Code).To(Equal(201))
		})
		It("for same favorite project, it should not create one and return 201 OK", func() {
			reqBody, err := json.Marshal(favRequest)
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal favourite project request", err)
				return
			}

			project_rows := sqlmock.NewRows([]string{"ID", "UUID"}).
				AddRow(1, projectId)

			user_rows := sqlmock.NewRows([]string{"ID", "Cookie"}).
				AddRow(1, ucookie)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(favRequest.Favourite, 1).
				WillReturnRows(project_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "preferred_projects" WHERE user_id = $1 and project_id = $2`)).
				WithArgs(1, 1).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/user/preference", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)

			handler.SaveFavouriteProject(c)
			Expect(w.Code).To(Equal(201))
		})
		It("for invalid project, it should return Project ID not found (404)", func() {
			reqBody, err := json.Marshal(favRequest)
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal favourite project request", err)
				return
			}
			user_rows := sqlmock.NewRows([]string{"ID", "Cookie"}).
				AddRow(1, ucookie)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(favRequest.Favourite, 1).
				WillReturnError(errors.New("Record not found"))

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/user/preference", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)

			handler.SaveFavouriteProject(c)
			Expect(w.Code).To(Equal(404))
		})
	})
	Context("when delete favourite project is invoked", func() {

		It("and delete the favorite project, it should delete and return 200 OK", func() {
			reqBody, err := json.Marshal("")
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal favourite project request", err)
				return
			}

			project_rows := sqlmock.NewRows([]string{"ID", "UUID"}).
				AddRow(1, projectId)

			user_rows := sqlmock.NewRows([]string{"ID", "Cookie"}).
				AddRow(1, ucookie)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(projectId, 1).
				WillReturnRows(project_rows)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "preferred_projects" WHERE user_id = $1 and project_id = $2`)).
				WithArgs(1, 1).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			mock.ExpectClose()

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/user/favourite/", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = append(c.Params, gin.Param{Key: "projectUUID", Value: projectId})
			c.Request = req

			handler := user.NewUserHandler(gormDb)

			handler.DeleteFavouriteProject(c)
			Expect(w.Code).To(Equal(200))
			Expect(w.Body.String()).To(Equal(`{"message":"Favourite Project 96ad860-2a9a-504f-8861-aeafd0b2ae29 deleted successfully"}`))
		})
		It("for invalid project, it should return Project ID not found (404)", func() {
			reqBody, err := json.Marshal("")
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal favourite project request", err)
				return
			}

			user_rows := sqlmock.NewRows([]string{"ID", "Cookie"}).
				AddRow(1, ucookie)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(projectId, 1).
				WillReturnError(errors.New("Record not found"))

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/user/favourite/", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = append(c.Params, gin.Param{Key: "projectUUID", Value: projectId})
			c.Request = req

			handler := user.NewUserHandler(gormDb)

			handler.DeleteFavouriteProject(c)
			Expect(w.Code).To(Equal(404))
		})
	})

	Context("when get favourite project is invoked", func() {
		It("will return favourite project list and return 200 OK", func() {
			projectUUIDs := []string{"96ad8601-2a9a-504f-8861-aeafd0b2ae29", "59e06cf8-f390-5093-af2e-3685be593a25"}

			user_rows := sqlmock.NewRows([]string{"id", "cookie"}).
				AddRow(1, ucookie)

			project_rows := sqlmock.NewRows([]string{"uuid"}).
				AddRow(projectUUIDs[0]).
				AddRow(projectUUIDs[1])

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT "project_details"."uuid" FROM "preferred_projects" 
			JOIN project_details ON preferred_projects.project_id = project_details.id 
			WHERE preferred_projects.user_id = $1 AND preferred_projects.group_id IS NULL`)).
				WithArgs(1).
				WillReturnRows(project_rows)

			req := httptest.NewRequest(http.MethodGet, "/api/user/favourite", nil)
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)
			handler.GetFavouriteProject(c)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response []string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response).To(Equal(projectUUIDs))
		})

		It("will return 404 if user is not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnError(errors.New("Record not found"))

			req := httptest.NewRequest(http.MethodGet, "/api/user/favourite", nil)
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)
			handler.GetFavouriteProject(c)

			Expect(w.Code).To(Equal(http.StatusNotFound))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response["error"]).To(ContainSubstring("User ID not found"))
		})

		It("will return 500 if there is an error fetching favourite projects", func() {
			user_rows := sqlmock.NewRows([]string{"id", "cookie"}).
				AddRow(1, ucookie)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT "project_details"."uuid" FROM "preferred_projects" 
			JOIN project_details ON preferred_projects.project_id = project_details.id 
			WHERE preferred_projects.user_id = $1 AND preferred_projects.group_id IS NULL`)).
				WithArgs(1).
				WillReturnError(errors.New("Database error"))

			req := httptest.NewRequest(http.MethodGet, "/api/user/favourite", nil)
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)
			handler.GetFavouriteProject(c)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response["error"]).To(Equal("error fetching favourite project uuids"))
		})
	})

	Context("when save user preference is invoked", func() {
		var userPrefRequest = user.UserPreferenceRequest{
			IsDark:   true,
			Timezone: "America/New_York",
		}

		It("and save the user preference, it should save and return 202 OK", func() {
			reqBody, err := json.Marshal(userPrefRequest)
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal user preference request", err)
				return
			}

			user_rows := sqlmock.NewRows([]string{"ID", "Cookie"}).
				AddRow(1, ucookie)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(`UPDATE "app_users" SET "is_dark"=$1,"timezone"=$2,"updated_at"=$3 WHERE cookie = $4`)).
				WithArgs(true, "America/New_York", sqlmock.AnyArg(), ucookie).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			mock.ExpectClose()

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/user/preference/", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)

			handler.SaveUserPreference(c)
			Expect(w.Code).To(Equal(202))
			Expect(w.Body.String()).To(ContainSubstring("{\"status\":\"success\"}"))
		})
	})

	Context("when get user preference is invoked", func() {
		It("will return user preference details", func() {
			reqBody, err := json.Marshal("")
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal user preference request", err)
				return
			}

			user_rows := sqlmock.NewRows([]string{"ID", "IsDark", "Timezone", "Cookie"}).
				AddRow(1, true, "America/New_York", ucookie)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/user/preference", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)

			handler.GetUserPreference(c)
			var responseBody models.AppUser
			err = json.Unmarshal(w.Body.Bytes(), &responseBody)

			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(true).To(Equal(responseBody.IsDark))
			Expect("America/New_York").To(Equal(responseBody.Timezone))
		})
	})

	Context("when save preferred project is invoked", func() {

		prefRequest := user.PreferredRequest{
			Preferred: []struct {
				GroupID   uint64   `json:"group_id"`
				GroupName string   `json:"group_name"`
				Projects  []string `json:"projects"`
			}{
				{
					GroupID:   0, // Empty group (new group creation)
					GroupName: "First Favorite Group",
					Projects:  []string{projectId},
				},
			},
		}
		It("with a new group and project, it should create one and return 201 OK", func() {
			reqBody, err := json.Marshal(prefRequest)
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal favourite project request", err)
				return
			}

			user_rows := sqlmock.NewRows([]string{"ID", "IsDark", "Timezone", "Cookie"}).
				AddRow(1, true, "America/New_York", ucookie)

			//project_group_rows := sqlmock.NewRows([]string{"GroupID", "UserID", "GroupName"}).
			//	AddRow(1, 1, "First Group")

			project_rows := sqlmock.NewRows([]string{"ID", "UUID"}).
				AddRow(1, projectId)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_groups" WHERE user_id = $1 AND group_id = $2 ORDER BY "project_groups"."group_id" LIMIT $3`)).
				WithArgs(1, 0, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "project_groups" ("user_id","group_name") VALUES ($1,$2) RETURNING "group_id"`)).
				WithArgs(1, "First Favorite Group").
				WillReturnRows(sqlmock.NewRows([]string{"group_id"}).AddRow(1))

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(projectId, 1).
				WillReturnRows(project_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "preferred_projects" ("user_id","project_id","group_id") VALUES ($1,$2,$3) RETURNING "id"`)).
				WithArgs(1, 1, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			mock.ExpectCommit()

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/user/preference", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)

			handler.SavePreferredProject(c)

			Expect(w.Code).To(Equal(201))
			Expect(w.Body.String()).To(ContainSubstring("{\"status\":\"success\"}"))

		})
		It("with an existing group and project, it should create one and return 201 OK", func() {

			prefRequest := user.PreferredRequest{
				Preferred: []struct {
					GroupID   uint64   `json:"group_id"`
					GroupName string   `json:"group_name"`
					Projects  []string `json:"projects"`
				}{
					{
						GroupID:   1,
						GroupName: "First Favorite Group",
						Projects:  []string{projectId},
					},
				},
			}

			reqBody, err := json.Marshal(prefRequest)
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal favourite project request", err)
				return
			}

			user_rows := sqlmock.NewRows([]string{"ID", "IsDark", "Timezone", "Cookie"}).
				AddRow(1, true, "America/New_York", ucookie)

			project_group_rows := sqlmock.NewRows([]string{"GroupID", "UserID", "GroupName"}).
				AddRow(1, 1, "First Group")

			project_rows := sqlmock.NewRows([]string{"ID", "UUID"}).
				AddRow(1, projectId)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectBegin()

			//DELETE FROM "preferred_projects" WHERE user_id = 1 AND group_id IN (2)
			mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "preferred_projects" WHERE user_id = $1 AND group_id IN ($2)`)).
				WithArgs(1, 1).
				WillReturnResult(sqlmock.NewResult(0, 1))

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_groups" WHERE user_id = $1 AND group_id = $2 ORDER BY "project_groups"."group_id" LIMIT $3`)).
				WithArgs(1, 1, 1).
				WillReturnRows(project_group_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE uuid = $1 ORDER BY "project_details"."id" LIMIT $2`)).
				WithArgs(projectId, 1).
				WillReturnRows(project_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "preferred_projects" ("user_id","project_id","group_id") VALUES ($1,$2,$3) RETURNING "id"`)).
				WithArgs(1, 1, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			mock.ExpectCommit()

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/user/preference", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)

			handler.SavePreferredProject(c)

			Expect(w.Code).To(Equal(201))
			Expect(w.Body.String()).To(ContainSubstring("{\"status\":\"success\"}"))

		})
	})

	Context("when get preferred project is invoked", func() {

		It("will return preferred project details", func() {
			reqBody, err := json.Marshal("")
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal user preference request", err)
				return
			}

			user_rows := sqlmock.NewRows([]string{"ID", "IsDark", "Timezone", "Cookie"}).
				AddRow(1, true, "America/New_York", ucookie)

			project_group_rows := sqlmock.NewRows([]string{"GroupID", "UserID", "GroupName"}).
				AddRow(1, 1, "First Group")

			project_rows := sqlmock.NewRows([]string{"ID", "UUID", "Name"}).
				AddRow(1, projectId, "First Project").
				AddRow(2, "59e06cf8-f390-5093-af2e-3685be593a25", "Second Project")

			preferred_projects := sqlmock.NewRows([]string{"ID", "UserID", "ProjectID", "GroupID"}).
				AddRow(1, 1, 1, 1).
				AddRow(2, 1, 2, 1)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "preferred_projects" WHERE user_id = $1`)).
				WithArgs(1).
				WillReturnRows(preferred_projects)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_groups" WHERE "project_groups"."group_id" = $1`)).
				WithArgs(1).
				WillReturnRows(project_group_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE "project_details"."id" IN ($1,$2)`)).
				WithArgs(1, 2).
				WillReturnRows(project_rows)

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/user/preference", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)
			handler.GetPreferredProject(c)

			var responseBody user.PreferenceResponse
			err = json.Unmarshal(w.Body.Bytes(), &responseBody)

			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(responseBody.Preferred).To(Not(Equal(0)))
			Expect(responseBody.Preferred[0].GroupName).To(Equal("First Group"))
			Expect(len(responseBody.Preferred[0].Projects)).To(Equal(2))
			Expect(responseBody.Preferred[0].Projects[0].UUID).To(Equal("96ad860-2a9a-504f-8861-aeafd0b2ae29"))
		})
		It("for empty preferred project details, will return empty object", func() {
			reqBody, err := json.Marshal("")
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal user preference request", err)
				return
			}

			user_rows := sqlmock.NewRows([]string{"ID", "IsDark", "Timezone", "Cookie"}).
				AddRow(1, true, "America/New_York", ucookie)

			project_rows := sqlmock.NewRows([]string{"ID", "UUID", "Name"}).
				AddRow(1, projectId, "First Project").
				AddRow(2, "59e06cf8-f390-5093-af2e-3685be593a25", "Second Project")

			preferred_projects := sqlmock.NewRows([]string{"ID", "UserID", "ProjectID", "GroupID"}) //empty rows

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "preferred_projects" WHERE user_id = $1`)).
				WithArgs(1).
				WillReturnRows(preferred_projects)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_details" WHERE "project_details"."id" IN ($1,$2)`)).
				WithArgs(1, 2).
				WillReturnRows(project_rows)

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/user/preference", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)
			handler.GetPreferredProject(c)

			var responseBody user.PreferenceResponse
			err = json.Unmarshal(w.Body.Bytes(), &responseBody)

			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(responseBody.Preferred).To(BeNil())
		})
	})

	Context("when delete preferred project is invoked", func() {

		delPrefRequest := user.DeletePreferredRequest{
			Preferred: []struct {
				GroupID uint64 `json:"group_id"`
			}{{
				GroupID: 1,
			}},
		}

		It("will delete preferred project", func() {
			reqBody, err := json.Marshal(delPrefRequest)
			if err != nil {
				utils.Log.Error("[TEST-ERROR]: Failed to Marshal delete preferred project request", err)
				return
			}

			user_rows := sqlmock.NewRows([]string{"ID", "IsDark", "Timezone", "Cookie"}).
				AddRow(1, true, "America/New_York", ucookie)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "app_users" WHERE cookie = $1 ORDER BY "app_users"."id" LIMIT $2`)).
				WithArgs(ucookie, 1).
				WillReturnRows(user_rows)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "preferred_projects" WHERE user_id = $1 AND group_id IN ($2)`)).
				WithArgs(1, 1).
				WillReturnResult(sqlmock.NewResult(0, 1))

			mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "project_groups" WHERE user_id = $1 AND group_id IN ($2)`)).
				WithArgs(1, 1).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/user/preference", bytes.NewBuffer([]byte(reqBody)))
			req.Header.Set("Content-Type", "application/json")

			// Set the cookie on the request
			req.AddCookie(&http.Cookie{
				Name:  utils.CookieName,
				Value: ucookie,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler := user.NewUserHandler(gormDb)
			handler.DeletePreferredProject(c)

			Expect(w.Code).To(Equal(200))
			Expect(w.Body.String()).To(ContainSubstring("{\"status\":\"deleted\"}"))
		})
	})
})
