//go:build unix

package linuxsystem

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func ownershipFromInfo(t *testing.T, info os.FileInfo, path string) (int, int) {
	t.Helper()
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("unsupported file info for %q", path)
	}
	return int(stat.Uid), int(stat.Gid)
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	content := []byte("first line\nsecond line")
	if err := os.WriteFile(src, content, 0o666); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	if err := os.Chmod(src, 0o666); err != nil {
		t.Fatalf("failed to set source file mode: %v", err)
	}

	dst := filepath.Join(tmpDir, "dest.txt")

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile returned error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("destination file content mismatch: got %q, want %q", string(got), string(content))
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		t.Fatalf("failed to stat source file: %v", err)
	}
	dstInfo, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("failed to stat destination file: %v", err)
	}

	if !dstInfo.Mode().IsRegular() {
		t.Fatalf("destination is not a regular file, mode: %v", dstInfo.Mode())
	}
	if dstInfo.Mode().Perm() != srcInfo.Mode().Perm() {
		t.Fatalf("destination permissions %v, want %v", dstInfo.Mode().Perm(), srcInfo.Mode().Perm())
	}

	srcUID, srcGID := ownershipFromInfo(t, srcInfo, src)
	dstUID, dstGID := ownershipFromInfo(t, dstInfo, dst)
	if srcUID != dstUID {
		t.Fatalf("destination uid %d, want %d", dstUID, srcUID)
	}
	if srcGID != dstGID {
		t.Fatalf("destination gid %d, want %d", dstGID, srcGID)
	}
}

func TestCopyDir(t *testing.T) {
	srcRoot := t.TempDir()

	if err := os.Chmod(srcRoot, 0o775); err != nil {
		t.Fatalf("failed to set root directory mode: %v", err)
	}

	// Build a small directory tree with explicit permissions.
	topFile := filepath.Join(srcRoot, "top.txt")
	if err := os.WriteFile(topFile, []byte("top-level"), 0o666); err != nil {
		t.Fatalf("failed to create top-level file: %v", err)
	}
	if err := os.Chmod(topFile, 0o666); err != nil {
		t.Fatalf("failed to set top-level file mode: %v", err)
	}

	nestedDir := filepath.Join(srcRoot, "nested", "deeper")
	if err := os.MkdirAll(nestedDir, 0o777); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}
	if err := os.Chmod(filepath.Join(srcRoot, "nested"), 0o773); err != nil {
		t.Fatalf("failed to set nested directory mode: %v", err)
	}
	if err := os.Chmod(nestedDir, 0o771); err != nil {
		t.Fatalf("failed to set deeper directory mode: %v", err)
	}

	nestedFile := filepath.Join(nestedDir, "data.txt")
	nestedContent := []byte("nested contents\nacross lines")
	if err := os.WriteFile(nestedFile, nestedContent, 0o662); err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}
	if err := os.Chmod(nestedFile, 0o662); err != nil {
		t.Fatalf("failed to set nested file mode: %v", err)
	}

	dstRoot := filepath.Join(t.TempDir(), "copy")

	if err := CopyDir(srcRoot, dstRoot); err != nil {
		t.Fatalf("CopyDir returned error: %v", err)
	}

	// Verify directories exist with matching permissions.
	dirChecks := []struct {
		src string
		dst string
	}{
		{src: srcRoot, dst: dstRoot},
		{src: filepath.Join(srcRoot, "nested"), dst: filepath.Join(dstRoot, "nested")},
		{src: filepath.Join(srcRoot, "nested", "deeper"), dst: filepath.Join(dstRoot, "nested", "deeper")},
	}

	for _, check := range dirChecks {
		srcInfo, err := os.Stat(check.src)
		if err != nil {
			t.Fatalf("failed to stat source directory %q: %v", check.src, err)
		}
		dstInfo, err := os.Stat(check.dst)
		if err != nil {
			t.Fatalf("failed to stat destination directory %q: %v", check.dst, err)
		}
		if !dstInfo.IsDir() {
			t.Fatalf("destination %q is not a directory", check.dst)
		}
		if dstInfo.Mode().Perm() != srcInfo.Mode().Perm() {
			t.Fatalf("permissions mismatch for directory %q: got %v, want %v", check.dst, dstInfo.Mode().Perm(), srcInfo.Mode().Perm())
		}

		srcUID, srcGID := ownershipFromInfo(t, srcInfo, check.src)
		dstUID, dstGID := ownershipFromInfo(t, dstInfo, check.dst)
		if srcUID != dstUID {
			t.Fatalf("uid mismatch for directory %q: got %d, want %d", check.dst, dstUID, srcUID)
		}
		if srcGID != dstGID {
			t.Fatalf("gid mismatch for directory %q: got %d, want %d", check.dst, dstGID, srcGID)
		}
	}

	// Verify files are copied with identical content and permissions.
	fileChecks := []struct {
		src string
		dst string
	}{
		{src: topFile, dst: filepath.Join(dstRoot, "top.txt")},
		{src: nestedFile, dst: filepath.Join(dstRoot, "nested", "deeper", "data.txt")},
	}

	for _, check := range fileChecks {
		srcInfo, err := os.Stat(check.src)
		if err != nil {
			t.Fatalf("failed to stat source file %q: %v", check.src, err)
		}
		dstInfo, err := os.Stat(check.dst)
		if err != nil {
			t.Fatalf("failed to stat destination file %q: %v", check.dst, err)
		}
		if !dstInfo.Mode().IsRegular() {
			t.Fatalf("destination %q is not a regular file", check.dst)
		}
		if dstInfo.Mode().Perm() != srcInfo.Mode().Perm() {
			t.Fatalf("permissions mismatch for file %q: got %v, want %v", check.dst, dstInfo.Mode().Perm(), srcInfo.Mode().Perm())
		}

		srcUID, srcGID := ownershipFromInfo(t, srcInfo, check.src)
		dstUID, dstGID := ownershipFromInfo(t, dstInfo, check.dst)
		if srcUID != dstUID {
			t.Fatalf("uid mismatch for file %q: got %d, want %d", check.dst, dstUID, srcUID)
		}
		if srcGID != dstGID {
			t.Fatalf("gid mismatch for file %q: got %d, want %d", check.dst, dstGID, srcGID)
		}

		srcContent, err := os.ReadFile(check.src)
		if err != nil {
			t.Fatalf("failed to read source file %q: %v", check.src, err)
		}
		dstContent, err := os.ReadFile(check.dst)
		if err != nil {
			t.Fatalf("failed to read destination file %q: %v", check.dst, err)
		}
		if string(dstContent) != string(srcContent) {
			t.Fatalf("content mismatch for %q: got %q, want %q", check.dst, string(dstContent), string(srcContent))
		}
	}
}
