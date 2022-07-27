package cmd

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/report"
	"github.com/spf13/cobra"
	"golang.org/x/tools/cover"
)

// TODO: improve this description.
var (
	getLong = `
Generate unit test diff coverage for go code.

Use this tool to generate diff coverage for go code between two branch.
In this way to make sure that new changes has a expected test coverage, and won't affect old codes.
`

	getExample = `# Generate diff coverage report in HTML format, and coverage baseline should be greater than 80%
gocover --cover-profile=coverage.out --compare-branch=origin/master --format html --coverage-baseline 80.0 --output coverage.html
`
)

// NewGoCoverCommand creates a command object for generating diff coverage reporter.
func NewGoCoverCommand() *cobra.Command {
	o := NewDiffOptions()

	cmd := &cobra.Command{
		Use:          "gocover",
		Short:        "Generate unit test diff coverage for go code",
		Long:         getLong,
		Example:      getExample,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Writer = cmd.OutOrStdout()
			if err := o.Run(cmd, args); err != nil {
				return fmt.Errorf("generate diff coverage %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&o.CoverProfile, "cover-profile", o.CoverProfile, `coverage profile produced by 'go test'`)
	cmd.Flags().StringVar(&o.CompareBranch, "compare-branch", o.CompareBranch, `branch to compare`)
	cmd.Flags().StringVar(&o.RepositoryPath, "repository-path", "./", `the root directory of git repository`)
	cmd.Flags().StringVar(&o.ReportFormat, "format", o.ReportFormat, "format of the diff coverage report, one of: html, json, markdown")
	cmd.Flags().StringSliceVar(&o.Excludes, "excludes", []string{}, "exclude files for diff coverage calucation")
	cmd.Flags().StringVarP(&o.Output, "output", "o", o.Output, "diff coverage output file")
	cmd.Flags().Float64Var(&o.CoverageBaseline, "coverage-baseline", o.CoverageBaseline, "returns an error code if coverage or quality score is less than coverage baseline")
	cmd.Flags().StringVar(&o.ReportName, "report-name", "coverage", "diff coverage report name")
	cmd.Flags().StringVar(&o.Style, "style", "colorful", "coverage report code format style, refer to https://pygments.org/docs/styles for more information")

	cmd.MarkFlagRequired("cover-profile")

	cmd.AddCommand(newFullCoverageCommand())
	return cmd
}

func newFullCoverageCommand() *cobra.Command {
	var (
		coverProfile   string
		moduleHostPath string
	)

	dbOption := &DBOption{
		Writer: &bytes.Buffer{},
	}

	cmd := &cobra.Command{
		Use: "full",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := dbOption.Validate(); err != nil {
				return err
			}
			dbOption.Writer = cmd.OutOrStdout()

			profiles, err := cover.ParseProfiles(coverProfile)
			if err != nil {
				return fmt.Errorf("parse %s: %s", coverProfile, err)
			}

			fullCoverage, err := report.NewFullCoverage(profiles, moduleHostPath, []string{})
			if err != nil {
				return fmt.Errorf("new full coverage: %s", err)
			}

			all := fullCoverage.BuildFullCoverageTree()
			for _, info := range all {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %d %d %.1f%%\n",
					info.Path,
					info.TotalCoveredLines,
					info.TotalLines,
					float64(info.TotalCoveredLines)/float64(info.TotalLines)*100,
				)
			}

			if dbOption.DataCollectionEnabled {
				dbClient, err := dbOption.GetDbClient()
				if err != nil {
					return fmt.Errorf("new db client: %w", err)
				}

				now := time.Now().UTC()
				for _, info := range all {
					err = dbClient.Store(context.Background(), &dbclient.Data{
						PreciseTimestamp: now,
						LinesCovered:     info.TotalCoveredLines,
						LinesValid:       info.TotalLines,
						ModulePath:       moduleHostPath,
						FilePath:         info.Path,
						Coverage:         float64(info.TotalCoveredLines) / float64(info.TotalLines),
					})
					if err != nil {
						return fmt.Errorf("store data: %w", err)
					}
				}

			}

			return nil
		},
	}

	cmd.Flags().StringVar(&coverProfile, "cover-profile", "", `coverage profile produced by 'go test'`)
	cmd.Flags().StringVar(&moduleHostPath, "host-path", "", "host path for the go project")
	cmd.Flags().BoolVar(&dbOption.DataCollectionEnabled, "data-collection-enabled", false, "whether or not enable collecting coverage data")
	cmd.Flags().StringVar((*string)(&dbOption.DbType), "", string(dbclient.Kusto), "db client type, default: kusto")
	cmd.Flags().StringVar(&dbOption.KustoOption.Endpoint, "endpoint", "", "kusto endpoint")
	cmd.Flags().StringVar(&dbOption.KustoOption.Database, "database", "", "kusto database")
	cmd.Flags().StringVar(&dbOption.KustoOption.Event, "event", "", "kusto event")
	cmd.Flags().StringSliceVar(&dbOption.KustoOption.CustomColumns, "custom-columns", []string{}, "custom kusto columns, format: {column}:{datatype}:{value}")
	cmd.MarkFlagRequired("cover-profile")

	return cmd
}
