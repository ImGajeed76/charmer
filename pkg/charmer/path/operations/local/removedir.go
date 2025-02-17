package pathlocal

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func RemoveDir(path string, missingOk bool, followSymlinks bool, recursive bool) error {
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
				Op:   "local-removedir-eval-symlinks",
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
			Op:   "local-removedir-stat",
			Path: path,
			Err:  err,
		}
	}

	// Verify it's a directory
	if !info.IsDir() {
		return &fs.PathError{
			Op:   "local-removedir-notdir",
			Path: path,
			Err:  fs.ErrInvalid,
		}
	}

	// Handle Windows-specific cases for directories
	if runtime.GOOS == "windows" {
		err = os.Chmod(targetPath, 0777)
		if err != nil {
			return &fs.PathError{
				Op:   "local-removedir-chmod",
				Path: path,
				Err:  err,
			}
		}
	}

	if recursive {
		// Use RemoveAll for recursive removal
		err = os.RemoveAll(targetPath)
		if err != nil {
			return &fs.PathError{
				Op:   "local-removedir-recursive",
				Path: path,
				Err:  err,
			}
		}
	} else {
		// Try to remove the directory
		err = os.Remove(targetPath)
		if err != nil {
			if isDirectoryNotEmpty(err) {
				return &fs.PathError{
					Op:   "local-removedir-notempty",
					Path: path,
					Err:  fs.ErrInvalid,
				}
			}
			return &fs.PathError{
				Op:   "local-removedir",
				Path: path,
				Err:  err,
			}
		}
	}

	return nil
}
