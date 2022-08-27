package cmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/gocover"
	"github.com/Azure/gocover/pkg/gtest"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

var dboption = &dbclient.DBOption{}

const (
	DefaultCoverageBaseline = 80.0
	FlagVerbose             = "verbose"
	FlagVerboseShort        = "v"
)

func createLogger(cmd *cobra.Command) *logrus.Logger {
	logger := logrus.New()
	verbose, err := cmd.Flags().GetBool(FlagVerbose)
	if err != nil {
		// no verbose flag on the command, It's OK.
		verbose = false
	}
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	}
	return logger
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

	cmd.PersistentFlags().BoolP(FlagVerbose, FlagVerboseShort, false, "verbose output")

	cmd.PersistentFlags().BoolVar(&dboption.DataCollectionEnabled, "data-collection-enabled", false, "whether or not enable collecting coverage data")
	cmd.PersistentFlags().StringVar((*string)(&dboption.DbType), "store-type", string(dbclient.None), "db client type")
	cmd.PersistentFlags().StringVar(&dboption.KustoOption.Endpoint, "endpoint", "", "kusto endpoint")
	cmd.PersistentFlags().StringVar(&dboption.KustoOption.Database, "database", "", "kusto database")
	cmd.PersistentFlags().StringVar(&dboption.KustoOption.Event, "event", "", "kusto event")
	cmd.PersistentFlags().StringSliceVar(&dboption.KustoOption.CustomColumns, "custom-columns", []string{}, "custom kusto columns, format: {column}:{datatype}:{value}")

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
	o := gocover.NewFullOption()

	cmd := &cobra.Command{
		Use:     "full",
		Short:   "generate coverage for go code unit test",
		Long:    fullLong,
		Example: fullExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Logger = createLogger(cmd)
			o.DbOption = dboption

			full, err := gocover.NewFullCover(o)
			if err != nil {
				return fmt.Errorf("NewFullCover: %w", err)
			}

			ctx := context.Background()
			if err := full.Run(ctx); err != nil {
				return fmt.Errorf("generate full coverage: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&o.CoverProfiles, "cover-profile", []string{}, `coverage profiles produced by 'go test'`)
	cmd.Flags().StringVar(&o.RepositoryPath, "repository-path", "./", `the root directory of git repository`)
	cmd.Flags().StringVar(&o.ReportFormat, "format", o.ReportFormat, "format of the diff coverage report, one of: html, json, markdown")
	cmd.Flags().StringSliceVar(&o.Excludes, "excludes", []string{}, "exclude files for diff coverage calucation")
	cmd.Flags().StringVarP(&o.Output, "output", "o", o.Output, "diff coverage output file")
	cmd.Flags().Float64Var(&o.CoverageBaseline, "coverage-baseline", o.CoverageBaseline, "returns an error code if coverage or quality score is less than coverage baseline")
	cmd.Flags().StringVar(&o.ReportName, "report-name", "coverage", "diff coverage report name")
	cmd.Flags().StringVar(&o.Style, "style", "colorful", "coverage report code format style, refer to https://pygments.org/docs/styles for more information")
	cmd.Flags().StringVar(&o.ModuleDir, "module-path", "", "module path for the go project")

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

			err = gocoverTest.EnsureGoTestFiles()
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
