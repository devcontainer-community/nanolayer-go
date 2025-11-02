package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/devcontainer-community/nanolayer-go/internal/linuxsystem"
)

type Release struct {
	TagName      string `json:"tag_name"`
	IsPreRelease bool   `json:"prerelease"`
}

func GetGitHubReleases(githubRepo string, allPages bool) ([]Release, error) {
	// use github api to get list of releases, ordered by release date
	// use GITHUB_TOKEN env var if available to increase rate limit

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", githubRepo)

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	// Set Accept header for GitHub API
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Add pagination parameter if allPages is false
	if !allPages {
		q := req.URL.Query()
		q.Add("per_page", "30")
		req.URL.RawQuery = q.Encode()
	}

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var releases []Release
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	// remove leading v from tag names
	for i, release := range releases {
		releases[i].TagName = strings.TrimPrefix(release.TagName, "v")
	}

	return releases, nil
}

func GetLatestRelease(githubRepo string, includePreReleases bool) (*Release, error) {
	releases, err := GetGitHubReleases(githubRepo, false)
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		if !release.IsPreRelease || includePreReleases {
			return &release, nil
		}
	}

	return nil, fmt.Errorf("no suitable release found")
}

func GetGitHubReleaseAsset(githubRepo string,
	version string,
	urlTemplate string,
	templateValues map[string]string) (string, error) {

	if version == "latest" {
		latestRelease, err := GetLatestRelease(githubRepo, false)
		if err != nil {
			return "", err
		}
		version = latestRelease.TagName
	}

	// get a release asset URL by replacing template values in the urlTemplate
	templateValues["Version"] = version
	assetURL := urlTemplate
	for key, value := range templateValues {
		placeholder := fmt.Sprintf("${%s}", key)
		assetURL = strings.ReplaceAll(assetURL, placeholder, value)
	}

	// return the final asset URL
	// check that the assetURL is valid
	if assetURL == "" {
		return "", fmt.Errorf("failed to generate asset URL")
	}
	// check if the assetURL is reachable
	resp, err := http.Head(assetURL)
	if err != nil {
		return "", fmt.Errorf("failed to reach asset URL: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("asset %s URL returned status %d", assetURL, resp.StatusCode)
	}
	return assetURL, nil
}

func DownloadAndInstallFromAssetUrl(repo string,
	version string,
	assetName string,
	assetUrlTemplate string,
	architectureReplacements map[string]string,
	fileDestinations map[string]string) error {

	architecture := string(linuxsystem.GetArchitecture())
	fmt.Printf("Detected architecture: %s\n", architecture)
	if replacement, ok := architectureReplacements[architecture]; ok {
		architecture = replacement
	}
	fmt.Printf("Using architecture: %s\n", architecture)
	assetURL, err := GetGitHubReleaseAsset(repo, version, assetUrlTemplate, map[string]string{
		"Repo":         repo,
		"Version":      version,
		"Architecture": architecture,
		"AssetName":    assetName,
	})
	if err != nil {
		return fmt.Errorf("failed to get asset URL: %w", err)
	}

	fmt.Printf("Using asset URL %s", assetURL)

	// Download the asset
	resp, err := http.Get(assetURL)
	if err != nil {
		return fmt.Errorf("failed to download asset from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("asset URL returned status %d", resp.StatusCode)
	}

	// Read the entire body into memory
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read asset body: %w", err)
	}

	// Detect archive type
	archiveType := detectArchiveType(assetURL, bodyBytes)
	fmt.Printf("Detected archive type: %s\n", archiveType)

	// Extract and list files
	files, err := extractArchive(archiveType, bodyBytes)
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// List the files
	fmt.Println("Files in archive:")
	for _, file := range files {
		if file.IsDir {
			fmt.Printf("  %s (directory)\n", file.Name)
		} else {
			fmt.Printf("  %s (%d bytes)\n", file.Name, len(file.Content))
			// Check if there is a destination path for this file
			for srcFileName, destPath := range fileDestinations {
				match := false
				if matched, _ := filepath.Match(srcFileName, file.Name); matched {
					match = true
				}
				if match {

					// Create the destination directory if it doesn't exist
					err := os.MkdirAll(filepath.Dir(destPath), 0755)
					if err != nil {
						return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
					}

					// Write the file to the destination path
					err = os.WriteFile(destPath, file.Content, 0755)
					if err != nil {
						return fmt.Errorf("failed to write file %s to %s: %w", file.Name, destPath, err)
					}
					fmt.Printf("Installed %s to %s\n", file.Name, destPath)
				}
			}
		}
	}
	return nil
}
