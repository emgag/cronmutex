package cmd

import (
	"fmt"

	"github.com/emgag/cronmutex/internal/lib/version"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of cronmutex",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cronmutex %s -- %s\n", version.Version, version.Commit)
	},
}
