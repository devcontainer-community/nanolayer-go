package aptget

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/devcontainer-community/nanolayer-go/internal/linuxsystem"
	"golang.org/x/crypto/openpgp/armor"
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
	args := append([]string{"install", "--no-install-recommends"}, pkg...)
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

func AddRepositoryKey(url string, destination string, deamor bool) error {
	// Example shell equivalence:
	// curl -fsSL https://pkg.cloudflareclient.com/pubkey.gpg | sudo gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg
	// Download the key from `url` and write it to `destination` atomically. If `deamor` is true,
	// run `gpg --dearmor --yes -o <destination>` and pass the downloaded data to gpg stdin.

	var data []byte

	// Support local file URLs
	if strings.HasPrefix(url, "file://") {
		localPath := strings.TrimPrefix(url, "file://")
		b, err := os.ReadFile(localPath)
		if err != nil {
			return fmt.Errorf("failed to read local key file %s: %w", localPath, err)
		}
		data = b
	} else {
		// Download over HTTP(S) with a timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request for %s: %w", url, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download key from %s: %w", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("failed to download key from %s: status %s", url, resp.Status)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read key body from %s: %w", url, err)
		}
		data = b
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	if deamor {
		// Try to decode ASCII-armored OpenPGP blocks in pure Go to avoid
		// requiring an external `gpg` binary. If decode succeeds, write
		// the binary data; if not, fall back to writing the raw data (it may
		// already be a binary key).
		if decoded, err := tryDearmor(data); err == nil {
			// write decoded bytes atomically
			tmpFile, err := os.CreateTemp(destDir, "apt-key-*")
			if err != nil {
				return fmt.Errorf("failed to create temp file in %s: %w", destDir, err)
			}
			tmpName := tmpFile.Name()
			defer func() {
				_ = tmpFile.Close()
				_ = os.Remove(tmpName)
			}()

			if _, err := tmpFile.Write(decoded); err != nil {
				return fmt.Errorf("failed to write dearmored key to temp file: %w", err)
			}
			if err := tmpFile.Sync(); err != nil {
				_ = tmpFile.Close()
				return fmt.Errorf("failed to sync temp key file: %w", err)
			}
			if err := tmpFile.Close(); err != nil {
				return fmt.Errorf("failed to close temp key file: %w", err)
			}

			if err := os.Rename(tmpName, destination); err != nil {
				in, err2 := os.Open(tmpName)
				if err2 != nil {
					return fmt.Errorf("failed to move dearmored key to destination: %w", err)
				}
				defer in.Close()
				out, err2 := os.Create(destination)
				if err2 != nil {
					return fmt.Errorf("failed to create destination file %s: %w", destination, err2)
				}
				if _, err2 = io.Copy(out, in); err2 != nil {
					out.Close()
					return fmt.Errorf("failed to copy dearmored key to destination: %w", err2)
				}
				if err2 = out.Close(); err2 != nil {
					return fmt.Errorf("failed to close destination file %s: %w", destination, err2)
				}
			}
			return nil
		}

		// If dearmor failed (input likely already binary), continue and write raw data below.
	}

	// Write atomically: write to a temp file in same dir then rename
	tmpFile, err := os.CreateTemp(destDir, "apt-key-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file in %s: %w", destDir, err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write key to temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to sync temp key file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp key file: %w", err)
	}

	// Try to rename into place; fallback to copy if rename fails (different FS)
	if err := os.Rename(tmpName, destination); err != nil {
		in, err2 := os.Open(tmpName)
		if err2 != nil {
			return fmt.Errorf("failed to move key to destination: %w", err)
		}
		defer in.Close()
		out, err2 := os.Create(destination)
		if err2 != nil {
			return fmt.Errorf("failed to create destination file %s: %w", destination, err2)
		}
		if _, err2 = io.Copy(out, in); err2 != nil {
			out.Close()
			return fmt.Errorf("failed to copy key to destination: %w", err2)
		}
		if err2 = out.Close(); err2 != nil {
			return fmt.Errorf("failed to close destination file %s: %w", destination, err2)
		}
	}

	return nil
} 

