package routers_test

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/pkg/api/handlers"
	"github.com/guidewire/fern-reporter/pkg/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"time"
)

var (
	router      *gin.Engine
	db          *gorm.DB
	handler     *handlers.Handler
	testRecords []models.TestRun
)

var _ = BeforeSuite(func() {
	// Set up a test SQLite database
	db, _ = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})

	db.AutoMigrate(&models.TestRun{})

	// Insert test records
	testRecords = []models.TestRun{
		{ID: 1, TestProjectName: "Test 1", TestSeed: 0, StartTime: time.Time{}, EndTime: time.Time{}, SuiteRuns: nil},
		{ID: 2, TestProjectName: "Test 2", TestSeed: 0, StartTime: time.Time{}, EndTime: time.Time{}, SuiteRuns: nil},
		{ID: 3, TestProjectName: "Test 3", TestSeed: 0, StartTime: time.Time{}, EndTime: time.Time{}, SuiteRuns: nil},
	}

	for _, record := range testRecords {
		err := db.Create(&record).Error
		if err != nil {
			fmt.Errorf("Failed to insert test record: %v", err)
		}
	}

	// Create a new Gin router for each test
	gin.SetMode(gin.TestMode)
	router = gin.Default()

	// Create a new instance of the Handler type with the test GORM DB
	handler = handlers.NewHandler(db)
	if handler == nil {
		panic("Failed to initialize handler")
	}

	api := router.Group("/api")
	{
		testRun := api.Group("/testrun/")
		testRun.GET("/", handler.GetTestRunAll)
		testRun.GET("/:id", handler.GetTestRunByID)
		testRun.POST("/", handler.CreateTestRun)
		testRun.PUT("/:id", handler.UpdateTestRun)
		testRun.DELETE("/:id", handler.DeleteTestRun)
	}

	//router = routers.RegisterRouters(router)
})

var _ = AfterSuite(func() {
	// Close the Gorm database connection in the AfterSuite hook
	if db != nil {
		gormDb, _ := db.DB()
		defer gormDb.Close()
		for _, record := range testRecords {
			err := db.Delete(&record).Error
			if err != nil {
				fmt.Errorf("Failed to delete test record: %v", err)
			}
		}

		// Delete the database file after closing the connection
		if err := os.Remove("test.db"); err != nil {
			panic("Failed to delete the database file: " + err.Error())
		}
	}

})
var _ = Describe("RegisterRouters", func() {

	Context("/api/testrun routes", func() {
		It("should register GET /api/testrun", func() {
			req, err := http.NewRequest("GET", "/api/testrun/", nil)
			Expect(err).NotTo(HaveOccurred())

			resp := httptest.NewRecorder()
			gin.CreateTestContext(resp)

			if router == nil {
				panic("Router is nil")
			}

			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			expectedJson := "[{\"id\":1,\"test_project_name\":\"Test 1\",\"test_seed\":0,\"start_time\":\"0001-01-01T00:00:00Z\",\"end_time\":\"0001-01-01T00:00:00Z\",\"suite_runs\":null},{\"id\":2,\"test_project_name\":\"Test 2\",\"test_seed\":0,\"start_time\":\"0001-01-01T00:00:00Z\",\"end_time\":\"0001-01-01T00:00:00Z\",\"suite_runs\":null},{\"id\":3,\"test_project_name\":\"Test 3\",\"test_seed\":0,\"start_time\":\"0001-01-01T00:00:00Z\",\"end_time\":\"0001-01-01T00:00:00Z\",\"suite_runs\":null}]"
			Expect(resp.Body).To(MatchJSON(expectedJson))

		})
	})
	Context("/api/testrun/:id routes", func() {
		It("should register GET /api/testrun/:id", func() {
			req, err := http.NewRequest(http.MethodGet, "/api/testrun/1", nil)
			Expect(err).NotTo(HaveOccurred())

			resp := httptest.NewRecorder()
			gin.CreateTestContext(resp)
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))
			var responseBody map[string]interface{}
			err = json.Unmarshal(resp.Body.Bytes(), &responseBody)
			Expect(err).To(BeNil())

			Expect(uint64(responseBody["id"].(float64))).To(Equal(testRecords[0].ID))
			Expect(responseBody["test_project_name"]).To(Equal(testRecords[0].TestProjectName))
		})

		//It("should register POST /api/testrun", func() {
		//	// Similar to the above test, perform requests for POST /api/testrun route
		//	payload := map[string]interface{}{
		//		"TestProjectName": "Test 4",
		//		"TestSeed":        0,
		//		"StartTime":       time.Time{},
		//		"EndTime":         time.Time{},
		//		"SuiteRuns":       nil,
		//	}
		//
		//	// Convert the payload to JSON
		//	payloadJSON, err := json.Marshal(payload)
		//	Expect(err).To(BeNil())
		//
		//	req, err := http.NewRequest(http.MethodPost, "/api/testrun/", bytes.NewBuffer(payloadJSON))
		//	Expect(err).NotTo(HaveOccurred())
		//
		//	resp := httptest.NewRecorder()
		//	gin.CreateTestContext(resp)
		//	router.ServeHTTP(resp, req)
		//
		//	Expect(resp.Code).To(Equal(http.StatusCreated))
		//	var createdTestRun models.TestRun
		//	err = json.Unmarshal(resp.Body.Bytes(), &createdTestRun)
		//	Expect(err).To(BeNil())
		//
		//	var responseBody map[string]interface{}
		//	err = json.Unmarshal(resp.Body.Bytes(), &responseBody)
		//	Expect(err).To(BeNil())
		//
		//	//Expect(uint64(responseBody["id"].(float64))).To(Equal(testRecords[0].ID))
		//	getResp := httptest.NewRecorder()
		//	getReq, err := http.NewRequest("GET", "/api/testrun/"+strconv.FormatUint(responseBody["id"].(uint64), 10), nil)
		//	Expect(err).To(BeNil())
		//	gin.Default().ServeHTTP(getResp, getReq)
		//})

	})

	//Context("/reports/testruns routes", func() {
	//	It("should register GET /reports/testruns", func() {
	//		req, err := http.NewRequest("GET", "/reports/testruns", nil)
	//		Expect(err).NotTo(HaveOccurred())
	//
	//		resp := httptest.NewRecorder()
	//		router.ServeHTTP(resp, req)
	//
	//		Expect(resp.Code).To(Equal(http.StatusOK))
	//		// Add more expectations based on your application logic
	//	})
	//
	//	It("should register GET /reports/testruns/:id", func() {
	//		// Similar to the above test, perform requests for GET /reports/testruns/:id route
	//		req, err := http.NewRequest("GET", "/reports/testruns/123", nil)
	//		Expect(err).NotTo(HaveOccurred())
	//
	//		resp := httptest.NewRecorder()
	//		router.ServeHTTP(resp, req)
	//
	//		Expect(resp.Code).To(Equal(http.StatusOK))
	//		// Add more expectations based on your application logic
	//	})
	//
	//	// Add more tests for other /reports/testruns routes as needed
	//})
})
