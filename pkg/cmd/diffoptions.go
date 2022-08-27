package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/gittool"
	"github.com/Azure/gocover/pkg/report"
	"github.com/sirupsen/logrus"
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
	ModulePath     string

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
func (o *DiffOptions) Run(cmd *cobra.Command, args []string, dbopt DBOption) error {
	o.Writer = cmd.OutOrStdout()
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

	diffCoverage, err := report.NewDiffCoverage(profiles, changes, o.Excludes, o.CompareBranch, o.RepositoryPath, o.ModulePath)
	if err != nil {
		return fmt.Errorf("new diff converage: %w", err)
	}
	statistics, all, err := diffCoverage.GenerateDiffCoverage()
	if err != nil {
		return fmt.Errorf("diff coverage: %w", err)
	}

	g := report.NewReportGenerator(o.Style, o.Output, o.ReportName, logrus.New())
	if err := g.GenerateReport(statistics); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	var finalError error = nil
	if statistics.TotalCoveragePercent < o.CoverageBaseline {
		finalError = fmt.Errorf("the coverage baseline pass rate is %.2f, currently is %.2f",
			o.CoverageBaseline,
			statistics.TotalCoveragePercent,
		)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", "Summary of coverage:")
	var coverage float64
	for _, info := range all {
		if info.TotalEffectiveLines == 0 {
			coverage = 100.0
		} else {
			coverage = float64(info.TotalCoveredLines) / float64(info.TotalEffectiveLines) * 100
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s %d %d %d %d %.1f%%\n",
			info.Path,
			info.TotalEffectiveLines,
			info.TotalCoveredLines,
			info.TotalIgnoredLines,
			info.TotalLines,
			coverage,
		)
	}

	if dbOption.DataCollectionEnabled {
		dbClient, err := dbOption.GetDbClient()
		if err != nil {
			return fmt.Errorf("new db client: %w", err)
		}

		now := time.Now().UTC()
		for _, info := range all {
			if info.TotalEffectiveLines == 0 {
				coverage = 100.0
			} else {
				coverage = float64(info.TotalCoveredLines) / float64(info.TotalEffectiveLines) * 100
			}

			err = dbClient.Store(context.Background(), &dbclient.Data{
				PreciseTimestamp: now,
				TotalLines:       info.TotalLines,
				EffectiveLines:   info.TotalEffectiveLines,
				IgnoredLines:     info.TotalIgnoredLines,
				CoveredLines:     info.TotalCoveredLines,
				ModulePath:       o.ModulePath,
				FilePath:         info.Path,
				Coverage:         coverage,
				CoverageMode:     string(dbclient.DiffCoverage),
			})
			if err != nil {
				return fmt.Errorf("store data: %w", err)
			}
		}
	}

	return finalError
}
