package pathlocal

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func Remove(path string, missingOk bool, followSymlinks bool) error {
	// Clean the path to ensure consistent formatting for the OS
	path = filepath.Clean(path)

	// Get the path to operate on based on symlink preference
	targetPath := path
	if followSymlinks {
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			if os.IsNotExist(err) && missingOk {
				return nil
			}
			return &fs.PathError{
				Op:   "local-remove-eval-symlinks",
				Path: path,
				Err:  err,
			}
		}
		targetPath = realPath
	}

	// Get file info, using Lstat to not follow symlinks
	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) && missingOk {
			return nil
		}
		return &fs.PathError{
			Op:   "local-remove-stat",
			Path: path,
			Err:  err,
		}
	}

	// Handle Windows-specific cases for directories
	if runtime.GOOS == "windows" && info.IsDir() && !info.Mode().Type().IsRegular() {
		err = os.Chmod(targetPath, 0777)
		if err != nil {
			return &fs.PathError{
				Op:   "local-remove-chmod",
				Path: path,
				Err:  err,
			}
		}
	}

	// Perform the removal
	err = os.Remove(targetPath)
	if err != nil {
		// Handle directory not empty case
		if info.IsDir() && isDirectoryNotEmpty(err) {
			return &fs.PathError{
				Op:   "local-remove-notempty",
				Path: path,
				Err:  fs.ErrInvalid,
			}
		}
		return &fs.PathError{
			Op:   "local-remove",
			Path: path,
			Err:  err,
		}
	}

	return nil
}

// isDirectoryNotEmpty checks if the error is because the directory is not empty
func isDirectoryNotEmpty(err error) bool {
	if runtime.GOOS == "windows" {
		return err.Error() == "The directory is not empty."
	}
	return err.Error() == "directory not empty"
}
