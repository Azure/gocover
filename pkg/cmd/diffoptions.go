package cmd

import (
	"fmt"
	"io"

	"github.com/Azure/gocover/pkg/gittool"
	"github.com/Azure/gocover/pkg/report"
	"github.com/spf13/cobra"
	"golang.org/x/tools/cover"
)

const (
	DefaultCompareBranch = "origin/master"
)

// DiffOptions contains the input to the gocover command.
type DiffOptions struct {
	CoverProfile   string
	CompareBranch  string
	RepositoryPath string

	CoverageBaseline float64
	ReportFormat     string
	ReportName       string
	Output           string
	Excludes         []string
	Style            string

	Writer io.Writer
}

// NewDiffOptions returns a Options with default values.
// TODO: make format as enumerate string type
func NewDiffOptions() *DiffOptions {
	return &DiffOptions{
		CompareBranch:    DefaultCompareBranch,
		ReportFormat:     "html",
		CoverageBaseline: 80.0,
	}
}

// Run do the actual actions on diff coverage.
// TODO: improve it.
func (o *DiffOptions) Run(cmd *cobra.Command, args []string) error {
	profiles, err := cover.ParseProfiles(o.CoverProfile)
	if err != nil {
		return fmt.Errorf("parse coverage profile: %w", err)
	}

	gitClient, err := gittool.NewGitClient(o.RepositoryPath)
	if err != nil {
		return fmt.Errorf("git repository: %w", err)
	}
	changes, err := gitClient.DiffChangesFromCommitted(o.CompareBranch)
	if err != nil {
		return fmt.Errorf("git diff: %w", err)
	}

	diffCoverage, err := report.NewDiffCoverage(profiles, changes, o.Excludes, o.CompareBranch, o.RepositoryPath)
	if err != nil {
		return fmt.Errorf("new diff converage: %w", err)
	}
	statistics, err := diffCoverage.GenerateDiffCoverage()
	if err != nil {
		return fmt.Errorf("diff coverage: %w", err)
	}

	g := report.NewReportGenerator(statistics, o.Style, o.Output, o.ReportName)
	if err := g.GenerateReport(); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	if statistics.TotalCoveragePercent < o.CoverageBaseline {
		return fmt.Errorf("the coverage baseline pass rate is %.2f, currently is %.2f",
			o.CoverageBaseline,
			statistics.TotalCoveragePercent,
		)
	}

	return nil
}
