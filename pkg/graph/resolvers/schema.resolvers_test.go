package resolvers_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	"time"
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
			// Mock the test_runs query
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", "1")

			mock.ExpectQuery(`SELECT test_runs.*, project_details.uuid, project_details.name AS test_project_name, project_details.team_name FROM "test_runs" JOIN project_details ON project_details.id = test_runs.project_id ORDER BY id ASC LIMIT \$1`).
				WithArgs(1).
				WillReturnRows(rows)

			// Mock the suite_runs query
			suiteRows := sqlmock.NewRows([]string{"ID", "TestRunID", "SuiteName"}).
				AddRow(1, 1, "suite 1")

			mock.ExpectQuery(`SELECT \* FROM "suite_runs" WHERE "suite_runs"."test_run_id" = \$1`).
				WithArgs(1).
				WillReturnRows(suiteRows)

			// Mock the spec_runs query
			specRunRows := sqlmock.NewRows([]string{"ID", "SuiteID", "SpecDescription", "Status", "Message", "StartTime", "EndTime"}).
				AddRow(1, 1, "Test Spec", "passed", "No errors", time.Now(), time.Now())

			mock.ExpectQuery(`SELECT \* FROM "spec_runs" WHERE "spec_runs"."suite_id" = \$1`).
				WithArgs(1).
				WillReturnRows(specRunRows)

			// Mock the spec_run_tags query
			specRunTagsRows := sqlmock.NewRows([]string{"ID", "SpecRunID", "TagID"}).
				AddRow(1, 1, 1).
				AddRow(2, 1, 2).
				AddRow(3, 1, 3)

			mock.ExpectQuery(`SELECT \* FROM "spec_run_tags" WHERE "spec_run_tags"."spec_run_id" = \$1`).
				WithArgs(1).
				WillReturnRows(specRunTagsRows)

			// Mock the tags query
			tagsRows := sqlmock.NewRows([]string{"ID", "Name"}).
				AddRow(1, "tag1").
				AddRow(2, "tag2").
				AddRow(3, "tag3")

			mock.ExpectQuery(`SELECT \* FROM "tags" WHERE "tags"."id" IN \(\$1,\$2,\$3\)`).
				WithArgs(1, 2, 3).
				WillReturnRows(tagsRows)

			// Mock the count query for test_runs
			countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(countRows)

			// Setup mock DB and GraphQL resolver
			queryResolver := &resolvers.Resolver{DB: gormDb}

			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
			gqlHandler.AddTransport(transport.POST{})

			srv := httptest.NewServer(gqlHandler)

			defer srv.Close()

			cli := client.New(gqlHandler)

			query := `
			query getTestRuns {
				  testRuns(first: 1, after:"" ) {
					edges {
					  cursor
					  testRun {
						id
						testProjectName
						testSeed
						startTime
						endTime
						suiteRuns{
						  id
						  suiteName
						  specRuns{
							id
							specDescription
							status
							message
							tags{
							  id
							  name
							}
						  }
						}
					   
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

			// Check total count and the first test run data
			Expect(gql_response.TestRuns.TotalCount).To(Equal(1))
			Expect(gql_response.TestRuns.Edges[0].TestRun.ID).To(Equal(1))
			Expect(gql_response.TestRuns.Edges[0].TestRun.TestProjectName).To(Equal("project 1"))
			Expect(gql_response.TestRuns.Edges[0].TestRun.TestSeed).To(Equal((1)))

			fmt.Print(gql_response.TestRuns.Edges[0])
			//// Check spec runs and tags
			//Expect(gql_response.TestRuns.Edges[0].TestRun.SuiteRuns[0].SpecRuns[0].Tags).To(HaveLen(2))
			//Expect(gql_response.TestRuns.Edges[0].TestRun.SuiteRuns[0].SpecRuns[0].Tags[0].ID).To(Equal(1))
			//Expect(gql_response.TestRuns.Edges[0].TestRun.SuiteRuns[0].SpecRuns[0].Tags[0].Name).To(Equal("tag1"))
			//Expect(gql_response.TestRuns.Edges[0].TestRun.SuiteRuns[0].SpecRuns[0].Tags[1].ID).To(Equal(2))
			//Expect(gql_response.TestRuns.Edges[0].TestRun.SuiteRuns[0].SpecRuns[0].Tags[1].Name).To(Equal("tag2"))

		})

		It("should query db to fetch first 2 test run records when pagesize 2 is selected", func() {

			// Setup mock rows for two records in test_runs (query for actual test runs)
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", 1).
				AddRow(2, "project 2", 2)

			// Expectation for the test_runs query to retrieve the actual records
			mock.ExpectQuery(`SELECT test_runs.*, project_details.uuid, project_details.name AS test_project_name, project_details.team_name FROM "test_runs" JOIN project_details ON project_details.id = test_runs.project_id ORDER BY id ASC LIMIT \$1`).
				WithArgs(2).
				WillReturnRows(rows)

			// Mock suite_runs query for the two test runs
			suiteRows := sqlmock.NewRows([]string{"ID", "TestRunID", "SuiteName"}).
				AddRow(1, 1, "suite 1").
				AddRow(2, 2, "suite 2")
			mock.ExpectQuery(`SELECT \* FROM "suite_runs" WHERE "suite_runs"."test_run_id" IN \(\$1,\$2\)`).
				WithArgs(1, 2).
				WillReturnRows(suiteRows)

			// Mock spec_runs query for suite_id = 1
			specRunRows := sqlmock.NewRows([]string{"ID", "SuiteID", "SpecDescription", "Status", "Message", "StartTime", "EndTime"}).
				AddRow(1, 1, "Test Spec", "passed", "No errors", time.Now(), time.Now())

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "spec_runs" WHERE "spec_runs"."suite_id" IN ($1,$2)`)).
				WithArgs(1, 2).
				WillReturnRows(specRunRows)

			// Mock spec_run_tags query for spec_run_id = 1
			specRunTagsRows := sqlmock.NewRows([]string{"ID", "SpecRunID", "TagID"}).
				AddRow(1, 1, 1).
				AddRow(2, 1, 2).
				AddRow(3, 1, 3)

			mock.ExpectQuery(`SELECT \* FROM "spec_run_tags" WHERE "spec_run_tags"."spec_run_id" = \$1`).
				WithArgs(1).
				WillReturnRows(specRunTagsRows)

			// Mock the tags query
			tagsRows := sqlmock.NewRows([]string{"ID", "Name"}).
				AddRow(1, "tag1").
				AddRow(2, "tag2").
				AddRow(3, "tag3")

			mock.ExpectQuery(`SELECT \* FROM "tags" WHERE "tags"."id" IN \(\$1,\$2,\$3\)`).
				WithArgs(1, 2, 3).
				WillReturnRows(tagsRows)

			// Mock the count query for test_runs
			countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(countRows)

			// Initialize the resolver and GraphQL handler
			queryResolver := &resolvers.Resolver{DB: gormDb}
			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
			gqlHandler.AddTransport(transport.POST{})
			srv := httptest.NewServer(gqlHandler)
			defer srv.Close()

			cli := client.New(gqlHandler)

			// Define the GraphQL query
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

			err := cli.Post(query, &gql_response)
			Expect(err).NotTo(HaveOccurred())

			// Check if the response contains the correct number of records
			Expect(len(gql_response.TestRuns.Edges)).To(Equal(2))

			// Verify the first record
			Expect(gql_response.TestRuns.Edges[0].TestRun.ID).To(Equal(1))
			Expect(gql_response.TestRuns.Edges[0].TestRun.TestProjectName).To(Equal("project 1"))

			// Verify the second record
			Expect(gql_response.TestRuns.Edges[1].TestRun.ID).To(Equal(2))
			Expect(gql_response.TestRuns.Edges[1].TestRun.TestProjectName).To(Equal("project 2"))

			// Verify pagination info
			Expect(gql_response.TestRuns.TotalCount).To(Equal(2))
			Expect(gql_response.TestRuns.PageInfo.HasNextPage).To(BeFalse())
			Expect(gql_response.TestRuns.PageInfo.HasPreviousPage).To(BeFalse())
		})

		It("should return no test run records when pagesize 0 is selected", func() {
			// Setup mock rows for no records
			rows := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"})

			// Setup mock for the count query
			countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)

			// Expectation for the data query
			mock.ExpectQuery(`SELECT test_runs.*, project_details.uuid, project_details.name AS test_project_name, project_details.team_name FROM "test_runs" JOIN project_details ON project_details.id = test_runs.project_id ORDER BY id ASC LIMIT \$1`).
				WithArgs(0).
				WillReturnRows(rows)

			// Expectation for the count query
			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(countRows)

			queryResolver := &resolvers.Resolver{DB: gormDb}

			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
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

			mock.ExpectQuery(`SELECT test_runs.*, project_details.uuid, project_details.name AS test_project_name, project_details.team_name FROM "test_runs" JOIN project_details ON project_details.id = test_runs.project_id ORDER BY id ASC LIMIT \$1`).
				WithArgs(5).
				WillReturnRows(rows)

			suiteRows := sqlmock.NewRows([]string{"ID", "TestRunID", "SuiteName"}).
				AddRow(1, 1, "suite 1")

			specRows := sqlmock.NewRows([]string{"ID", "SuiteID", "SpecDescription"}).
				AddRow(1, 1, "spec 1")

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "suite_runs" WHERE "suite_runs"."test_run_id" IN ($1,$2,$3)`)).
				WithArgs(1, 2, 3).
				WillReturnRows(suiteRows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "spec_runs" WHERE "spec_runs"."suite_id" = $1`)).
				WithArgs(1).
				WillReturnRows(specRows)

			// Mock the spec_run_tags query
			specRunTagsRows := sqlmock.NewRows([]string{"ID", "SpecRunID", "TagID"}).
				AddRow(1, 1, 1).
				AddRow(2, 1, 2).
				AddRow(3, 1, 3)

			mock.ExpectQuery(`SELECT \* FROM "spec_run_tags" WHERE "spec_run_tags"."spec_run_id" = \$1`).
				WithArgs(1).
				WillReturnRows(specRunTagsRows)

			// Mock the tags query
			tagsRows := sqlmock.NewRows([]string{"ID", "Name"}).
				AddRow(1, "tag1").
				AddRow(2, "tag2").
				AddRow(3, "tag3")

			mock.ExpectQuery(`SELECT \* FROM "tags" WHERE "tags"."id" IN \(\$1,\$2,\$3\)`).
				WithArgs(1, 2, 3).
				WillReturnRows(tagsRows)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "test_runs"`)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

			queryResolver := &resolvers.Resolver{DB: gormDb}

			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
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
			desc := true

			// Expected test data
			totalCount := int64(5)

			testRuns := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(3, "project 3", 3).
				AddRow(4, "project 4", 4)

			// Mocking the expected SQL queries and results in the correct order
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT test_runs.*, project_details.uuid, project_details.name AS test_project_name, project_details.team_name FROM "test_runs" JOIN project_details ON project_details.id = test_runs.project_id ORDER BY id DESC LIMIT $1 OFFSET $2`)).
				WithArgs(first, 2). // first=2, after=2 means starting from 3rd record
				WillReturnRows(testRuns)

			suiteRows := sqlmock.NewRows([]string{"ID", "TestRunID", "SuiteName"}).
				AddRow(1, 3, "suite 1")
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "suite_runs" WHERE "suite_runs"."test_run_id" IN ($1,$2)`)).
				WithArgs(3, 4).
				WillReturnRows(suiteRows)

			specRows := sqlmock.NewRows([]string{"ID", "SuiteID", "SpecDescription"}).
				AddRow(1, 1, "spec 1")
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "spec_runs" WHERE "spec_runs"."suite_id" = $1`)).
				WithArgs(1).
				WillReturnRows(specRows)

			// Mock the spec_run_tags query
			specRunTagsRows := sqlmock.NewRows([]string{"ID", "SpecRunID", "TagID"}).
				AddRow(1, 1, 1).
				AddRow(2, 1, 2).
				AddRow(3, 1, 3)

			mock.ExpectQuery(`SELECT \* FROM "spec_run_tags" WHERE "spec_run_tags"."spec_run_id" = \$1`).
				WithArgs(1).
				WillReturnRows(specRunTagsRows)

			// Mock the tags query
			tagsRows := sqlmock.NewRows([]string{"ID", "Name"}).
				AddRow(1, "tag1").
				AddRow(2, "tag2").
				AddRow(3, "tag3")

			mock.ExpectQuery(`SELECT \* FROM "tags" WHERE "tags"."id" IN \(\$1,\$2,\$3\)`).
				WithArgs(1, 2, 3).
				WillReturnRows(tagsRows)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Execute the resolver function
			result, err := queryResolver.Query().TestRuns(ctx, &first, &after, &desc)
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
			desc := true

			// Expected test data
			totalCount := int64(3)

			testRuns := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(3, "project 3", 3)

			// Mocking the expected SQL queries and results in the correct order
			//mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" ORDER BY id DESC LIMIT $1 OFFSET $2`)).
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT test_runs.*, project_details.uuid, project_details.name AS test_project_name, project_details.team_name FROM "test_runs" JOIN project_details ON project_details.id = test_runs.project_id ORDER BY id DESC LIMIT $1 OFFSET $2`)).
				WithArgs(first, 2). // first=1, after=2 means starting from 3rd record
				WillReturnRows(testRuns)

			suiteRows := sqlmock.NewRows([]string{"ID", "TestRunID", "SuiteName"}).
				AddRow(1, 3, "suite 1")
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "suite_runs" WHERE "suite_runs"."test_run_id" = $1`)).
				WithArgs(3).
				WillReturnRows(suiteRows)

			specRows := sqlmock.NewRows([]string{"ID", "SuiteID", "SpecDescription"}).
				AddRow(1, 1, "spec 1")
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "spec_runs" WHERE "spec_runs"."suite_id" = $1`)).
				WithArgs(1).
				WillReturnRows(specRows)

			// Mock the spec_run_tags query
			specRunTagsRows := sqlmock.NewRows([]string{"ID", "SpecRunID", "TagID"}).
				AddRow(1, 1, 1).
				AddRow(2, 1, 2).
				AddRow(3, 1, 3)

			mock.ExpectQuery(`SELECT \* FROM "spec_run_tags" WHERE "spec_run_tags"."spec_run_id" = \$1`).
				WithArgs(1).
				WillReturnRows(specRunTagsRows)

			// Mock the tags query
			tagsRows := sqlmock.NewRows([]string{"ID", "Name"}).
				AddRow(1, "tag1").
				AddRow(2, "tag2").
				AddRow(3, "tag3")

			mock.ExpectQuery(`SELECT \* FROM "tags" WHERE "tags"."id" IN \(\$1,\$2,\$3\)`).
				WithArgs(1, 2, 3).
				WillReturnRows(tagsRows)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Execute the resolver function
			result, err := queryResolver.Query().TestRuns(ctx, &first, &after, &desc)
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
			desc := true
			ctx := context.Background()
			// Mock the query to fetch test runs with an error
			//mock.ExpectQuery(`SELECT \* FROM "test_runs" .*`).
			mock.ExpectQuery(`SELECT test_runs.*, project_details.uuid, project_details.name AS test_project_name, project_details.team_name FROM "test_runs" JOIN project_details ON project_details.id = test_runs.project_id ORDER BY id DESC`).
				WillReturnError(errors.New("database error when fetching test_runs"))

			// Act: Call the TestRuns method
			_, err := queryResolver.Query().TestRuns(ctx, &testFirst, &testAfter, &desc)

			// Assert: Verify that an error occurred and it contains the correct message
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database error"))
		})

		It("should return an error when fetching the total count of TestRun records fails", func() {
			// Arrange: Setup the resolver and mock context
			queryResolver := &resolvers.Resolver{DB: gormDb} // Assuming your resolver structure
			testFirst := 3
			testAfter := ""
			desc := true
			ctx := context.Background()

			mock.ExpectQuery(`SELECT test_runs.*, project_details.uuid, project_details.name AS test_project_name, project_details.team_name FROM "test_runs" JOIN project_details ON project_details.id = test_runs.project_id ORDER BY id DESC`).
				WillReturnRows(sqlmock.NewRows([]string{"id", "test_project_name"}).
					AddRow(1, "Project A").
					AddRow(2, "Project B").
					AddRow(3, "Project C"))

			suiteRows := sqlmock.NewRows([]string{"ID", "TestRunID", "SuiteName"}).
				AddRow(1, 3, "suite 1")
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "suite_runs" WHERE "suite_runs"."test_run_id" IN ($1,$2,$3)`)).
				WithArgs(1, 2, 3).
				WillReturnRows(suiteRows)

			specRows := sqlmock.NewRows([]string{"ID", "SuiteID", "SpecDescription"}).
				AddRow(1, 1, "spec 1")
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "spec_runs" WHERE "spec_runs"."suite_id" = $1`)).
				WithArgs(1).
				WillReturnRows(specRows)

			// Mock the spec_run_tags query
			specRunTagsRows := sqlmock.NewRows([]string{"ID", "SpecRunID", "TagID"}).
				AddRow(1, 1, 1).
				AddRow(2, 1, 2).
				AddRow(3, 1, 3)

			mock.ExpectQuery(`SELECT \* FROM "spec_run_tags" WHERE "spec_run_tags"."spec_run_id" = \$1`).
				WithArgs(1).
				WillReturnRows(specRunTagsRows)

			// Mock the tags query
			tagsRows := sqlmock.NewRows([]string{"ID", "Name"}).
				AddRow(1, "tag1").
				AddRow(2, "tag2").
				AddRow(3, "tag3")

			mock.ExpectQuery(`SELECT \* FROM "tags" WHERE "tags"."id" IN \(\$1,\$2,\$3\)`).
				WithArgs(1, 2, 3).
				WillReturnRows(tagsRows) // Mock the query to fetch total count with an error
			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnError(errors.New("database error when fetching total count"))

			// Act: Call the TestRuns method
			_, err := queryResolver.Query().TestRuns(ctx, &testFirst, &testAfter, &desc)

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

			// Correct response struct
			var response struct {
				TestRun []struct {
					ID              int    `json:"id"`
					TestProjectName string `json:"testProjectName"`
					TestSeed        int    `json:"testSeed"`
					StartTime       string `json:"startTime"`
					EndTime         string `json:"endTime"`
					SuiteRuns       []struct {
						ID        int    `json:"id"`
						TestRunID int    `json:"testRunId"`
						SuiteName string `json:"suiteName"`
						StartTime string `json:"startTime"`
						EndTime   string `json:"endTime"`
						SpecRuns  []struct {
							ID              int    `json:"id"`
							SuiteID         int    `json:"suiteId"`
							SpecDescription string `json:"specDescription"`
							Status          string `json:"status"`
							Message         string `json:"message"`
							StartTime       string `json:"startTime"`
							EndTime         string `json:"endTime"`
							Tags            []struct {
								ID   int    `json:"id"`
								Name string `json:"name"`
							} `json:"tags"`
						} `json:"specRuns"`
					} `json:"suiteRuns"`
				} `json:"testRun"`
			}

			// Make the actual GraphQL request
			err := cli.Post(query, &response)
			Expect(err).NotTo(HaveOccurred())

			// Check if all mock expectations were met
			err = mock.ExpectationsWereMet()
			Expect(err).NotTo(HaveOccurred())

			// Print the response for debugging
			fmt.Println(response)

			// Assertions based on the response format
			//Expect(response.Data.TestRun[0].ID).To(Equal(1))
			//Expect(response.Data.TestRun[0].TestProjectName).To(Equal("project 1"))
			//Expect(response.Data.TestRun[0].TestSeed).To(Equal(1))
			//Expect(len(response.Data.TestRun[0].SuiteRuns)).To(Equal(1))
			//Expect(response.Data.TestRun[0].SuiteRuns[0].ID).To(Equal(1))
			//Expect(response.Data.TestRun[0].SuiteRuns[0].TestRunID).To(Equal(1))
			//Expect(response.Data.TestRun[0].SuiteRuns[0].SuiteName).To(Equal("suite 1"))
		})

	})

	Context("test TestRunByID resolver", func() {
		It("should query db to fetch one test run record by ID", func() {
			// Expected test data
			totalCount := 1

			testRuns := sqlmock.NewRows([]string{"ID", "TestProjectName", "TestSeed"}).
				AddRow(1, "project 1", 1).AddRow(2, "project 2", 2)

			// Mocking the expected SQL queries and results in the correct order
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "test_runs" WHERE id = $1 ORDER BY "test_runs"."id" LIMIT $2`)).
				WithArgs(1, 1).
				WillReturnRows(testRuns)

			suiteRows := sqlmock.NewRows([]string{"ID", "TestRunID", "SuiteName"}).
				AddRow(1, 1, "suite 1")

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "suite_runs" WHERE "suite_runs"."test_run_id" = $1`)).
				WithArgs(1).
				WillReturnRows(suiteRows)

			specRows := sqlmock.NewRows([]string{"ID", "SuiteID", "SpecDescription"}).
				AddRow(1, 1, "spec 1")
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "spec_runs" WHERE "spec_runs"."suite_id" = $1`)).
				WithArgs(1).
				WillReturnRows(specRows)

			// Mock the spec_run_tags query
			specRunTagsRows := sqlmock.NewRows([]string{"ID", "SpecRunID", "TagID"}).
				AddRow(1, 1, 1).
				AddRow(2, 1, 2).
				AddRow(3, 1, 3)

			mock.ExpectQuery(`SELECT \* FROM "spec_run_tags" WHERE "spec_run_tags"."spec_run_id" = \$1`).
				WithArgs(1).
				WillReturnRows(specRunTagsRows)

			// Mock the tags query
			tagsRows := sqlmock.NewRows([]string{"ID", "Name"}).
				AddRow(1, "tag1").
				AddRow(2, "tag2").
				AddRow(3, "tag3")

			mock.ExpectQuery(`SELECT \* FROM "tags" WHERE "tags"."id" IN \(\$1,\$2,\$3\)`).
				WithArgs(1, 2, 3).
				WillReturnRows(tagsRows)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "test_runs"`).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Create a new instance of the resolver with the mock database
			queryResolver := &resolvers.Resolver{DB: gormDb}

			// Create a new GraphQL handler with the resolver
			gqlHandler := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: queryResolver}))
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
                        startTime
                        endTime
                        suiteRuns {
                          id
                          suiteName
                          specRuns {
                            id
                            suiteId
                            specDescription
                            message
                            tags {
                              id
                              name
                            }
                          }
                        }
                    }
                }
        `

			// Define the responseonse struct to unmarshal the GraphQL responseonse
			var response struct {
				TestRunByID struct {
					ID              int    `json:"id"`
					TestProjectName string `json:"testProjectName"`
					TestSeed        int    `json:"testSeed"`
					StartTime       string `json:"startTime"`
					EndTime         string `json:"endTime"`
					SuiteRuns       []struct {
						ID        int    `json:"id"`
						SuiteName string `json:"suiteName"`
						SpecRuns  []struct {
							ID              int    `json:"id"`
							SuiteID         int    `json:"suiteId"`
							SpecDescription string `json:"specDescription"`
							Status          string `json:"status"`
							Message         string `json:"message"`
							Tags            []struct {
								ID   int    `json:"id"`
								Name string `json:"name"`
							} `json:"tags"`
						} `json:"specRuns"`
					} `json:"suiteRuns"`
				}
			}

			// Execute the GraphQL query and unmarshal the responseonse into the gql_response struct
			err := cli.Post(query, &response)

			Expect(err).NotTo(HaveOccurred())

			//fmt.Println(response)

			// Verify the responseonse fields match the expected values
			Expect(response.TestRunByID.ID).To(Equal(1))
			Expect(response.TestRunByID.TestProjectName).To(Equal("project 1"))
			Expect(response.TestRunByID.TestSeed).To(Equal(1))

			Expect(response.TestRunByID.SuiteRuns[0].ID).To(Equal(1))
			Expect(response.TestRunByID.SuiteRuns[0].SuiteName).To(Equal("suite 1"))
			Expect(response.TestRunByID.SuiteRuns[0].SpecRuns[0].ID).To(Equal(1))
			Expect(response.TestRunByID.SuiteRuns[0].SpecRuns[0].SuiteID).To(Equal(1))
			Expect(response.TestRunByID.SuiteRuns[0].SpecRuns[0].SpecDescription).To(Equal("spec 1"))
			Expect(response.TestRunByID.SuiteRuns[0].SpecRuns[0].Tags[0].ID).To(Equal(1))
			Expect(response.TestRunByID.SuiteRuns[0].SpecRuns[0].Tags[0].Name).To(Equal("tag1"))

			Expect(response.TestRunByID.SuiteRuns[0].SpecRuns[0].Tags[1].ID).To(Equal(2))
			Expect(response.TestRunByID.SuiteRuns[0].SpecRuns[0].Tags[1].Name).To(Equal("tag2"))

			Expect(response.TestRunByID.SuiteRuns[0].SpecRuns[0].Tags[2].ID).To(Equal(3))
			Expect(response.TestRunByID.SuiteRuns[0].SpecRuns[0].Tags[2].Name).To(Equal("tag3"))

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
				StartTime       string `json:"startTime"` // Assuming string for time, adjust if needed
				EndTime         string `json:"endTime"`   // Assuming string for time, adjust if needed
				SuiteRuns       []struct {
					ID        int    `json:"id"`
					SuiteName string `json:"suiteName"`
					SpecRuns  []struct {
						ID              int    `json:"id"`
						SpecDescription string `json:"specDescription"`
						Status          string `json:"status"`
						Message         string `json:"message"`
						Tags            []struct {
							ID   int    `json:"id"`
							Name string `json:"name"`
						} `json:"tags"`
					} `json:"specRuns"`
				} `json:"suiteRuns"`
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
