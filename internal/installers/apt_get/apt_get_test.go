package aptget

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devcontainer-community/nanolayer-go/internal/linuxsystem"
)

func TestAddAptRepository(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Define test parameters
	repo := "https://pkg.cloudflareclient.com/"
	keyringPath := "/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg"
	distribution := "jammy"
	component := "main"
	destination := filepath.Join(tmpDir, "cloudflare-client.list")
	
	// Call the function
	err := AddAptRepository(repo, keyringPath, distribution, component, destination)
	
	// Check for errors based on current system
	if !isDebianLike() {
		// Should return error on non-Debian systems
		if err == nil {
			t.Fatal("expected error on non-Debian system, got nil")
		}
		if !strings.Contains(err.Error(), "only supported on Debian-based distributions") {
			t.Fatalf("expected Debian error message, got: %v", err)
		}
		return // Skip the rest of the test
	}
	
	// On Debian-like systems, should succeed
	if err != nil {
		t.Fatalf("AddAptRepository failed: %v", err)
	}
	
	// Check that the file was created
	if _, err := os.Stat(destination); os.IsNotExist(err) {
		t.Fatal("repository file was not created")
	}
	
	// Read and verify the file content
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("failed to read repository file: %v", err)
	}
	
	contentStr := string(content)
	
	// Verify the content format
	if !strings.Contains(contentStr, "deb [arch=") {
		t.Fatalf("repository line does not contain expected format, got: %s", contentStr)
	}
	
	if !strings.Contains(contentStr, "signed-by="+keyringPath) {
		t.Fatalf("repository line does not contain keyring path, got: %s", contentStr)
	}
	
	if !strings.Contains(contentStr, repo) {
		t.Fatalf("repository line does not contain repository URL, got: %s", contentStr)
	}
	
	if !strings.Contains(contentStr, distribution) {
		t.Fatalf("repository line does not contain distribution, got: %s", contentStr)
	}
	
	if !strings.Contains(contentStr, component) {
		t.Fatalf("repository line does not contain component, got: %s", contentStr)
	}
	
	t.Logf("Successfully created repository file with content: %s", contentStr)
}

func TestAddAptRepositoryWithoutKeyring(t *testing.T) {
	// Skip this test on non-Debian systems
	if !isDebianLike() {
		t.Skip("skipping test on non-Debian system")
	}
	
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Define test parameters without keyring
	repo := "https://deb.nodesource.com/node_20.x"
	keyringPath := "" // Empty keyring path
	distribution := "jammy"
	component := "main"
	destination := filepath.Join(tmpDir, "nodesource.list")
	
	// Call the function
	err := AddAptRepository(repo, keyringPath, distribution, component, destination)
	if err != nil {
		t.Fatalf("AddAptRepository failed: %v", err)
	}
	
	// Read and verify the file content
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("failed to read repository file: %v", err)
	}
	
	contentStr := string(content)
	
	// Verify that it doesn't contain signed-by when keyring is empty
	if strings.Contains(contentStr, "signed-by=") {
		t.Fatalf("repository line should not contain signed-by when keyring is empty, got: %s", contentStr)
	}
	
	// But should still contain the basic format
	if !strings.Contains(contentStr, "deb [arch=") {
		t.Fatalf("repository line does not contain expected format, got: %s", contentStr)
	}
	
	t.Logf("Successfully created repository file without keyring: %s", contentStr)
}

func TestAddAptRepositoryArchitectureMapping(t *testing.T) {
	// Skip this test on non-Debian systems
	if !isDebianLike() {
		t.Skip("skipping test on non-Debian system")
	}
	
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.list")
	
	// Call the function
	err := AddAptRepository("https://example.com/", "", "jammy", "main", destination)
	if err != nil {
		t.Fatalf("AddAptRepository failed: %v", err)
	}
	
	// Read the file content
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("failed to read repository file: %v", err)
	}
	
	contentStr := string(content)
	
	// Check that the architecture is properly mapped
	architecture := linuxsystem.GetArchitecture()
	expectedArch := ""
	switch architecture {
	case linuxsystem.ARM64:
		expectedArch = "arm64"
	case linuxsystem.X86_64:
		expectedArch = "amd64"
	case linuxsystem.ARMV7:
		expectedArch = "armhf"
	case linuxsystem.I386:
		expectedArch = "i386"
	default:
		// For other architectures, we don't know what dpkg will return
		// Just verify that some architecture is present
		if !strings.Contains(contentStr, "deb [arch=") {
			t.Fatalf("repository line does not contain architecture, got: %s", contentStr)
		}
		t.Logf("Architecture %s mapped to content: %s", architecture, contentStr)
		return
	}
	
	if !strings.Contains(contentStr, "arch="+expectedArch) {
		t.Fatalf("expected architecture %s not found in repository line, got: %s", expectedArch, contentStr)
	}
	
	t.Logf("Architecture %s correctly mapped to %s", architecture, expectedArch)
}