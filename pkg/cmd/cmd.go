package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// TODO: improve this description.
var (
	getLong = `
Generate unit test diff coverage for go code.

Use this tool to generate diff coverage for go code between two branch.
In this way to make sure that new changes has a expected test coverage, and won't affect old codes.
`

	getExample = `# Generate diff coverage report in HTML format, and failure rate should be less than 20%
gocover --cover-profile=coverage.out --compare-branch=origin/master --format html --failure-rate 20.0 --output coverage.html
`
)

// NewGoCoverCommand creates a command object for generating diff coverage reporter.
func NewGoCoverCommand() *cobra.Command {
	o := NewDiffOptions()

	cmd := &cobra.Command{
		Use:     "gocover",
		Short:   "Generate unit test diff coverage for go code",
		Long:    getLong,
		Example: getExample,
		RunE: func(cmd *cobra.Command, args []string) error {
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
	cmd.Flags().Float64Var(&o.FailureRate, "failure-rate", o.FailureRate, "returns an error code if coverage or quality score is above failure rate")
	cmd.Flags().StringVar(&o.ReportName, "report-name", "coverage", "diff coverage report name")
	cmd.Flags().StringVar(&o.Style, "style", "colorful", "coverage report code format style, refer to https://pygments.org/docs/styles for more information")

	cmd.MarkFlagRequired("cover-profile")

	return cmd
}
