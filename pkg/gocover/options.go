package gocover

import (
	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/sirupsen/logrus"
)

// FullOption contains the input for gocover full command.
type FullOption struct {
	CoverProfiles  []string
	RepositoryPath string
	ModuleDir      string

	CoverageBaseline float64
	ReportFormat     string
	ReportName       string
	Output           string
	Excludes         []string
	Style            string

	DbOption *dbclient.DBOption

	Logger logrus.FieldLogger
}

// NewDiffOption returns a Full Option with default values.
func NewFullOption() *FullOption {
	return &FullOption{
		CoverageBaseline: DefaultCoverageBaseline,
		ReportFormat:     "html",
	}
}

func (o *FullOption) Validate() error {
	return o.DbOption.Validate()
}

// DiffOptions contains the input to the gocover command.
type DiffOption struct {
	CoverProfiles  []string
	CompareBranch  string
	RepositoryPath string
	ModuleDir      string
	ModulePath     string

	CoverageBaseline float64
	ReportFormat     string
	ReportName       string
	Output           string
	Excludes         []string
	Style            string

	DbOption *dbclient.DBOption

	Logger logrus.FieldLogger
}

// NewDiffOptions returns a Options with default values.
func NewDiffOption() *DiffOption {
	return &DiffOption{
		CompareBranch:    DefaultCompareBranch,
		CoverageBaseline: DefaultCoverageBaseline,
		ReportFormat:     "html",
	}
}

func (o *DiffOption) Validate() error {
	return o.DbOption.Validate()
}
