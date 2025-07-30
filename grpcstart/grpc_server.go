package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/grpcfiles/createtestrun"
	"github.com/guidewire/fern-reporter/grpcfiles/deletetestrun"
	"github.com/guidewire/fern-reporter/grpcfiles/gettestrunall"
	gtid "github.com/guidewire/fern-reporter/grpcfiles/gettestrunbyid"
	"github.com/guidewire/fern-reporter/grpcfiles/processtags"
	pb "github.com/guidewire/fern-reporter/grpcfiles/reporter"
	"github.com/guidewire/fern-reporter/grpcfiles/reporttestrunall"
	"github.com/guidewire/fern-reporter/grpcfiles/reporttestrunbyid"
	"github.com/guidewire/fern-reporter/grpcfiles/updatetestrun"
	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type grpcServer struct {
	pb.UnimplementedReporterServer
}

type server struct {
	pb.UnimplementedPingServiceServer
}

type servertestbyid struct {
	gtid.UnimplementedTestRunServiceServer
	db *gorm.DB
}

type TestServiceServer struct {
	gettestrunall.UnimplementedTestServiceServer
	db *gorm.DB
}

// Define the TestRunServiceServer implementation
type TestRunServiceServerid struct {
	reporttestrunbyid.UnimplementedTestRunServiceServer
	db *gorm.DB
}

// Implement the TestRunServiceServer interface for reporttestrunall
type TestRunServiceServer struct {
	reporttestrunall.UnimplementedTestRunServiceServer
	db *gorm.DB
}

// Define the TestRunServiceServer implementation
type TestRunServiceServerDelete struct {
	deletetestrun.UnimplementedTestRunServiceServer
	db *gorm.DB
}

type Server struct {
	db *gorm.DB
	updatetestrun.UnimplementedTestRunServiceServer
}

type tagServiceServer struct {
	processtags.UnimplementedTagServiceServer
	db *gorm.DB
}

type testRunServiceServer struct {
	createtestrun.UnimplementedTestRunServiceServer
	db *gorm.DB
}

func (s *grpcServer) SendReport(ctx context.Context, req *pb.ReportRequest) (*pb.ReportResponse, error) {
	utils.GetLogger().Info(fmt.Sprintf("[gRPC-LOG]: Received gRPC report: %s", req.Message))
	return &pb.ReportResponse{Status: "Report received successfully"}, nil
}

// Correct method signature (use types from the generated pb package)
func (s *server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	utils.GetLogger().Info(fmt.Sprintf("[gRPC-LOG]: Received message: %s", req.GetMessage()))
	return &pb.PingResponse{Message: "Pong"}, nil
}

// Implement ReportTestRunById
func (s *TestRunServiceServerid) ReportTestRunById(ctx context.Context, req *reporttestrunbyid.ReportTestRunByIdRequest) (*reporttestrunbyid.ReportTestRunByIdResponse, error) {
	var testRun models.TestRun

	method, ok := grpc.Method(ctx)
	if !ok {
		utils.GetLogger().Error("[ERROR]: Method not found in context", nil)
		return nil, fmt.Errorf("method not found in context")
	}

	// Parse ID
	testRunID, err := strconv.Atoi(req.Id)
	if err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("[REQUEST-ERROR]: Invalid ID format at %s", method))
		return nil, fmt.Errorf("invalid ID format")
	}

	// Query database with preloading related fields
	s.db.Preload("SuiteRuns.SpecRuns").Where("id = ?", testRunID).First(&testRun)

	// Map database model to protobuf
	var pbSuiteRuns []*reporttestrunbyid.SuiteRun
	for _, sr := range testRun.SuiteRuns {
		var pbSpecRuns []*reporttestrunbyid.SpecRun
		for _, spec := range sr.SpecRuns {
			var pbTags []*reporttestrunbyid.Tag
			for _, tag := range spec.Tags {
				pbTags = append(pbTags, &reporttestrunbyid.Tag{Name: tag.Name})
			}
			pbSpecRuns = append(pbSpecRuns, &reporttestrunbyid.SpecRun{Tags: pbTags})
		}
		pbSuiteRuns = append(pbSuiteRuns, &reporttestrunbyid.SuiteRun{SpecRuns: pbSpecRuns})
	}

	// Return response
	return &reporttestrunbyid.ReportTestRunByIdResponse{
		ReportHeader: "Report Header", // Replace with actual header logic
		TestRun: &reporttestrunbyid.TestRun{
			Id:        strconv.Itoa(testRunID), // Convert ID back to string
			SuiteRuns: pbSuiteRuns,
		},
	}, nil
}

