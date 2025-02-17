package pathlocal

import (
	"io/fs"
	"os"
	"path/filepath"
)

func MakeDir(path string, parents bool, existsOk bool) error {
	// Clean the path to ensure consistent formatting for the OS
	path = filepath.Clean(path)

	// Check if path exists
	info, err := os.Stat(path)
	if err == nil {
		// Path exists
		if info.IsDir() {
			// It's a directory
			if existsOk {
				return nil
			}
			return &fs.PathError{
				Op:   "local-mkdir-exists",
				Path: path,
				Err:  fs.ErrExist,
			}
		}
		// Path exists but is not a directory
		return &fs.PathError{
			Op:   "local-mkdir-notdir",
			Path: path,
			Err:  fs.ErrExist,
		}
	}

	if !os.IsNotExist(err) {
		// Error is not about non-existence (e.g., permission denied)
		return &fs.PathError{
			Op:   "local-mkdir-stat",
			Path: path,
			Err:  err,
		}
	}

	if !parents {
		// Single directory creation
		err = os.Mkdir(path, 0755)
		if err != nil {
			return &fs.PathError{
				Op:   "local-mkdir",
				Path: path,
				Err:  err,
			}
		}
		return nil
	}

	// Create parent directories
	err = os.MkdirAll(path, 0755)
	if err != nil {
		return &fs.PathError{
			Op:   "local-mkdir-all",
			Path: path,
			Err:  err,
		}
	}

	return nil
}
