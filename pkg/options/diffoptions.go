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
	CoverProfile  string
	CompareBranch string

	FailureRate  float64
	ReportFormat string
	Output       string
	Exclude      []string
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

	gitClient := gittool.NewGitClient()
	patch, err := gitClient.DiffCommitted(o.CompareBranch)
	if err != nil {
		return fmt.Errorf("git diff: %w", err)
	}

	diff := report.NewDiffCoverageReport(o.Exclude)
	if err := diff.DiffCoverage(profiles, patch); err != nil {
		return fmt.Errorf("diff coverage %w", err)
	}
	if err := diff.GenerateReport(profiles); err != nil {
		return fmt.Errorf("generate report %w", err)
	}

	return nil
}