// reporttestrunall
func (s *TestRunServiceServer) ReportTestRunAll(ctx context.Context, req *reporttestrunall.ReportTestRunAllRequest) (*reporttestrunall.ReportTestRunAllResponse, error) {
	var testRuns []models.TestRun
	s.db.Preload("SuiteRuns.SpecRuns.Tags").Find(&testRuns)

	// Convert database model to protobuf response
	var pbTestRuns []*reporttestrunall.TestRun
	for _, tr := range testRuns {
		var pbSuiteRuns []*reporttestrunall.SuiteRun
		for _, sr := range tr.SuiteRuns {
			var pbSpecRuns []*reporttestrunall.SpecRun
			for _, spec := range sr.SpecRuns {
				var pbTags []*reporttestrunall.Tag
				for _, tag := range spec.Tags {
					pbTags = append(pbTags, &reporttestrunall.Tag{Name: tag.Name})
				}
				pbSpecRuns = append(pbSpecRuns, &reporttestrunall.SpecRun{Tags: pbTags})
			}
			pbSuiteRuns = append(pbSuiteRuns, &reporttestrunall.SuiteRun{SpecRuns: pbSpecRuns})
		}
		pbTestRuns = append(pbTestRuns, &reporttestrunall.TestRun{
			Id:        strconv.FormatUint(tr.ID, 10),
			SuiteRuns: pbSuiteRuns,
		})
	}

	return &reporttestrunall.ReportTestRunAllResponse{
		ReportHeader: config.GetHeaderName(),
		TestRuns:     pbTestRuns,
	}, nil
}

// Implement DeleteTestRun
func (s *TestRunServiceServer) DeleteTestRun(ctx context.Context, req *deletetestrun.DeleteTestRunRequest) (*deletetestrun.DeleteTestRunResponse, error) {
	var testRun models.TestRun

	method, ok := grpc.Method(ctx)
	if !ok {
		utils.GetLogger().Error("[ERROR]: Method not found in context", nil)
		return nil, fmt.Errorf("method not found in context")
	}

	// Parse ID
	testRunID, err := strconv.Atoi(req.Id)
	if err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("[REQUEST-ERROR]: Invalid ID format at %s", method))
		return &deletetestrun.DeleteTestRunResponse{
			Success: false,
			Message: "Invalid ID format",
		}, nil
	}

	testRun.ID = uint64(testRunID)

	// Delete operation
	result := s.db.Delete(&testRun)
	if result.Error != nil {
		// Database error
		utils.GetLogger().Error(fmt.Sprintf("[ERROR]: Failed to delete TestRun at %s", method), result.Error)
		return &deletetestrun.DeleteTestRunResponse{
			Success: false,
			Message: "Error deleting test run",
		}, nil
	} else if result.RowsAffected == 0 {
		// No rows affected (test run not found)
		utils.GetLogger().Warn(fmt.Sprintf("[REQUEST-ERROR]: Test run %d not found at %s", testRunID, method))
		return &deletetestrun.DeleteTestRunResponse{
			Success: false,
			Message: "Test run not found",
		}, nil
	}

	// Success response
	utils.GetLogger().Info(fmt.Sprintf("[gRPC-LOG]: Test run %d deleted successfully at %s", testRunID, method))
	return &deletetestrun.DeleteTestRunResponse{
		Success: true,
		Message: "Test run deleted successfully",
	}, nil
}

