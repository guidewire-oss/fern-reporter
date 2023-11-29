package models

import (
	"time"
)

type TimeLog struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type TestRun struct {
	ID        uint64     `json:"id" gorm:"primaryKey"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
	SuiteRuns []SuiteRun `json:"suite_runs" gorm:"foreignKey:TestRunID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type SuiteRun struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	TestRunID uint64    `json:"test_run_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	SpecRuns  []SpecRun `json:"spec_runs" gorm:"foreignKey:SuiteID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type SpecRun struct {
	ID              uint64    `json:"id" gorm:"primaryKey"`
	SuiteID         uint64    `json:"suite_id"`
	SpecDescription string    `json:"spec_description"`
	Status          string    `json:"status"`
	Message         string    `json:"message"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
}
