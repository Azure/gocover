package gocover

import (
	"errors"
	"io"

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
	OutputDir        string
	Excludes         []string
	Style            string

	DbOption *dbclient.DBOption

	Logger logrus.FieldLogger
}

// NewDiffOption returns a Full Option with default values.
func NewFullOption() *FullOption {
	return &FullOption{
		CoverageBaseline: DefaultCoverageBaseline,
		ReportFormat:     DefaultReportFormat,
	}
}

func (o *FullOption) Validate() error {
	return o.DbOption.Validate()
}

// DiffOption contains the input to the gocover diff command.
type DiffOption struct {
	CoverProfiles  []string
	CompareBranch  string
	RepositoryPath string
	ModuleDir      string
	ModulePath     string

	CoverageBaseline float64
	ReportFormat     string
	ReportName       string
	OutputDir        string
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
		ReportFormat:     DefaultReportFormat,
	}
}

func (o *DiffOption) Validate() error {
	return o.DbOption.Validate()
}

type CoverageMode string
type ExecutorMode string

const (
	FullCoverage CoverageMode = "full"
	DiffCoverage CoverageMode = "diff"

	GoExecutor     ExecutorMode = "go"
	GinkgoExecutor ExecutorMode = "ginkgo"
)

var ErrUnknownCoverageMode = errors.New("unknown coverage mode")
var ErrUnknownExecutorMode = errors.New("unknown executor mode")

// GoCoverTestOption contains the input to the gocover govtest command.
type GoCoverTestOption struct {
	CoverProfiles  []string
	CompareBranch  string
	RepositoryPath string
	ModuleDir      string
	ModulePath     string
	CoverageMode   CoverageMode
	ExecutorMode   ExecutorMode

	CoverageBaseline float64
	ReportFormat     string
	ReportName       string
	OutputDir        string
	Excludes         []string
	Style            string

	DbOption *dbclient.DBOption

	StdOut io.Writer
	StdErr io.Writer
	Logger logrus.FieldLogger
}

// NewGoCoverTestOption returns a Options with default values.
func NewGoCoverTestOption() *GoCoverTestOption {
	return &GoCoverTestOption{
		CompareBranch:    DefaultCompareBranch,
		CoverageBaseline: DefaultCoverageBaseline,
		ReportFormat:     DefaultReportFormat,
	}
}
