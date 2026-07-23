package service

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileSize returns the size in bytes of the regular file at path.
func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("stat file %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("path %q is not a regular file", path)
	}
	return info.Size(), nil
}

// DirectorySize returns the total size in bytes of regular files below root.
// Symbolic links are not followed.
func DirectorySize(root string) (int64, error) {
	var size int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %q: %w", path, err)
		}
		size += info.Size()
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("measure directory %q: %w", root, err)
	}
	return size, nil
}