func CleanUp() error {
	// Implementation for cleaning up apt-get caches
	return nil
}
// tryDearmor attempts to decode ASCII-armored OpenPGP data and return the
// raw binary packet stream. If the input is not armored, an error is
// returned.
func tryDearmor(data []byte) ([]byte, error) {
	rdr := bytes.NewReader(data)
	block, err := armor.Decode(rdr)
	if err != nil {
		return nil, err
	}
	out, err := io.ReadAll(block.Body)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func AddPpaRepository(ppa string) error {
	// Implementation for adding a PPA repository
	return nil
}

// AddAptRepository adds an APT repository to the system by creating a sources.list entry.
// 
// Parameters:
//   - repo: Repository URL (e.g., "https://pkg.cloudflareclient.com/")
//   - keyringPath: Path to the GPG keyring file for repository signing (empty string to skip)
//   - distribution: Distribution codename (e.g., "jammy", "bookworm"). If empty, will attempt to detect
//   - component: Repository component (e.g., "main", "contrib", "non-free")  
//   - destination: Path where to write the sources.list file (e.g., "/etc/apt/sources.list.d/myrepo.list")
//
// Shell equivalent example:
// echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/cloudflare-client.list
func AddAptRepository(repo string, keyringPath string, distribution string, component string, destination string) error {
	
	if !isDebianLike() {
		return fmt.Errorf("error: Command only supported on Debian-based distributions")
	}

	architecture := linuxsystem.GetArchitecture()
	
	// Convert the architecture to dpkg format
	dpkgArch := ""
	switch architecture {
	case linuxsystem.ARM64:
		dpkgArch = "arm64"
	case linuxsystem.X86_64:
		dpkgArch = "amd64"
	case linuxsystem.ARMV7:
		dpkgArch = "armhf"
	case linuxsystem.I386:
		dpkgArch = "i386"
	default:
		// For other architectures, use dpkg --print-architecture
		cmd := exec.Command("dpkg", "--print-architecture")
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get architecture from dpkg: %w", err)
		}
		dpkgArch = strings.TrimSpace(string(output))
	}

	// If distribution is empty, try to get it from lsb_release or fallback
	if distribution == "" {
		// Try lsb_release first
		cmd := exec.Command("lsb_release", "-cs")
		output, err := cmd.Output()
		if err == nil {
			distribution = strings.TrimSpace(string(output))
		} else {
			// Fallback based on detected distribution
			switch linuxsystem.GetDistribution() {
			case linuxsystem.Ubuntu:
				distribution = "focal" // Default to a common Ubuntu release
			case linuxsystem.Debian:
				distribution = "bookworm" // Default to a common Debian release
			default:
				return fmt.Errorf("could not determine distribution codename and none provided")
			}
		}
	}

	// Construct the repository line
	var repoLine string
	if keyringPath != "" {
		repoLine = fmt.Sprintf("deb [arch=%s signed-by=%s] %s %s %s\n", 
			dpkgArch, keyringPath, repo, distribution, component)
	} else {
		repoLine = fmt.Sprintf("deb [arch=%s] %s %s %s\n", 
			dpkgArch, repo, distribution, component)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Write atomically: write to a temp file in same dir then rename
	tmpFile, err := os.CreateTemp(destDir, "apt-repo-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file in %s: %w", destDir, err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.WriteString(repoLine); err != nil {
		return fmt.Errorf("failed to write repository line to temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to sync temp repository file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp repository file: %w", err)
	}

	// Try to rename into place; fallback to copy if rename fails (different FS)
	if err := os.Rename(tmpName, destination); err != nil {
		in, err2 := os.Open(tmpName)
		if err2 != nil {
			return fmt.Errorf("failed to move repository file to destination: %w", err)
		}
		defer in.Close()
		out, err2 := os.Create(destination)
		if err2 != nil {
			return fmt.Errorf("failed to create destination file %s: %w", destination, err2)
		}
		if _, err2 = io.Copy(out, in); err2 != nil {
			out.Close()
			return fmt.Errorf("failed to copy repository file to destination: %w", err2)
		}
		if err2 = out.Close(); err2 != nil {
			return fmt.Errorf("failed to close destination file %s: %w", destination, err2)
		}
	}

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
