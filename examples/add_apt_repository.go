package main

import (
	"fmt"
	"log"

	aptget "github.com/devcontainer-community/nanolayer-go/internal/installers/apt_get"
)

// Example usage of AddAptRepository function
func main() {
	// Example 1: Add Cloudflare repository with keyring
	fmt.Println("Example 1: Adding Cloudflare repository with keyring...")
	err := aptget.AddAptRepository(
		"https://pkg.cloudflareclient.com/",                                      // repo URL
		"/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg",               // keyring path
		"jammy",                                                                  // distribution (Ubuntu 22.04 LTS)
		"main",                                                                   // component
		"/etc/apt/sources.list.d/cloudflare-client.list",                       // destination file
	)
	if err != nil {
		log.Printf("Error adding Cloudflare repository: %v", err)
	} else {
		fmt.Println("✓ Successfully added Cloudflare repository")
	}

	// Example 2: Add NodeSource repository without explicit keyring
	fmt.Println("\nExample 2: Adding NodeSource repository without keyring...")
	err = aptget.AddAptRepository(
		"https://deb.nodesource.com/node_20.x",                                 // repo URL
		"",                                                                      // no keyring
		"jammy",                                                                 // distribution
		"main",                                                                  // component
		"/etc/apt/sources.list.d/nodesource.list",                             // destination file
	)
	if err != nil {
		log.Printf("Error adding NodeSource repository: %v", err)
	} else {
		fmt.Println("✓ Successfully added NodeSource repository")
	}

	// Example 3: Add repository with automatic distribution detection
	fmt.Println("\nExample 3: Adding repository with auto-detected distribution...")
	err = aptget.AddAptRepository(
		"https://packages.microsoft.com/repos/code/",                           // repo URL
		"/usr/share/keyrings/packages.microsoft.gpg",                          // keyring path
		"",                                                                      // empty - will auto-detect distribution
		"main",                                                                  // component
		"/etc/apt/sources.list.d/vscode.list",                                 // destination file
	)
	if err != nil {
		log.Printf("Error adding VS Code repository: %v", err)
	} else {
		fmt.Println("✓ Successfully added VS Code repository")
	}
}