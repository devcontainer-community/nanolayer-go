package github

import (
	"fmt"
	"os"
	"strings"

	"github.com/devcontainer-community/nanolayer-go/internal/installers/github"
	"github.com/spf13/cobra"
)

var GithubCmd = &cobra.Command{
	Use:   "github",
	Short: "Install packages from GitHub releases",
	Long:  `Install packages and tools from GitHub releases.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("GitHub install command executed!")
		if len(args) > 0 {
			fmt.Printf("Arguments: %v\n", args)
		}
		// Add your GitHub installation logic here
		if len(args) < 1 {
			fmt.Println("Please provide a GitHub repository in the format 'owner/repo'.")
			os.Exit(1)
		}
		repo := args[0]
		fmt.Printf("Installing from GitHub repository: %s\n", repo)

		parts := strings.Split(repo, "/")
		if len(parts) != 2 {
			fmt.Println("Error: Repository must be in the format 'owner/repo'.")
			return
		}
		assetName := parts[1]
		if flagAssetName, _ := cmd.Flags().GetString("asset-name"); flagAssetName != "" {
			assetName = flagAssetName
		}
		fmt.Printf("Using asset name: %s\n", assetName)

		version := "latest"
		if flagVersion, _ := cmd.Flags().GetString("asset-version"); flagVersion != "" {
			version = flagVersion
		}
		fmt.Printf("Using version: %s\n", version)

		assetUrlTemplate, _ := cmd.Flags().GetString("asset-url-template")
		if assetUrlTemplate == "" {
			assetUrlTemplate = "https://github.com/${Repo}/releases/download/v${Version}/${AssetName}_${Version}_lLinux_${Architecture}.tar.gz"
		}
		fmt.Printf("Using asset URL template: %s\n", assetUrlTemplate)

		// Parse architecture replacements
		architectureReplacements := make(map[string]string)
		archReplacementPairs, _ := cmd.Flags().GetStringArray("architecture-replacement")
		for _, pair := range archReplacementPairs {
			parts := strings.Fields(pair)
			if len(parts) == 2 {
				architectureReplacements[parts[0]] = parts[1]
			}
		}
		if len(architectureReplacements) > 0 {
			fmt.Printf("Architecture replacements: %v\n", architectureReplacements)
		}

		// Parse file destinations
		fileDestinations := make(map[string]string)
		fileDestPairs, _ := cmd.Flags().GetStringArray("file-destination")
		if len(fileDestPairs) > 0 {
			for _, pair := range fileDestPairs {
				parts := strings.Fields(pair)
				if len(parts) == 2 {
					fileDestinations[parts[0]] = parts[1]
				}
			}
		} else {
			// Use default if no file destinations provided
			fileDestinations[fmt.Sprintf("*/%s", assetName)] = fmt.Sprintf("/usr/local/bin/%s", assetName)
		}
		if len(fileDestinations) > 0 {
			fmt.Printf("File destinations: %v\n", fileDestinations)
		}

		err := github.DownloadAndInstallFromAssetUrl(repo,
			version,
			assetName,
			assetUrlTemplate,
			architectureReplacements,
			fileDestinations)
		if err != nil {
			fmt.Printf("Error during installation: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Installation completed successfully!")

	},
}

func init() {
	GithubCmd.Flags().String("asset-url-template", "", "Custom asset URL template (e.g., https://github.com/${Repo}/releases/download/v${Version}/${AssetName}_${Version}_Linux_${Architecture}.tar.gz)")
	GithubCmd.Flags().String("asset-name", "", "Override the asset name derived from the repository (e.g., --asset-name gum)")
	GithubCmd.Flags().String("asset-version", "", "Override the version used when fetching the asset (e.g., --asset-version 1.10.3)")
	GithubCmd.Flags().StringArray("architecture-replacement", []string{}, "Architecture replacement pairs (e.g., --architecture-replacement 'arm64 aarch64' --architecture-replacement 'amd64 intel')")
	GithubCmd.Flags().StringArray("file-destination", []string{}, "File destination mappings (e.g., --file-destination '*/gum /usr/local/bin/gum')")
}
