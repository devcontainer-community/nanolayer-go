package cmd

import (
	"fmt"

	"github.com/devcontainer-community/nanolayer-go/internal"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Display the version, commit hash, and build date of nanolayer.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("nanolayer version %s\n", internal.Version)
		fmt.Printf("commit: %s\n", internal.Commit)
		fmt.Printf("built at: %s\n", internal.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Print version information")
}
