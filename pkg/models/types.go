package models

import (
	"time"
)

type TimeLog struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
}

type TestRun struct {
	Id        uint64     `json:"id"`
	SuiteRuns []SuiteRun `json:"suite_runs"`
	TimeLog
}

type SuiteRun struct {
	SpecRuns []SpecRun `json:"spec_run"`
	TimeLog
}

type SpecRun struct {
	SpecDescription string `json:"spec_description"`
	Status          string `json:"status"`
	Message         string `json:message`
	TimeLog
}
