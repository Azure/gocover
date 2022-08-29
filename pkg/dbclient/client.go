package dbclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type ClientType string

const (
	None  ClientType = "None"
	Kusto ClientType = "Kusto"
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

var ErrUnsupportedDBType = errors.New(`supportted type are "Kusto", unsupported DB client type`)

type DBOption struct {
	DataCollectionEnabled bool
	DbType                ClientType
	KustoOption           KustoOption
}

func (o *DBOption) Validate() error {
	if !o.DataCollectionEnabled {
		return nil
	}

	if o.DbType == Kusto {
		return o.KustoOption.Validate()
	}
	return fmt.Errorf("%w: %s", ErrUnsupportedDBType, o.DbType)
}

func (o *DBOption) GetDbClient(logger logrus.FieldLogger) (DbClient, error) {
	switch o.DbType {
	case Kusto:
		o.KustoOption.Logger = logger
		return NewKustoClient(&o.KustoOption)
	default:
		return nil, nil
	}
}
