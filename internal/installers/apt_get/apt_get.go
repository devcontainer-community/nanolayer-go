package aptget

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devcontainer-community/nanolayer-go/internal/linuxsystem"
)

func isDebianLike() bool {
	return linuxsystem.Debian == linuxsystem.GetDistribution() ||
		linuxsystem.Ubuntu == linuxsystem.GetDistribution() ||
		linuxsystem.Raspbian == linuxsystem.GetDistribution()
}

func InstallPackage(pkg []string) error {
	if !isDebianLike() {
		return fmt.Errorf("error: Command only supported on Debian-based distributions")
	}
	// Create temporary directory and copy /var/lib/apt/lists to it (using native Go)
	tmpDir, err := os.MkdirTemp("", "apt-cache-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir) // Clean up temp directory when done

	cachePath := "/var/lib/apt/lists"
	// Check if cache directory exists
	if _, err := os.Stat(cachePath); err == nil {
		// Copy cache directory to temp location
		err = linuxsystem.CopyDir(cachePath, filepath.Join(tmpDir, "apt"))
		if err != nil {
			return fmt.Errorf("failed to backup APT cache: %w", err)
		}
	}

	UpdatePackageLists()
	// Build the command: apt-get install --no-cache <packages>
	args := append([]string{"install", "--no-cache"}, pkg...)
	cmd := exec.Command("apt-get", args...)

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install packages %s: %w\nOutput: %s",
			strings.Join(pkg, ", "), err, string(output))
	}

	CleanUp()
	// Restore the original APT cache
	if _, err := os.Stat(cachePath); err == nil {
		// Copy back the cache from temp location
		err = linuxsystem.CopyDir(filepath.Join(tmpDir, "apt"), cachePath)
		if err != nil {
			return fmt.Errorf("failed to restore APT cache: %w", err)
		}
	}

	return nil
}

func CleanUp() error {
	// Implementation for cleaning up apt-get caches
	return nil
}

func AddPpaRepository(ppa string) error {
	// Implementation for adding a PPA repository
	return nil
}

func AddAptRepository(repo string) error {
	// Implementation for adding a custom apt repository
	return nil
}

func UpdatePackageLists() error {
	// Implementation for updating apt package lists
	return nil
}

func RunUpgrade() error {
	// Implementation for running apt-get upgrade
	return nil
}

func RemovePackage(pkg string) error {
	// Implementation for removing a package using apt-get
	return nil
}

func IsPackageInstalled(pkg string) (bool, error) {
	// Implementation for checking if a package is installed
	return false, nil
}
