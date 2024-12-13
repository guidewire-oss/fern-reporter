package resolvers_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/guidewire/fern-reporter/pkg/graph/generated"
	"github.com/guidewire/fern-reporter/pkg/graph/resolvers"
	"github.com/guidewire/fern-reporter/pkg/utils"
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
		It("should query db to fetch first test run records when pagesize 1 is selected", func() {
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", "1")

			mock.ExpectQuery(`SELECT \* FROM "test_runs" LIMIT \$1`).
				WithArgs(1).
				WillReturnRows(rows)

			// Setup mock for the count query
			countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
			// Expectation for the count query
			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(countRows)

			queryResolver := &resolvers.Resolver{DB: gormDb}

			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
			// Add transports for POST requests
			gqlHandler.AddTransport(transport.POST{})

			srv := httptest.NewServer(gqlHandler)
			defer srv.Close()

			cli := client.New(gqlHandler)

			query := `
			query GetTestRuns {
				  testRuns(first: 1, after: "") {
					edges {
					  cursor
					  testRun {
						id
						testProjectName
						testSeed
					  }
					}
					pageInfo {
					  hasNextPage
					  hasPreviousPage
					  startCursor
					  endCursor
					}
					totalCount
				  }
				}
		`

			err := cli.Post(query, &gql_response)
			Expect(err).NotTo(HaveOccurred())

			// Check if at least one edge is returned before accessing it
			if len(gql_response.TestRuns.Edges) == 0 {
				Fail("No test runs returned")
			}

			Expect(gql_response.TestRuns.TotalCount).To(Equal(1))
			Expect(gql_response.TestRuns.Edges[0].TestRun.ID).To(Equal(1))
			Expect(gql_response.TestRuns.Edges[0].TestRun.TestProjectName).To(Equal("project 1"))
			Expect(gql_response.TestRuns.Edges[0].TestRun.TestSeed).To(Equal(1))
			Expect(gql_response.TestRuns.TotalCount).To(Equal(1))

		})

		It("should query db to fetch first 2 test run records when pagesize 2 is selected", func() {
			// Setup mock rows for two records
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", 1).
				AddRow(2, "project 2", 2)

			// Setup mock for the count query
			countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)

			// Expectation for the data query
			mock.ExpectQuery(`SELECT \* FROM "test_runs" LIMIT \$1`).
				WithArgs(2).
				WillReturnRows(rows)

			// Expectation for the count query
			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(countRows)

			queryResolver := &resolvers.Resolver{DB: gormDb}

			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))

			// Add transports for POST requests
			gqlHandler.AddTransport(transport.POST{})

			srv := httptest.NewServer(gqlHandler)
			defer srv.Close()

			cli := client.New(gqlHandler)

			query := `
        query GetTestRuns {
              testRuns(first: 2, after: "") {
                edges {
                  cursor
                  testRun {
                    id
                    testProjectName
                  }
                }
                pageInfo {
                  hasNextPage
                  hasPreviousPage
                  startCursor
                  endCursor
                }
                totalCount
              }
            }
    `

			var response struct {
				TestRuns struct {
					Edges []struct {
						Cursor  string `json:"cursor"`
						TestRun struct {
							ID              int    `json:"id"`
							TestProjectName string `json:"testProjectName"`
						} `json:"testRun"`
					} `json:"edges"`
					PageInfo struct {
						HasNextPage     bool   `json:"hasNextPage"`
						HasPreviousPage bool   `json:"hasPreviousPage"`
						StartCursor     string `json:"startCursor"`
						EndCursor       string `json:"endCursor"`
					} `json:"pageInfo"`
					TotalCount int `json:"totalCount"`
				} `json:"testRuns"`
			}

			err := cli.Post(query, &response)
			Expect(err).NotTo(HaveOccurred())

			// Check if the responseonse contains the correct number of records
			Expect(len(response.TestRuns.Edges)).To(Equal(2))

			// Verify the first record
			Expect(response.TestRuns.Edges[0].TestRun.ID).To(Equal(1))
			Expect(response.TestRuns.Edges[0].TestRun.TestProjectName).To(Equal("project 1"))

			// Verify the second record
			Expect(response.TestRuns.Edges[1].TestRun.ID).To(Equal(2))
			Expect(response.TestRuns.Edges[1].TestRun.TestProjectName).To(Equal("project 2"))

			// Verify pagination info
			Expect(response.TestRuns.TotalCount).To(Equal(2))
			Expect(response.TestRuns.PageInfo.HasNextPage).To(BeFalse())
			Expect(response.TestRuns.PageInfo.HasPreviousPage).To(BeFalse())
		})

		It("should return no test run records when pagesize 0 is selected", func() {
			// Setup mock rows for no records
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"})

			// Setup mock for the count query
			countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)

			// Expectation for the data query
			mock.ExpectQuery(`SELECT \* FROM "test_runs" LIMIT \$1`).
				WithArgs(0).
				WillReturnRows(rows)

			// Expectation for the count query
			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(countRows)

			queryResolver := &resolvers.Resolver{DB: gormDb}

			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
			// Add transports for POST requests
			gqlHandler.AddTransport(transport.POST{})

			srv := httptest.NewServer(gqlHandler)
			defer srv.Close()

			cli := client.New(gqlHandler)

			query := `
        query GetTestRuns {
              testRuns(first: 0, after: "") { 
                edges {
                  cursor
                  testRun {
                    id
                    testProjectName
                  }
                }
                pageInfo {
                  hasNextPage
                  hasPreviousPage
                  startCursor
                  endCursor
                }
                totalCount
              }
            }
    `

			err := cli.Post(query, &gql_response)
			Expect(err).NotTo(HaveOccurred())

			// Verify that there are no records
			Expect(len(gql_response.TestRuns.Edges)).To(Equal(0))

			// Verify pagination info
			Expect(gql_response.TestRuns.TotalCount).To(Equal(0))
			Expect(gql_response.TestRuns.PageInfo.HasNextPage).To(BeFalse())
			Expect(gql_response.TestRuns.PageInfo.HasPreviousPage).To(BeFalse())
			Expect(gql_response.TestRuns.PageInfo.StartCursor).To(BeEmpty())
			Expect(gql_response.TestRuns.PageInfo.EndCursor).To(BeEmpty())
		})

		It("should return all test run records when pagesize is larger than available records", func() {
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", 1).
				AddRow(2, "project 2", 2).
				AddRow(3, "project 3", 3)

			countRows := sqlmock.NewRows([]string{"count"}).AddRow(3)

			mock.ExpectQuery(`SELECT \* FROM "test_runs" LIMIT \$1`).
				WithArgs(5).
				WillReturnRows(rows)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(countRows)

			queryResolver := &resolvers.Resolver{DB: gormDb}

			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
			// Add transports for POST requests
			gqlHandler.AddTransport(transport.POST{})

			srv := httptest.NewServer(gqlHandler)
			defer srv.Close()

			cli := client.New(gqlHandler)

			query := `
        query GetTestRuns {
              testRuns(first: 5, after: "") {
                edges {
                  cursor
                  testRun {
                    id
                    testProjectName
                  }
                }
                pageInfo {
                  hasNextPage
                  hasPreviousPage
                  startCursor
                  endCursor
                }
                totalCount
              }
            }
    `
			err := cli.Post(query, &gql_response)
			Expect(err).NotTo(HaveOccurred())

			// Verify all records are returned
			Expect(len(gql_response.TestRuns.Edges)).To(Equal(3))

			// Verify pagination info
			Expect(gql_response.TestRuns.TotalCount).To(Equal(3))
			Expect(gql_response.TestRuns.PageInfo.HasNextPage).To(BeFalse())
			Expect(gql_response.TestRuns.PageInfo.HasPreviousPage).To(BeFalse())
			Expect(gql_response.TestRuns.PageInfo.StartCursor).To(Equal(gql_response.TestRuns.Edges[0].Cursor))
			Expect(gql_response.TestRuns.PageInfo.EndCursor).To(Equal(gql_response.TestRuns.Edges[2].Cursor))
		})

		It("should return correct records when good values of first and after are provided - total 5 records", func() {
			queryResolver := &resolvers.Resolver{DB: gormDb}
			ctx := context.Background()
			first := 2
			after := utils.EncodeCursor(2) // Assuming the offset is 2

			// Expected test data
			totalCount := int64(5)

			testRuns := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(3, "project 3", 3).
				AddRow(4, "project 4", 4)

			// Mocking the expected SQL queries and results in the correct order
			mock.ExpectQuery(`SELECT \* FROM "test_runs" LIMIT \$1 OFFSET \$2`).
				WithArgs(first, 2). // first=2, after=2 means starting from 3rd record
				WillReturnRows(testRuns)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Execute the resolver function
			result, err := queryResolver.Query().TestRuns(ctx, &first, &after)
			Expect(err).NotTo(HaveOccurred())

			// Validate the results
			Expect(result.Edges).To(HaveLen(2))
			Expect(result.Edges[0].Cursor).To(Equal(utils.EncodeCursor(3)))
			Expect(result.Edges[1].Cursor).To(Equal(utils.EncodeCursor(4)))
			Expect(result.PageInfo.HasNextPage).To(BeTrue())
			Expect(result.PageInfo.HasPreviousPage).To(BeTrue())
			Expect(result.TotalCount).To(Equal(int(totalCount)))

			// Ensure all expectations were met
			err = mock.ExpectationsWereMet()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return correct records when good values of first and after are provided - total 3 records", func() {
			queryResolver := &resolvers.Resolver{DB: gormDb}
			ctx := context.Background()
			first := 1
			after := utils.EncodeCursor(2) // Assuming the offset is 2

			// Expected test data
			totalCount := int64(3)

			testRuns := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(3, "project 3", 3)

			// Mocking the expected SQL queries and results in the correct order
			mock.ExpectQuery(`SELECT \* FROM "test_runs" LIMIT \$1 OFFSET \$2`).
				WithArgs(first, 2). // first=1, after=2 means starting from 3rd record
				WillReturnRows(testRuns)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Execute the resolver function
			result, err := queryResolver.Query().TestRuns(ctx, &first, &after)
			Expect(err).NotTo(HaveOccurred())

			// Validate the results
			Expect(result.Edges).To(HaveLen(1))
			Expect(result.Edges[0].Cursor).To(Equal(utils.EncodeCursor(3)))
			Expect(result.PageInfo.HasNextPage).To(BeFalse())
			Expect(result.PageInfo.HasPreviousPage).To(BeTrue())
			Expect(result.TotalCount).To(Equal(int(totalCount)))

			// Ensure all expectations were met
			err = mock.ExpectationsWereMet()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error when fetching TestRun records fails", func() {
			queryResolver := &resolvers.Resolver{DB: gormDb}
			testFirst := 3
			testAfter := ""
			ctx := context.Background()
			// Mock the query to fetch test runs with an error
			mock.ExpectQuery(`SELECT \* FROM "test_runs" .*`).
				WillReturnError(errors.New("database error when fetching test_runs"))

			// Act: Call the TestRuns method
			_, err := queryResolver.Query().TestRuns(ctx, &testFirst, &testAfter)

			// Assert: Verify that an error occurred and it contains the correct message
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database error"))
		})

		It("should return an error when fetching the total count of TestRun records fails", func() {
			// Arrange: Setup the resolver and mock context
			queryResolver := &resolvers.Resolver{DB: gormDb} // Assuming your resolver structure
			testFirst := 3
			testAfter := ""
			ctx := context.Background()

			mock.ExpectQuery(`SELECT \* FROM "test_runs" .*`).
				WillReturnRows(sqlmock.NewRows([]string{"id", "test_project_name"}).
					AddRow(1, "Project A").
					AddRow(2, "Project B").
					AddRow(3, "Project C"))

			// Mock the query to fetch total count with an error
			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnError(errors.New("database error when fetching total count"))

			// Act: Call the TestRuns method
			_, err := queryResolver.Query().TestRuns(ctx, &testFirst, &testAfter)

			// Assert: Verify that an error occurred and it contains the correct message
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database error when fetching total count"))
		})

	})

	Context("test testRun resolver", func() {
		It("should query db to fetch test run record", func() {
			testRunRows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", 1)

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

			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
			// Add transports for POST requests
			gqlHandler.AddTransport(transport.POST{})

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
			// responseonse struct
			var response struct {
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

			err := cli.Post(query, &response)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.TestRun[0].ID).To(Equal(1))
			Expect(response.TestRun[0].TestProjectName).To(Equal("project 1"))
			Expect(response.TestRun[0].TestSeed).To(Equal(1))
			Expect(len(response.TestRun[0].SuiteRuns)).To(Equal(1))
			Expect(response.TestRun[0].SuiteRuns[0].ID).To(Equal(1))
			Expect(response.TestRun[0].SuiteRuns[0].TestRunID).To(Equal(1))
			Expect(response.TestRun[0].SuiteRuns[0].SuiteName).To(Equal("suite 1"))
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
			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
			// Add transports for POST requests
			gqlHandler.AddTransport(transport.POST{})

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

			// Define the responseonse struct to unmarshal the GraphQL responseonse
			var response struct {
				TestRunByID struct {
					ID              int
					TestProjectName string
					TestSeed        int
				}
			}

			// Execute the GraphQL query and unmarshal the responseonse into the gql_response struct
			err := cli.Post(query, &response)

			Expect(err).NotTo(HaveOccurred())

			// Verify the responseonse fields match the expected values
			Expect(response.TestRunByID.ID).To(Equal(1))
			Expect(response.TestRunByID.TestProjectName).To(Equal("project 1"))
			Expect(response.TestRunByID.TestSeed).To(Equal(1))

			Expect(response.TestRunByID.ID).ToNot(Equal(2))
			Expect(response.TestRunByID.TestProjectName).ToNot(Equal("project 2"))
			Expect(response.TestRunByID.TestSeed).ToNot(Equal(2))
		})
	})

})

var gql_response struct {
	TestRuns struct {
		Edges []struct {
			Cursor  string `json:"cursor"`
			TestRun struct {
				ID              int    `json:"id"`
				TestProjectName string `json:"testProjectName"`
				TestSeed        int    `json:"testSeed"`
			} `json:"testRun"`
		} `json:"edges"`
		PageInfo struct {
			HasNextPage     bool   `json:"hasNextPage"`
			HasPreviousPage bool   `json:"hasPreviousPage"`
			StartCursor     string `json:"startCursor"`
			EndCursor       string `json:"endCursor"`
		} `json:"pageInfo"`
		TotalCount int `json:"totalCount"`
	} `json:"testRuns"`
}
