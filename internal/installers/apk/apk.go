package apk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devcontainer-community/nanolayer-go/internal/linuxsystem"
)

func isAlpine() bool {
	return linuxsystem.Alpine == linuxsystem.GetDistribution()
}

func InstallPackage(pkg []string) error {
	if !isAlpine() {
		return fmt.Errorf("error: Command only supported on Alpine Linux")
	}

	if len(pkg) == 0 {
		return fmt.Errorf("error: No packages specified")
	}

	// Create temporary directory and copy /var/cache/apk to it (using native Go)
	tmpDir, err := os.MkdirTemp("", "apk-cache-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir) // Clean up temp directory when done

	cachePath := "/var/cache/apk"
	// Check if cache directory exists
	if _, err := os.Stat(cachePath); err == nil {
		// Copy cache directory to temp location
		err = linuxsystem.CopyDir(cachePath, filepath.Join(tmpDir, "apk"))
		if err != nil {
			return fmt.Errorf("failed to backup APK cache: %w", err)
		}
	}

	// Build the command: apk add --no-cache <packages>
	args := append([]string{"add", "--no-cache"}, pkg...)
	cmd := exec.Command("apk", args...)

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install packages %s: %w\nOutput: %s",
			strings.Join(pkg, ", "), err, string(output))
	}

	cleanUp()
	// Restore the original APK cache
	if _, err := os.Stat(cachePath); err == nil {
		// Copy back the cache from temp location
		err = linuxsystem.CopyDir(filepath.Join(tmpDir, "apk"), cachePath)
		if err != nil {
			return fmt.Errorf("failed to restore APK cache: %w", err)
		}
	}

	fmt.Printf("Successfully installed: %s\n", strings.Join(pkg, ", "))
	return nil
}

func cleanUp() error {
	if !isAlpine() {
		return fmt.Errorf("error: Command only supported on Alpine Linux")
	}

	// Remove the APK cache directory
	cachePath := "/var/cache/apk"
	err := os.RemoveAll(cachePath)
	if err != nil {
		return fmt.Errorf("error: Failed to clean up APK cache: %w", err)
	}

	fmt.Println("Successfully cleaned up APK cache")
	return nil
}
