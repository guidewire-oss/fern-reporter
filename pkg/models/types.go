package models

import (
	"time"
)

type TimeLog struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type TestRunStatus string

const (
	StatusFailed  TestRunStatus = "FAILED"
	StatusSkipped TestRunStatus = "SKIPPED"
	StatusPassed  TestRunStatus = "PASSED"
)

type TestRun struct {
	ID                uint64        `json:"id" gorm:"primaryKey"`
	TestProjectName   string        `json:"test_project_name"`
	TestProjectID     string        `json:"test_project_id" gorm:"-"`
	ProjectID         uint64        `json:"project_id" gorm:"column:project_id"` // Foreign key
	TestSeed          uint64        `json:"test_seed"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
	GitBranch         string        `json:"git_branch"`
	GitSha            string        `json:"git_sha"`
	BuildTriggerActor string        `json:"build_trigger_actor"`
	BuildUrl          string        `json:"build_url"`
	SuiteRuns         []SuiteRun    `json:"suite_runs" gorm:"foreignKey:TestRunID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Status            TestRunStatus `gorm:"type:test_run_status"`

	// Relationship with ProjectDetails
	Project ProjectDetails `gorm:"foreignKey:ProjectID;references:ID"`
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

type TestSummary struct {
	SuiteRunID           uint
	SuiteName            string
	TestProjectName      string
	StartTime            time.Time
	TotalPassedSpecRuns  int64
	TotalSkippedSpecRuns int64
	TotalSpecRuns        int64
}

type Tag struct {
	ID   uint64 `json:"id" gorm:"primaryKey"`
	Name string `json:"name"`
}

type ProjectDetails struct {
	ID        uint64    `json:"-" gorm:"primaryKey"`
	UUID      string    `json:"uuid" gorm:"->;column:uuid"`
	Name      string    `json:"name"`
	TeamName  string    `json:"team_name"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type PreferredProject struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	UserID    uint64
	ProjectID uint64
	GroupID   *uint64 // Nullable field (for ungrouped)

	User    AppUser        `gorm:"foreignKey:ID;constraint:OnDelete:CASCADE"`
	Project ProjectDetails `gorm:"foreignKey:ProjectID;references:ID;constraint:OnDelete:CASCADE"`
	Group   *ProjectGroup  `gorm:"foreignKey:GroupID;references:GroupID;constraint:OnDelete:SET NULL"`
}

type ProjectGroup struct {
	GroupID   uint64 `gorm:"primaryKey;autoIncrement;column:group_id"`
	UserID    uint64
	GroupName string
}

type AppUser struct {
	ID        uint64    `gorm:"primaryKey"`
	IsDark    bool      `gorm:"column:is_dark;default:false"`
	Timezone  string    `gorm:"size:40"`
	Cookie    string    `gorm:"size:40;index:idx_app_user_cookie"`
	CreatedAt time.Time `gorm:"autoCreateTime"` // set once when created
	UpdatedAt time.Time `gorm:"autoUpdateTime"` // updated automatically on update
}
