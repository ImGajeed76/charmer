package pathlocal

import (
	"io/fs"
	"os"
	"path/filepath"
)

// List returns a list of paths for all items in the directory.
// If recursive is true, it will include paths from all subdirectories.
// Returns absolute paths by default.
func List(dirPath string, recursive bool) ([]string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, &fs.PathError{Op: "local-list-abs", Path: dirPath, Err: err}
	}

	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, &fs.PathError{Op: "local-list-stat", Path: dirPath, Err: err}
	}
	if !info.IsDir() {
		return nil, &fs.PathError{
			Op:   "local-list-check",
			Path: dirPath,
			Err:  fs.ErrInvalid,
		}
	}

	var paths []string

	if recursive {
		// Walk through all subdirectories
		err = filepath.Walk(absPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path != absPath { // Skip the root directory itself
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, &fs.PathError{Op: "local-list-walk", Path: dirPath, Err: err}
		}
	} else {
		// Read only the immediate directory
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return nil, &fs.PathError{Op: "local-list-read", Path: dirPath, Err: err}
		}

		// Convert entries to absolute paths
		for _, entry := range entries {
			paths = append(paths, filepath.Join(absPath, entry.Name()))
		}
	}

	return paths, nil
}
