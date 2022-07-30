package dbclient

import (
	"context"
	"time"
)

type ClientType string

const (
	Kusto ClientType = "Kusto"
)

type CoverageMode string

const (
	FullCoverage CoverageMode = "full"
	DiffCoverage CoverageMode = "diff"
)

// DbClient interface for storing gocover data.
type DbClient interface {
	Store(context context.Context, data *Data) error
}

type Data struct {
	PreciseTimestamp time.Time `json:"preciseTimestamp"` // time send to db
	TotalLines       int64     `json:"totalLines"`       // total lines of the entire repo/module.
	EffectiveLines   int64     `json:"effectiveLines"`   // the lines for coverage base, total lines - ignored lines
	IgnoredLines     int64     `json:"ignoredLines"`     // the lines ignored.
	CoveredLines     int64     `json:"coveredLines"`     // the lines covered by test
	Coverage         float64   `json:"coverage"`         // unit test coverage, CoveredLines / EffectiveLines
	CoverageMode     string    `json:"coverageMode"`     // coverage mode, diff or full subcommand
	ModulePath       string    `json:"modulePath"`       // module name, which is declared in go.mod
	FilePath         string    `json:"filePath"`         // file path for a concrete file or directory

	Extra map[string]interface{} // extra data that passing accordingly
}
