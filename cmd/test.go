package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/devcontainer-community/nanolayer-go/internal/installers/github"
	"github.com/devcontainer-community/nanolayer-go/internal/linuxsystem"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run a test command",
	Long:  `This is a test command to verify the CLI is working correctly.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Test command executed successfully!")
		if len(args) > 0 {
			fmt.Printf("Arguments: %v\n", args)
		}
		fmt.Printf("architecture: %s\n", linuxsystem.GetArchitecture())
		fmt.Printf("is linux: %t\n", linuxsystem.IsLinux())
		// repo := "stelviodev/stelvio"
		repo := "charmbracelet/gum"
		versions, err := github.GetGitHubReleases(repo, false)
		if err != nil {
			fmt.Printf("Error fetching versions: %v\n", err)
		} else {
			fmt.Printf("versions: %v\n", versions)
		}
		latestRelease, err := github.GetLatestRelease(repo, false)
		if err != nil {
			fmt.Printf("Error fetching latest release: %v\n", err)
		} else {
			fmt.Printf("latest release: %v\n", latestRelease)
		}
		assetURL, err := github.GetGitHubReleaseAsset(repo, "latest",
			"https://github.com/${Repo}/releases/download/v${Version}/${AssetName}_${Version}_Linux_${Architecture}.tar.gz", map[string]string{
				"Repo":         repo,
				"Version":      latestRelease.TagName,
				"AssetName":    "gum",
				"Architecture": "x86_64",
			})
		if err != nil {
			fmt.Printf("Error fetching asset URL: %v\n", err)
		} else {
			fmt.Printf("Asset URL: %v\n", assetURL)
		}
		err = github.DownloadAndInstallFromAssetUrl(repo,
			"latest",
			"gum",
			"https://github.com/${Repo}/releases/download/v${Version}/${AssetName}_${Version}_Linux_${Architecture}.tar.gz",
			map[string]string{
				// "arm64": "aarch64",
			},
			map[string]string{
				"*/gum": "/tmp/usr/local/bin/gum",
			})
		if err != nil {
			fmt.Printf("Error downloading and installing from asset URL: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
