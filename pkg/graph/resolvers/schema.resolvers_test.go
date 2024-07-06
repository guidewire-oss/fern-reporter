package resolvers_test

import (
	"database/sql"
	"fmt"
	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/guidewire/fern-reporter/pkg/graph/generated"
	"github.com/guidewire/fern-reporter/pkg/graph/resolvers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http/httptest"
	"regexp"
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
	err := db.Close()
	if err != nil {
		return
	}
})

var _ = Describe("Handlers", func() {
	Context("test testRuns resolver", func() {
		It("should query db to fetch all test run records", func() {
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", "1")

			mock.ExpectQuery("SELECT (.+) FROM \"test_runs\"").
				WithoutArgs().
				WillReturnRows(rows)

			queryResolver := &resolvers.Resolver{DB: gormDb}

			gqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
			srv := httptest.NewServer(gqlHandler)
			defer srv.Close()

			cli := client.New(gqlHandler)

			//t.Run("Test TestRuns Resolver", func(t *testing.T) {
			query := `
			query {
				testRuns {
					id
					testProjectName
				}
			}
		`

			var resp struct {
				TestRuns []struct {
					ID              int
					TestProjectName string
					TestSeed        int
				}
			}

			err := cli.Post(query, &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(resp.TestRuns)).To(Equal(1))
			Expect(resp.TestRuns[0].ID).To(Equal(1))
			Expect(resp.TestRuns[0].TestProjectName).To(Equal("project 1"))
			Expect(resp.TestRuns[0].TestSeed).To(BeZero())
		})
	})

	Context("test testRun resolver", func() {
		It("should query db to fetch test run record", func() {
			testRunRows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", "1")

			suiteRows := sqlmock.NewRows([]string{"ID", "TestRunID", "SuiteName"}).
				AddRow(1, 1, "suite 1")

			specRows := sqlmock.NewRows([]string{"ID", "SuiteID", "SpecDescription"}).
				AddRow(1, 1, "spec 1")

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 AND test_project_name = $2`)).
				WithArgs(1, "project 1").
				WillReturnRows(testRunRows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "suite_runs" WHERE "suite_runs"."test_run_id" = $1`)).
				WithArgs(1).
				WillReturnRows(suiteRows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "spec_runs" WHERE "spec_runs"."suite_id" = $1`)).
				WithArgs(1).
				WillReturnRows(specRows)

			queryResolver := &resolvers.Resolver{DB: gormDb}

			gqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
			srv := httptest.NewServer(gqlHandler)
			defer srv.Close()

			cli := client.New(gqlHandler)

			query := `
			 query {
			  testRun(testRunFilter: { id: 1, testProjectName:"project 1" }) {
				id
				testProjectName
				testSeed
				suiteRuns {
					id
					testRunId
					suiteName
				}
			  }
			}`
			// Response struct
			var resp struct {
				TestRun []struct {
					ID              int
					TestProjectName string
					TestSeed        int
					SuiteRuns       []struct {
						ID        int
						TestRunID int
						SuiteName string
						StartTime string
						EndTime   string
					}
				}
			}

			err := cli.Post(query, &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.TestRun[0].ID).To(Equal(1))
			Expect(resp.TestRun[0].TestProjectName).To(Equal("project 1"))
			Expect(resp.TestRun[0].TestSeed).To(Equal(1))
			Expect(len(resp.TestRun[0].SuiteRuns)).To(Equal(1))
			Expect(resp.TestRun[0].SuiteRuns[0].ID).To(Equal(1))
			Expect(resp.TestRun[0].SuiteRuns[0].TestRunID).To(Equal(1))
			Expect(resp.TestRun[0].SuiteRuns[0].SuiteName).To(Equal("suite 1"))
		})
	})

	Context("test TestRunByID resolver", func() {
		It("should query db to fetch one test run record by ID", func() {
			// Define the expected rows to be returned by the mock database
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", "1").
				AddRow(2, "project 2", "2")

			// Expect a query to be executed against the "test_runs" table with the specified ID
			mock.ExpectQuery("SELECT (.+) FROM \"test_runs\" WHERE id = \\$1  ORDER BY \"test_runs\".\"id\" LIMIT \\$2").
				WithArgs(1, 1).
				WillReturnRows(rows)

			// Create a new instance of the resolver with the mock database
			queryResolver := &resolvers.Resolver{DB: gormDb}

			// Create a new GraphQL handler with the resolver
			gqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))

			// Create a new HTTP test server with the GraphQL handler
			srv := httptest.NewServer(gqlHandler)
			defer srv.Close()

			// Create a new GraphQL client with the HTTP test server
			cli := client.New(gqlHandler)

			// Define the GraphQL query to fetch a test run by ID
			query := `
            query {
                testRunById(id: 1) {
                    id
                    testProjectName
                    testSeed
                }
            }
        `

			// Define the response struct to unmarshal the GraphQL response
			var resp struct {
				TestRunByID struct {
					ID              int
					TestProjectName string
					TestSeed        int
				}
			}

			// Execute the GraphQL query and unmarshal the response into the resp struct
			err := cli.Post(query, &resp)

			fmt.Print(resp)
			Expect(err).NotTo(HaveOccurred())

			// Verify the response fields match the expected values
			Expect(resp.TestRunByID.ID).To(Equal(1))
			Expect(resp.TestRunByID.TestProjectName).To(Equal("project 1"))
			Expect(resp.TestRunByID.TestSeed).To(Equal(1))

			Expect(resp.TestRunByID.ID).ToNot(Equal(2))
			Expect(resp.TestRunByID.TestProjectName).ToNot(Equal("project 2"))
			Expect(resp.TestRunByID.TestSeed).ToNot(Equal(2))
		})
	})

})
