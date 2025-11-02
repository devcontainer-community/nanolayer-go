package system

import (
	"fmt"

	"github.com/devcontainer-community/nanolayer-go/internal"
	"github.com/devcontainer-community/nanolayer-go/internal/linuxsystem"
	"github.com/spf13/cobra"
)

var SystemCmd = &cobra.Command{
	Use:   "system",
	Short: "System-related commands",
	Long:  `Commands for system information and operations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("IsLinux=%t\n", linuxsystem.IsLinux())
		fmt.Printf("Architecture=%s\n", linuxsystem.GetArchitecture())
		fmt.Printf("Distribution=%s\n", linuxsystem.GetDistribution())
		fmt.Printf("HasRootPrivileges=%t\n", linuxsystem.HasRootPrivileges())
		fmt.Printf("NanolayerVersion=%s\n", internal.Version)
		fmt.Printf("NanolayerCommit=%s\n", internal.Commit)
		fmt.Printf("NanolayerDate=%s\n", internal.Date)
	},
}

func init() {
	// Add subcommands here if needed
}
