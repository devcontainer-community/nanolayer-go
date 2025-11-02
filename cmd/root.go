package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/devcontainer-community/nanolayer-go/cmd/install"
	"github.com/devcontainer-community/nanolayer-go/cmd/system"
)

var rootCmd = &cobra.Command{
	Use:   "nanolayer",
	Short: "Nanolayer - A developer container tool",
	Long:  `Nanolayer is a tool for working with development containers.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check for version flag
		versionFlag, _ := cmd.Flags().GetBool("version")
		if versionFlag {
			fmt.Printf("nanolayer version %s\n", Version)
			fmt.Printf("commit: %s\n", Commit)
			fmt.Printf("built at: %s\n", Date)
			return
		}
		// If no subcommand is provided, show help
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
	rootCmd.AddCommand(install.InstallCmd)
	rootCmd.AddCommand(system.SystemCmd)
}
