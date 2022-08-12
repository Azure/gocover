package cmd

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/gtest"
	"github.com/Azure/gocover/pkg/report"
	"github.com/spf13/cobra"
	"golang.org/x/tools/cover"
)

var (
	diffLong = `Generate diff coverage for go code unit test.

Use this tool to generate diff coverage for go code between two branches.
and make sure that new changes has the expected test coverage, and won't affect old codes.
`

	diffExample = `# Generate diff coverage report in HTML format, and coverage baseline should be greater than 80%
gocover diff --cover-profile=coverage.out --compare-branch=origin/master --format html --coverage-baseline 80.0 --output /tmp

# Generate diff coverage report and send the coverage data to kusto database.
export KUSTO_TENANT_ID=00000000-0000-0000-0000-000000000000
export KUSTO_CLIENT_ID=00000000-0000-0000-0000-000000000000
export KUSTO_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxx
gocover diff --cover-profile=coverage.out --compare-branch=origin/master --format html --coverage-baseline 80.0 --output /tmp \
	--host-path github.com/Azure/gocover \
	--data-collection-enabled \
	--endpoint https://your.kusto.windows.net/ \
	--database kustodb_name \
	--event kusto_event
`

	fullLong = `Generate coverage for go code unit test.

Use this tool to generate coverage for go code at the module level.
`

	fullExample = `# Generate full coverage for go code at module level.
gocover full --cover-profile coverage.out --host-path github.com/Azure/gocover

# # Generate full coverage for go code at module level and send the coverage data to kusto database.
export KUSTO_TENANT_ID=00000000-0000-0000-0000-000000000000
export KUSTO_CLIENT_ID=00000000-0000-0000-0000-000000000000
export KUSTO_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxx
gocover full --cover-profile coverage.out --host-path github.com/Azure/gocover \
	--host-path github.com/Azure/gocover \
	--data-collection-enabled \
	--endpoint https://your.kusto.windows.net/ \
	--database kustodb_name \
	--event kusto_event
`
)

var dbOption = &DBOption{
	Writer: &bytes.Buffer{},
}

// NewGoCoverCommand creates a command object for generating diff coverage reporter.
func NewGoCoverCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:          "gocover",
		Short:        "coverage tool for go code",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			dbOption.Writer = cmd.OutOrStdout()
			if err := dbOption.Validate(); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&dbOption.DataCollectionEnabled, "data-collection-enabled", false, "whether or not enable collecting coverage data")
	cmd.PersistentFlags().StringVar((*string)(&dbOption.DbType), "", string(dbclient.Kusto), "db client type, default: kusto")
	cmd.PersistentFlags().StringVar(&dbOption.KustoOption.Endpoint, "endpoint", "", "kusto endpoint")
	cmd.PersistentFlags().StringVar(&dbOption.KustoOption.Database, "database", "", "kusto database")
	cmd.PersistentFlags().StringVar(&dbOption.KustoOption.Event, "event", "", "kusto event")
	cmd.PersistentFlags().StringSliceVar(&dbOption.KustoOption.CustomColumns, "custom-columns", []string{}, "custom kusto columns, format: {column}:{datatype}:{value}")

	cmd.AddCommand(newDiffCoverageCommand())
	cmd.AddCommand(newFullCoverageCommand())
	cmd.AddCommand(newTestCommand())
	return cmd
}

func newDiffCoverageCommand() *cobra.Command {
	o := NewDiffOptions()

	cmd := &cobra.Command{
		Use:     "diff",
		Short:   "generate diff coverage for go code unit test",
		Long:    diffLong,
		Example: diffExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Run(cmd, args, *dbOption); err != nil {
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
	cmd.Flags().StringVar(&o.ModulePath, "host-path", "", "host path for the go project")

	cmd.MarkFlagRequired("cover-profile")

	return cmd
}

func newFullCoverageCommand() *cobra.Command {
	var (
		modulePath     string
		repositoryPath string
		coverProfile   string
	)

	cmd := &cobra.Command{
		Use:     "full",
		Short:   "generate coverage for go code unit test",
		Long:    fullLong,
		Example: fullExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := cover.ParseProfiles(coverProfile)
			if err != nil {
				return fmt.Errorf("parse %s: %s", coverProfile, err)
			}

			fullCoverage, err := report.NewFullCoverage(profiles, modulePath, repositoryPath, []string{}, cmd.OutOrStdout())
			if err != nil {
				return fmt.Errorf("new full coverage: %s", err)
			}

			all, err := fullCoverage.BuildFullCoverageTree()
			if err != nil {
				return fmt.Errorf("build full coverage tree: %s", err)
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
						ModulePath:       modulePath,
						FilePath:         info.Path,
						Coverage:         coverage,
						CoverageMode:     string(dbclient.FullCoverage),
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
	cmd.Flags().StringVar(&repositoryPath, "repository-path", "./", `the root directory of git repository`)
	cmd.Flags().StringVar(&modulePath, "host-path", "", "host path for the go project")

	cmd.MarkFlagRequired("cover-profile")

	return cmd
}

func newTestCommand() *cobra.Command {
	var (
		repositoryPath string
		compareBranch  string
	)
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			gocoverTest, err := gtest.NewGocoverTest(repositoryPath, compareBranch, cmd.OutOrStdout())
			if err != nil {
				return fmt.Errorf("new gocover test: %s", err)
			}

			err = gocoverTest.CheckGoTestFiles()
			if err != nil {
				return fmt.Errorf("check go test files: %s", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repositoryPath, "repository-path", "./", `the root directory of git repository`)
	cmd.Flags().StringVar(&compareBranch, "compare-branch", "origin/master", `branch to compare`)
	return cmd
}
