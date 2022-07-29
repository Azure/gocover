package dbclient

import (
	"context"
	"time"
)

type ClientType string

const (
	Kusto ClientType = "Kusto"
)

// DbClient interface for storing gocover data.
type DbClient interface {
	Store(context context.Context, data *Data) error
}

type Data struct {
	PreciseTimestamp time.Time `json:"preciseTimestamp"` // time send to db
	LinesCovered     int64     `json:"linesCovered"`     // unit test covered lines
	LinesValid       int64     `json:"linesValid"`       // unit test total lines
	Coverage         float64   `json:"coverage"`         // unit test coverage, LinesCovered / LinesValid
	ModulePath       string    `json:"modulePath"`       // module name, which is declared in go.mod
	FilePath         string    `json:"filePath"`         // file path for a concrete file or directory

	Extra map[string]interface{} // extra data that passing accordingly
}