func (s *Server) UpdateTestRun(ctx context.Context, req *updatetestrun.UpdateTestRunRequest) (*updatetestrun.TestRunResponse, error) {
	var testRun models.TestRun

	method, ok := grpc.Method(ctx)
	if !ok {
		utils.GetLogger().Error("[ERROR]: Method not found in context", nil)
		return nil, fmt.Errorf("method not found in context")
	}

	// Find the TestRun by ID
	if err := s.db.Where("id = ?", req.GetId()).First(&testRun).Error; err != nil {

		utils.GetLogger().Warn(fmt.Sprintf("[REQUEST-ERROR]: TestRun %s not found at %s", req.GetId(), method))
		return &updatetestrun.TestRunResponse{
			Success: false,
			Message: "TestRun not found",
		}, fmt.Errorf("TestRun not found: %v", err)
	}

	// Update the fields of testRun based on the request
	testRun.TestProjectName = req.GetName() // Update the necessary fields

	// Save the updated TestRun in the database
	if err := s.db.Save(&testRun).Error; err != nil {
		utils.GetLogger().Error(fmt.Sprintf("[ERROR]: Failed to update TestRun %d at %s", testRun.ID, method), err)
		return &updatetestrun.TestRunResponse{
			Success: false,
			Message: "Failed to update TestRun",
		}, fmt.Errorf("failed to update TestRun: %v", err)
	}

	utils.GetLogger().Info(fmt.Sprintf("[gRPC-LOG]: TestRun %d updated successfully at %s", testRun.ID, method))
	// Return success response with updated TestRun
	return &updatetestrun.TestRunResponse{
		Success: true,
		Message: "TestRun updated successfully",
		TestRun: &updatetestrun.TestRun{
			Id:   strconv.FormatUint(testRun.ID, 10),
			Name: testRun.TestProjectName, // Include other fields as needed
		},
	}, nil
}

func (s *servertestbyid) GetTestRunByID(ctx context.Context, req *gtid.GetTestRunByIDRequest) (*gtid.GetTestRunByIDResponse, error) {
	var testRun models.TestRun
	id := req.GetId()
	result := s.db.Where("id = ?", id).First(&testRun)
	if result.Error != nil {
		return nil, result.Error
	}

	response := &gtid.GetTestRunByIDResponse{
		TestRun: &gtid.TestRun{
			Id: strconv.FormatUint(testRun.ID, 10),
			// Add other fields here
		},
	}
	return response, nil
}

func (s *TestServiceServer) GetTestRunAll(ctx context.Context, req *gettestrunall.EmptyRequest) (*gettestrunall.TestRunList, error) {
	var testRuns []models.TestRun
	if err := s.db.Find(&testRuns).Error; err != nil {
		return nil, err
	}

	// Convert testRuns to gRPC message format
	var grpcTestRuns []*gettestrunall.TestRun
	for _, t := range testRuns {
		grpcTestRuns = append(grpcTestRuns, &gettestrunall.TestRun{
			Id:     t.ID,
			Name:   t.TestProjectName,
			Status: t.StartTime.String(),
		})
	}

	return &gettestrunall.TestRunList{TestRuns: grpcTestRuns}, nil
}

func ProcessTags(db *gorm.DB, testRun *processtags.TestRun) (*processtags.ProcessTagsResponse, error) {
	// Process the tags as before
	for i, suite := range testRun.SuiteRuns {
		for j, spec := range suite.SpecRuns {
			var processedTags []*processtags.Tag // Use pointer slice

			for _, tag := range spec.Tags {
				var existingTag processtags.Tag

				// Check if the tag already exists
				result := db.Where("name = ?", tag.Name).First(&existingTag)

				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					// If the tag does not exist, create a new one
					newTag := &processtags.Tag{Name: tag.Name} // Use pointer directly
					if err := db.Create(newTag).Error; err != nil {
						return nil, err // Return error if tag creation fails
					}
					processedTags = append(processedTags, newTag)
				} else if result.Error != nil {
					// Return error if there is a problem fetching the tag
					return nil, result.Error
				} else {
					// If the tag exists, use the existing tag
					processedTags = append(processedTags, &existingTag) // Take pointer
				}
			}
			// Correctly associate the processed tags with the specific spec run
			testRun.SuiteRuns[i].SpecRuns[j].Tags = processedTags
		}
	}

	return &processtags.ProcessTagsResponse{
		ErrorMessage: "Tags processed successfully",
	}, nil
}

