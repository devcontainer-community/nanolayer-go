//go:build unix

package linuxsystem

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func setOwnership(path string, info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}

	uid := int(stat.Uid)
	gid := int(stat.Gid)

	var err error
	if info.Mode()&os.ModeSymlink != 0 {
		err = os.Lchown(path, uid, gid)
	} else {
		err = os.Chown(path, uid, gid)
	}

	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.ENOTSUP) {
			var current os.FileInfo
			var statErr error
			if info.Mode()&os.ModeSymlink != 0 {
				current, statErr = os.Lstat(path)
			} else {
				current, statErr = os.Stat(path)
			}
			if statErr != nil {
				return statErr
			}
			if cur, ok := current.Sys().(*syscall.Stat_t); ok {
				if int(cur.Uid) == uid && int(cur.Gid) == gid {
					return nil
				}
			}
		}
		return err
	}

	return nil
}

func copySymlink(src, dst string, info os.FileInfo) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}

	if err := os.Symlink(target, dst); err != nil {
		return err
	}

	return setOwnership(dst, info)
}

// CopyDir recursively copies a directory from src to dst
func CopyDir(src, dst string) error {
	// Get properties of source directory
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return copyDirWithInfo(src, dst, srcInfo)
}

// CopyFile copies a single file from src to dst
func CopyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}

	return setOwnership(dst, srcInfo)
}

func copyDirWithInfo(src, dst string, srcInfo os.FileInfo) error {
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}

	if err := setOwnership(dst, srcInfo); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == "." || entry.Name() == ".." {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		entryInfo, err := os.Lstat(srcPath)
		if err != nil {
			return err
		}

		switch {
		case entryInfo.Mode()&os.ModeSymlink != 0:
			if err := copySymlink(srcPath, dstPath, entryInfo); err != nil {
				return err
			}
		case entryInfo.IsDir():
			if err := copyDirWithInfo(srcPath, dstPath, entryInfo); err != nil {
				return err
			}
		default:
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
