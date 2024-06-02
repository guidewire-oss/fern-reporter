package models

import (
	"time"
)

type TimeLog struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type TestRun struct {
	ID              uint64     `json:"id" gorm:"primaryKey"`
	TestProjectName string     `json:"test_project_name"`
	TestSeed        uint64     `json:"test_seed"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         time.Time  `json:"end_time"`
	SuiteRuns       []SuiteRun `json:"suite_runs" gorm:"foreignKey:TestRunID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type SuiteRun struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	TestRunID uint64    `json:"test_run_id"`
	SuiteName string    `json:"suite_name"`
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
	Tags            []Tag     `json:"tags" gorm:"many2many:spec_run_tags;"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
}

type TestRunInsight struct {
	SuiteID         uint64    `json:"suite_id" gorm:"column:id"`
	TestProjectName string    `json:"test_project_name"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	PassRate        float32   `json:"pass_rate"`
}

type Tag struct {
	ID   uint64 `json:"id" gorm:"primaryKey"`
	Name string `json:"name"`
}
