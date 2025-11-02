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

	hiddenFile := filepath.Join(srcRoot, ".hidden-config")
	hiddenContent := []byte("hidden values")
	if err := os.WriteFile(hiddenFile, hiddenContent, 0o640); err != nil {
		t.Fatalf("failed to create hidden file: %v", err)
	}
	if err := os.Chmod(hiddenFile, 0o640); err != nil {
		t.Fatalf("failed to set hidden file mode: %v", err)
	}

	hiddenDir := filepath.Join(srcRoot, ".hidden", "configs")
	if err := os.MkdirAll(hiddenDir, 0o765); err != nil {
		t.Fatalf("failed to create hidden directory tree: %v", err)
	}
	if err := os.Chmod(filepath.Join(srcRoot, ".hidden"), 0o765); err != nil {
		t.Fatalf("failed to set hidden directory mode: %v", err)
	}
	if err := os.Chmod(hiddenDir, 0o754); err != nil {
		t.Fatalf("failed to set hidden subdirectory mode: %v", err)
	}

	hiddenDirFile := filepath.Join(hiddenDir, "config.yml")
	hiddenDirContent := []byte("secret: true\n")
	if err := os.WriteFile(hiddenDirFile, hiddenDirContent, 0o660); err != nil {
		t.Fatalf("failed to create hidden dir file: %v", err)
	}
	if err := os.Chmod(hiddenDirFile, 0o660); err != nil {
		t.Fatalf("failed to set hidden dir file mode: %v", err)
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

	symlinkToTop := filepath.Join(srcRoot, "link-to-top")
	if err := os.Symlink("top.txt", symlinkToTop); err != nil {
		t.Fatalf("failed to create symlink to top file: %v", err)
	}

	symlinkToHidden := filepath.Join(srcRoot, "link-to-hidden")
	if err := os.Symlink(".hidden", symlinkToHidden); err != nil {
		t.Fatalf("failed to create symlink to hidden directory: %v", err)
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
		{src: filepath.Join(srcRoot, ".hidden"), dst: filepath.Join(dstRoot, ".hidden")},
		{src: filepath.Join(srcRoot, ".hidden", "configs"), dst: filepath.Join(dstRoot, ".hidden", "configs")},
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
		{src: hiddenFile, dst: filepath.Join(dstRoot, ".hidden-config")},
		{src: hiddenDirFile, dst: filepath.Join(dstRoot, ".hidden", "configs", "config.yml")},
		{src: nestedFile, dst: filepath.Join(dstRoot, "nested", "deeper", "data.txt")},
	}

	symlinkChecks := []struct {
		src string
		dst string
	}{
		{src: symlinkToTop, dst: filepath.Join(dstRoot, "link-to-top")},
		{src: symlinkToHidden, dst: filepath.Join(dstRoot, "link-to-hidden")},
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

	for _, check := range symlinkChecks {
		srcInfo, err := os.Lstat(check.src)
		if err != nil {
			t.Fatalf("failed to lstat source symlink %q: %v", check.src, err)
		}
		dstInfo, err := os.Lstat(check.dst)
		if err != nil {
			t.Fatalf("failed to lstat destination symlink %q: %v", check.dst, err)
		}
		if srcInfo.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("source %q is not a symlink", check.src)
		}
		if dstInfo.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("destination %q is not a symlink", check.dst)
		}

		srcTarget, err := os.Readlink(check.src)
		if err != nil {
			t.Fatalf("failed to read source symlink %q: %v", check.src, err)
		}
		dstTarget, err := os.Readlink(check.dst)
		if err != nil {
			t.Fatalf("failed to read destination symlink %q: %v", check.dst, err)
		}
		if srcTarget != dstTarget {
			t.Fatalf("symlink target mismatch for %q: got %q, want %q", check.dst, dstTarget, srcTarget)
		}

		srcUID, srcGID := ownershipFromInfo(t, srcInfo, check.src)
		dstUID, dstGID := ownershipFromInfo(t, dstInfo, check.dst)
		if srcUID != dstUID {
			t.Fatalf("uid mismatch for symlink %q: got %d, want %d", check.dst, dstUID, srcUID)
		}
		if srcGID != dstGID {
			t.Fatalf("gid mismatch for symlink %q: got %d, want %d", check.dst, dstGID, srcGID)
		}
	}
}

func TestCopyDirEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	srcRoot := filepath.Join(tmpDir, "empty-src")
	if err := os.Mkdir(srcRoot, 0o751); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}
	if err := os.Chmod(srcRoot, 0o751); err != nil {
		t.Fatalf("failed to set source directory mode: %v", err)
	}

	dstRoot := filepath.Join(tmpDir, "empty-dst")

	if err := CopyDir(srcRoot, dstRoot); err != nil {
		t.Fatalf("CopyDir returned error for empty directory: %v", err)
	}

	dstInfo, err := os.Stat(dstRoot)
	if err != nil {
		t.Fatalf("failed to stat destination directory: %v", err)
	}
	if !dstInfo.IsDir() {
		t.Fatalf("destination %q is not a directory", dstRoot)
	}

	srcInfo, err := os.Stat(srcRoot)
	if err != nil {
		t.Fatalf("failed to stat source directory: %v", err)
	}

	if dstInfo.Mode().Perm() != srcInfo.Mode().Perm() {
		t.Fatalf("permissions mismatch for empty directory copy: got %v, want %v", dstInfo.Mode().Perm(), srcInfo.Mode().Perm())
	}

	srcUID, srcGID := ownershipFromInfo(t, srcInfo, srcRoot)
	dstUID, dstGID := ownershipFromInfo(t, dstInfo, dstRoot)
	if srcUID != dstUID {
		t.Fatalf("uid mismatch for empty directory copy: got %d, want %d", dstUID, srcUID)
	}
	if srcGID != dstGID {
		t.Fatalf("gid mismatch for empty directory copy: got %d, want %d", dstGID, srcGID)
	}

	entries, err := os.ReadDir(dstRoot)
	if err != nil {
		t.Fatalf("failed to read destination directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected destination directory to be empty, found %d entries", len(entries))
	}
}
