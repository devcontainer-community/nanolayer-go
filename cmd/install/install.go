package install

import (
	"github.com/spf13/cobra"

	"github.com/devcontainer-community/nanolayer-go/cmd/install/github"

	"github.com/devcontainer-community/feature-installer/cmd/feature/install"
)

var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install packages and tools",
	Long:  `Install various packages and tools from different sources.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, show help
		cmd.Help()
	},
}

func init() {
	// Add subcommands here
	InstallCmd.AddCommand(github.GithubCmd)

	// Rename the devcontainer feature install command
	devcontainerFeatureCmd := install.InstallCmd
	devcontainerFeatureCmd.Use = "devcontainer-feature"
	InstallCmd.AddCommand(devcontainerFeatureCmd)
}
