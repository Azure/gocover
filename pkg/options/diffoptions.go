package options

import (
	"fmt"

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

	FailureRate  float64
	ReportFormat string
	ReportName   string
	Output       string
	Excludes     []string
	Style        string
}

// NewDiffOptions returns a Options with default values.
// TODO: make format as enumerate string type
func NewDiffOptions() *DiffOptions {
	return &DiffOptions{
		CompareBranch: DefaultCompareBranch,
		ReportFormat:  "html",
		FailureRate:   20.0,
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

	diffCoverage := report.NewDiffCoverage(profiles, changes, o.Excludes, o.CompareBranch)
	statistics, err := diffCoverage.GenerateDiffCoverage()
	if err != nil {
		return fmt.Errorf("diff coverage: %w", err)
	}

	g := report.NewReportGenerator(statistics, o.Style, o.Output, o.ReportName)
	if err := g.GenerateReport(); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	if 100-statistics.TotalCoveragePercent >= int(o.FailureRate) {
		return fmt.Errorf("total coverage pass rate is: %d", statistics.TotalCoveragePercent)
	}

	return nil
}
