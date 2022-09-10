package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(version, commit, date string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "print Gocover's version",
		Example: "gocover version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Gocover Version %s\n", version)
			fmt.Fprintf(cmd.OutOrStdout(), "Runtime SHA: %s\n", commit)
			fmt.Fprintf(cmd.OutOrStdout(), "Created At: %s\n", date)
			return nil
		},
	}
	return cmd
}
