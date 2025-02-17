package pathlocal

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func RenameFile(oldPath string, newName string) error {
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
	realPath, err := filepath.EvalSymlinks(oldPath)
	if err != nil {
		return &fs.PathError{Op: "local-rename-eval-symlinks", Path: oldPath, Err: err}
	}

	// Get the directory of the real path
	dir := filepath.Dir(realPath)

	// Construct the new full path using the directory and new name
	newPath := filepath.Join(dir, newName)

	// On Windows, we need special handling for open files
	if runtime.GOOS == "windows" {
		// Try to ensure the target doesn't exist first
		// Windows is more restrictive about renaming to existing files
		_ = os.Remove(newPath) // Ignore error if file doesn't exist
	}

	// Perform the rename operation
	err = os.Rename(realPath, newPath)
	if err != nil {
		// On Windows, if rename fails, it might be due to file being in use
		// You might want to add retry logic here for Windows
		return &fs.PathError{Op: "local-rename", Path: oldPath, Err: err}
	}

	return nil
}
