package pathlocal

import (
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func RenameFile(oldPath string, newName string, followSymlinks bool) error {
	// Clean paths to ensure consistent formatting for the OS
	oldPath = filepath.Clean(oldPath)
	newName = filepath.Clean(newName)

	// Check if newName contains path separators, which would indicate
	// an attempt to move the file to a different directory
	if filepath.Base(newName) != newName {
		return &fs.PathError{
			Op:   "local-rename-validate",
			Path: oldPath,
			Err:  fs.ErrInvalid,
		}
	}

	// Evaluate any symbolic links to get the real path
	var pathToRename string
	if followSymlinks {
		// Get the real path by evaluating symlinks
		realPath, err := filepath.EvalSymlinks(oldPath)
		if err != nil {
			return &pathmodels.PathError{
				Op:   "local-rename-realpath",
				Path: oldPath,
				Err:  err,
			}
		}
		pathToRename = realPath
	} else {
		// Use the original path without resolving symlinks
		pathToRename = oldPath
	}

	// Get the directory of the real path
	dir := filepath.Dir(pathToRename)

	// Construct the new full path using the directory and new name
	newPath := filepath.Join(dir, newName)

	// On Windows, we need special handling for open files
	if runtime.GOOS == "windows" {
		// Try to ensure the target doesn't exist first
		// Windows is more restrictive about renaming to existing files
		_ = os.Remove(newPath) // Ignore error if file doesn't exist
	}

	// Perform the rename operation
	err := os.Rename(pathToRename, newPath)
	if err != nil {
		// On Windows, if rename fails, it might be due to file being in use
		// You might want to add retry logic here for Windows
		return &fs.PathError{Op: "local-rename", Path: oldPath, Err: err}
	}

	return nil
}
