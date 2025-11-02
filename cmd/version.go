package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version information - set via ldflags at build time
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Display the version, commit hash, and build date of nanolayer.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("nanolayer version %s\n", Version)
		fmt.Printf("commit: %s\n", Commit)
		fmt.Printf("built at: %s\n", Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Print version information")
}
