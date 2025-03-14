package handlers

import (
	"github.com/guidewire/fern-reporter/pkg/models"
	"time"
)

const timeQueryLayout = "2006-01-02T15:04:05"

func GetLongestTestRuns(h *Handler, projectName string, startTimeRange time.Time, endTimeRange time.Time) []models.TestRunInsight {
	var testRuns []models.TestRunInsight

	h.db.Table("test_runs").
		Joins("INNER JOIN suite_runs ON test_runs.id = suite_runs.test_run_id").
		Joins("INNER JOIN spec_runs ON suite_runs.id = spec_runs.suite_id").
		Select("suite_runs.id, test_runs.test_project_name, test_runs.start_time, test_runs.end_time,"+
			"ROUND(AVG(CASE WHEN spec_runs.status = 'passed' THEN 100.0 ELSE 0.0 END), 3) AS pass_rate, "+
			"(test_runs.end_time - test_runs.start_time) AS duration").
		Where("test_runs.start_time >= ?", startTimeRange).
		Where("test_runs.start_time <= ?", endTimeRange).
		Where("test_project_name = ?", projectName).
		Group("suite_runs.id, test_runs.test_project_name, test_runs.start_time, test_runs.end_time").
		Order("duration DESC").
		Find(&testRuns)

	return testRuns
}

func GetAverageDuration(h *Handler, projectName string, startTimeRange time.Time, endTimeRange time.Time) float64 {
	var averageDuration float64
	h.db.Table("test_runs").
		Select("AVG(EXTRACT(EPOCH FROM (end_time - start_time)))").
		Where("test_project_name = ?", projectName).
		Where("start_time >= ?", startTimeRange).
		Where("start_time <= ?", endTimeRange).
		Scan(&averageDuration)
	return averageDuration
}

func ParseTimeFromStringWithDefault(timeString string, defaultTime time.Time) (time.Time, error) {
	if timeString == "" {
		return defaultTime, nil
	}

	parsedTime, err := time.Parse(timeQueryLayout, timeString)

	if err != nil {
		return time.Now(), err
	}
	return parsedTime, nil
}

func GetProjectSpecStatistics(h *Handler, projectId string) []models.TestSummary {
	var testSummaries []models.TestSummary
	h.db.Table("test_runs").
		Joins("INNER JOIN suite_runs ON test_runs.id = suite_runs.test_run_id").
		Joins("INNER JOIN spec_runs ON suite_runs.id = spec_runs.suite_id").
		Select(`suite_runs.id AS suite_run_id, 
			suite_runs.suite_name,
            test_runs.start_time, 
            COUNT(spec_runs.id) FILTER (WHERE spec_runs.status = 'passed') AS total_passed_spec_runs, 
			COUNT(spec_runs.id) FILTER (WHERE spec_runs.status = 'skipped') AS total_skipped_spec_runs, 
            COUNT(spec_runs.id) AS total_spec_runs`).
		Where("test_runs.project_id = ?", projectId).
		Group("suite_runs.id, test_runs.start_time").
		Order("test_runs.start_time").
		Scan(&testSummaries)
	return testSummaries
}
