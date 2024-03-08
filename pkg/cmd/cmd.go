package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/gocover"
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
	--coverage-event kusto_event \
	--ignore-event ignore_event
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
	--coverage-event kusto_event \
	--ignore-event ignore_event
`

	gocoverTestLong = `Run unit tests on the module, then apply full coverage or diff coverage calculation on the results.
	`
	gocoverTestExample = "" +
		`# Run unit tests and generate diff coverage result.
gocover test --coverage-mode diff --compare-branch=origin/master --outputdir /tmp

# Run unit tests and generate full coverage result on the whole module.
gocover test --coverage-mode full --outputdir /tmp
`
)

var (
	dbOption         = &dbclient.DBOption{}
	timeoutInSeconds int
)

const (
	FlagVerbose             = "verbose"
	FlagVerboseShort        = "v"
	defaultTimeoutInSeconds = 60 * 60 // 3600 seconds, 1 hour
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
func NewGoCoverCommand(version, commit, date string) *cobra.Command {

	cmd := &cobra.Command{
		Use:          "gocover",
		Short:        "coverage tool for go code",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := dbOption.Validate(); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.PersistentFlags().BoolP(FlagVerbose, FlagVerboseShort, false, "verbose output")

	cmd.PersistentFlags().BoolVar(&dbOption.DataCollectionEnabled, "data-collection-enabled", false, "whether or not enable collecting coverage data")
	cmd.PersistentFlags().StringVar((*string)(&dbOption.DbType), "store-type", string(dbclient.None), "db client type")
	cmd.PersistentFlags().StringVar(&dbOption.KustoOption.Endpoint, "endpoint", "", "kusto endpoint")
	cmd.PersistentFlags().StringVar(&dbOption.KustoOption.Database, "database", "", "kusto database")
	cmd.PersistentFlags().StringVar(&dbOption.KustoOption.CoverageEvent, "coverage-event", "", "kusto event for coverage")
	cmd.PersistentFlags().StringVar(&dbOption.KustoOption.IgnoreEvent, "ignore-event", "", "kusto event for ignore information")
	cmd.PersistentFlags().StringSliceVar(&dbOption.KustoOption.CustomColumns, "custom-columns", []string{}, "custom kusto columns, format: {column}:{datatype}:{value}")
	cmd.PersistentFlags().IntVar(&timeoutInSeconds, "timeout", defaultTimeoutInSeconds, "execute timeout in seconds")

	cmd.AddCommand(newDiffCoverageCommand())
	cmd.AddCommand(newFullCoverageCommand())
	cmd.AddCommand(newGoCoverTestCommand())
	cmd.AddCommand(newVersionCommand(version, commit, date))
	return cmd
}

func newDiffCoverageCommand() *cobra.Command {
	o := gocover.NewDiffOption()
	cmd := &cobra.Command{
		Use:     "diff",
		Short:   "generate diff coverage for go code unit test",
		Long:    diffLong,
		Example: diffExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Logger = createLogger(cmd)
			o.DbOption = dbOption

			diff, err := gocover.NewDiffCover(o)
			if err != nil {
				return fmt.Errorf("NewDiffCover: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeoutInSeconds*time.Second)
			defer cancel()

			if err := diff.Run(ctx); err != nil {
				return fmt.Errorf("generate diff coverage: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&o.CoverProfiles, "cover-profile", []string{}, `coverage profile produced by 'go test'`)
	cmd.Flags().StringVar(&o.CompareBranch, "compare-branch", o.CompareBranch, `branch to compare`)
	cmd.Flags().StringVar(&o.RepositoryPath, "repository-path", "./", `the root directory of git repository`)
	cmd.Flags().StringVar(&o.ModuleDir, "module-dir", "./", "module directory contains go.mod file that relative to the project")
	cmd.Flags().StringVar(&o.ReportFormat, "format", o.ReportFormat, "format of the diff coverage report, one of: html, json, markdown")
	cmd.Flags().StringSliceVar(&o.Excludes, "excludes", []string{}, "exclude files for diff coverage calucation")
	cmd.Flags().StringVarP(&o.OutputDir, "outputdir", "o", o.OutputDir, "diff coverage output directory")
	cmd.Flags().Float64Var(&o.CoverageBaseline, "coverage-baseline", o.CoverageBaseline, "returns an error code if coverage or quality score is less than coverage baseline")
	cmd.Flags().StringVar(&o.ReportName, "report-name", "coverage", "diff coverage report name")
	cmd.Flags().StringVar(&o.Style, "style", "colorful", "coverage report code format style, refer to https://pygments.org/docs/styles for more information")
	cmd.Flags().StringVar(&o.GitHash, "gitHash", "", "gitHash for generating working source code links")

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
			o.DbOption = dbOption

			full, err := gocover.NewFullCover(o)
			if err != nil {
				return fmt.Errorf("NewFullCover: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeoutInSeconds*time.Second)
			defer cancel()

			if err := full.Run(ctx); err != nil {
				return fmt.Errorf("generate full coverage: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&o.CoverProfiles, "cover-profile", []string{}, `coverage profiles produced by 'go test'`)
	cmd.Flags().StringVar(&o.RepositoryPath, "repository-path", "./", `the root directory of git repository`)
	cmd.Flags().StringVar(&o.ModuleDir, "module-dir", "./", "module directory contains go.mod file that relative to the project")
	cmd.Flags().StringVar(&o.ReportFormat, "format", o.ReportFormat, "format of the diff coverage report, one of: html, json, markdown")
	cmd.Flags().StringSliceVar(&o.Excludes, "excludes", []string{}, "exclude files for diff coverage calucation")
	cmd.Flags().StringVarP(&o.OutputDir, "outputdir", "o", o.OutputDir, "diff coverage output directory")
	cmd.Flags().Float64Var(&o.CoverageBaseline, "coverage-baseline", o.CoverageBaseline, "returns an error code if coverage or quality score is less than coverage baseline")
	cmd.Flags().StringVar(&o.ReportName, "report-name", "coverage", "diff coverage report name")
	cmd.Flags().StringVar(&o.Style, "style", "colorful", "coverage report code format style, refer to https://pygments.org/docs/styles for more information")
	cmd.Flags().StringVar(&o.GitHash, "gitHash", "", "gitHash for generating working source code links")

	cmd.MarkFlagRequired("cover-profile")

	return cmd
}

func newGoCoverTestCommand() *cobra.Command {
	o := gocover.NewGoCoverTestOption()

	cmd := &cobra.Command{
		Use:     "test",
		Short:   "run tests and coverage calculation on the module",
		Long:    gocoverTestLong,
		Example: gocoverTestExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Logger = createLogger(cmd)
			o.DbOption = dbOption
			o.StdOut = cmd.OutOrStdout()
			o.StdErr = cmd.ErrOrStderr()

			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeoutInSeconds*time.Second)
			defer cancel()

			t, err := gocover.NewGoCoverTestExecutor(o)
			if err != nil {
				return fmt.Errorf("NewGoCoverTestExecutor: %w", err)
			}
			return t.Run(ctx)
		},
	}

	cmd.Flags().StringSliceVar(&o.CoverProfiles, "cover-profile", []string{}, `coverage profile produced by 'go test'`)
	cmd.Flags().StringVar(&o.CompareBranch, "compare-branch", o.CompareBranch, `branch to compare`)
	cmd.Flags().StringVar(&o.RepositoryPath, "repository-path", "./", `the root directory of git repository`)
	cmd.Flags().StringVar(&o.ModuleDir, "module-dir", "./", "module directory contains go.mod file that relative to the project")
	cmd.Flags().StringVar(&o.ReportFormat, "format", o.ReportFormat, "format of the diff coverage report, one of: html, json, markdown")
	cmd.Flags().StringSliceVar(&o.Excludes, "excludes", []string{}, "exclude files for diff coverage calucation")
	cmd.Flags().StringVarP(&o.OutputDir, "outputdir", "o", o.OutputDir, "diff coverage output directory")
	cmd.Flags().Float64Var(&o.CoverageBaseline, "coverage-baseline", o.CoverageBaseline, "returns an error code if coverage or quality score is less than coverage baseline")
	cmd.Flags().StringVar(&o.ReportName, "report-name", "coverage", "diff coverage report name")
	cmd.Flags().StringVar(&o.Style, "style", "colorful", "coverage report code format style, refer to https://pygments.org/docs/styles for more information")
	cmd.Flags().StringVar(&o.GitHash, "gitHash", "", "gitHash for generating working source code links")
	cmd.Flags().StringVar((*string)(&o.CoverageMode), "coverage-mode", string(gocover.FullCoverage), `mode for coverage, "full" or "diff"`)
	cmd.Flags().StringVar((*string)(&o.ExecutorMode), "executor-mode", string(gocover.GoExecutor), `unit test mode, "go" or "ginkgo"`)
	cmd.Flags().StringSliceVar(&o.GinkgoFlags, "ginkgo-flags", []string{"-r", "-trace", "-cover", "-coverpkg=./..."}, "ginkgo flags")
	cmd.Flags().StringSliceVar(&o.GoFlags, "go-flags", []string{}, "go flags")
	return cmd
}
