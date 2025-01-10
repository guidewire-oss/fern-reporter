package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.55

import (
	"context"

	"github.com/guidewire/fern-reporter/pkg/graph/generated"
	"github.com/guidewire/fern-reporter/pkg/graph/modelv2"
	"github.com/guidewire/fern-reporter/pkg/utils"
)

// TestRuns is the resolver for the testRuns field.
func (r *queryResolver) TestRuns(ctx context.Context, first *int, after *string, desc *bool) (*modelv2.TestRunConnection, error) {
	// Convert the `after` cursor to an offset.
	offset := utils.DecodeCursor(after)

	order := "id ASC" // Default to ascending order
	if desc != nil && *desc {
		order = "id DESC"
	}

	var testRuns []*modelv2.TestRun
	if err := r.DB.Preload("SuiteRuns.SpecRuns.Tags").Offset(offset).Limit(*first).Order(order).Find(&testRuns).Error; err != nil {
		return nil, err
	}

	// Get the total count of TestRun records.
	var totalCount int64
	if err := r.DB.Model(&modelv2.TestRun{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	edges := make([]*modelv2.TestRunEdge, len(testRuns))
	for i, run := range testRuns {
		edges[i] = &modelv2.TestRunEdge{
			Cursor:  utils.EncodeCursor(offset + i + 1),
			TestRun: run,
		}
	}

	pageInfo := &modelv2.PageInfo{
		HasNextPage:     len(testRuns) == *first && offset+len(testRuns) < int(totalCount),
		HasPreviousPage: offset > 0,
	}

	// Only set StartCursor and EndCursor if there are test runs
	if len(edges) > 0 {
		pageInfo.StartCursor = edges[0].Cursor
		pageInfo.EndCursor = edges[len(edges)-1].Cursor
	}

	return &modelv2.TestRunConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: int(totalCount),
	}, nil
}

// TestRun is the resolver for the testRun field.
func (r *queryResolver) TestRun(ctx context.Context, testRunFilter modelv2.TestRunFilter) ([]*modelv2.TestRun, error) {
	var testRuns []*modelv2.TestRun
	r.DB.Preload("SuiteRuns.SpecRuns.Tags").Where("id = ?", testRunFilter.ID).Where("test_project_name = ?", testRunFilter.TestProjectName).Find(&testRuns)
	return testRuns, nil
}

// TestRunByID is the resolver for the testRunById field.
func (r *queryResolver) TestRunByID(ctx context.Context, id int) (*modelv2.TestRun, error) {
	var testRun *modelv2.TestRun
	r.DB.Preload("SuiteRuns.SpecRuns.Tags").Where("id = ?", id).First(&testRun)

	return testRun, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