// Convert via JSON
func convertTestRun(source *createtestrun.TestRun) (*processtags.TestRun, error) {
	jsonBytes, err := json.Marshal(source) // Serialize source
	if err != nil {
		return nil, err
	}

	var target processtags.TestRun
	err = json.Unmarshal(jsonBytes, &target) // Deserialize into target
	if err != nil {
		return nil, err
	}

	return &target, nil
}

func (s *testRunServiceServer) CreateTestRun(ctx context.Context, req *createtestrun.CreateTestRunRequest) (*createtestrun.CreateTestRunResponse, error) {
	testRun := req.GetTestRun()

	// Check if it's a new record
	isNewRecord := testRun.GetId() == 0

	// If not a new record, check if it exists
	if !isNewRecord {
		var existingTestRun models.TestRun
		if err := s.db.Where("id = ?", testRun.GetId()).First(&existingTestRun).Error; err != nil {
			return &createtestrun.CreateTestRunResponse{Success: false, ErrorMessage: "record not found"}, err
		}
	}

	mappedTestRun, err := convertTestRun(testRun)
	if err != nil {
		return nil, err // Handle conversion error
	}

	// Process tags (assuming ProcessTags function exists)
	response, err := ProcessTags(s.db, mappedTestRun)
	if err != nil {
		return &createtestrun.CreateTestRunResponse{
			Success: false,
			//	ErrorMessage: err.Error(),
			ErrorMessage: response.ErrorMessage,
		}, err
	}

	// Save or update the TestRun record
	testRunModel := models.TestRun{
		ID:              uint64(testRun.GetId()),
		TestProjectName: testRun.GetName(),
		// Map other fields as needed
	}

	if err := s.db.Save(&testRunModel).Error; err != nil {
		return &createtestrun.CreateTestRunResponse{Success: false, ErrorMessage: "error saving record"}, err
	}

	// Return the saved test run as part of the response
	return &createtestrun.CreateTestRunResponse{
		Success: true,
		TestRun: &createtestrun.TestRun{Id: int64(testRunModel.ID), Name: testRunModel.TestProjectName}, // Map other fields
	}, nil
}

func StartGRPCServer(context context.Context) {
	//	lis, err := net.Listen("tcp", ":50051") // Use the desired gRPC port
	lis, err := net.Listen("tcp", "0.0.0.0:50051")

	if err != nil {
		utils.GetLogger().Fatal("[gRPC-ERROR]: Failed to listen on port 50051: ", err)
	}

	s := grpc.NewServer()
	pb.RegisterReporterServer(s, &grpcServer{})

	// testid starts here
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		utils.GetLogger().Fatal("[gRPC-ERROR]: Failed to connect to database: ", err)
	}
	pb.RegisterPingServiceServer(s, &server{}) // Register server

	// Register the reporttestrunbyid service
	reporttestrunbyid.RegisterTestRunServiceServer(s, &TestRunServiceServerid{db: db})
	// Register the service
	reporttestrunall.RegisterTestRunServiceServer(s, &TestRunServiceServer{db: db})
	//deletetestrun
	deletetestrun.RegisterTestRunServiceServer(s, &TestRunServiceServerDelete{db: db})
	//updatetestrun
	updatetestrun.RegisterTestRunServiceServer(s, &Server{db: db})
	//gettestrunall and gettetsrunbyid
	gtid.RegisterTestRunServiceServer(s, &servertestbyid{db: db})
	if err := s.Serve(lis); err != nil {
		utils.GetLogger().Fatal("[gRPC-ERROR]: Failed to serve gRPC: ", err)
	}

	testService := &TestServiceServer{db: db}
	gettestrunall.RegisterTestServiceServer(s, testService)
	// processtags
	tagService := &tagServiceServer{db: db}
	// Register the service
	processtags.RegisterTagServiceServer(s, tagService)
	// createtestrun
	testRunService := &testRunServiceServer{db: db}
	createtestrun.RegisterTestRunServiceServer(s, testRunService)

	// Enable reflection for testing
	reflection.Register(s)

	// Run the gRPC server in a goroutine
	go func() {
		utils.GetLogger().Info("[gRPC-LOG]: gRPC server is starting on port 50051")
		if err := s.Serve(lis); err != nil {
			utils.GetLogger().Fatal("[gRPC-ERROR]: Failed to serve gRPC: ", err)
		}
	}()

}
