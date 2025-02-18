package pathsftplocal

import (
	"context"
	"fmt"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"os"
	"path/filepath"
)

func Move(src string, dest string, details sftpmanager.ConnectionDetails, overwrite bool, opts ...pathmodels.CopyOptions) error {
	// Apply default options if none provided
	options := pathmodels.CopyOptions{
		PathOption: pathmodels.DefaultPathOption(),
	}
	if len(opts) > 0 {
		options = opts[0]
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	// Get SFTP client
	client, err := sftpmanager.GetClient(ctx, details)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-move-get-client", Path: src, Err: err}
	}

	// Get source file info
	srcInfo, err := client.Stat(src)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-stat", Path: src, Err: err}
	}

	// Check if destination exists and handle overwrite
	_, err = os.Stat(dest)
	if err == nil {
		if !overwrite {
			return &pathmodels.PathError{
				Op:   "move",
				Path: dest,
				Err:  fmt.Errorf("destination already exists and overwrite is false"),
			}
		}
		// If overwrite is true, we'll handle it during the copy operation
	}

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(dest)
	if err := os.MkdirAll(parentDir, os.FileMode(options.Permissions)); err != nil {
		return &pathmodels.PathError{Op: "mkdir", Path: parentDir, Err: err}
	}

	// Copy the file/directory from SFTP to local
	if err := Copy(src, dest, details, opts...); err != nil {
		return &pathmodels.PathError{Op: "sftp-local-copy", Path: src, Err: err}
	}

	// After successful copy, delete the source from SFTP
	if srcInfo.IsDir() {
		if err := client.RemoveAll(src); err != nil {
			return &pathmodels.PathError{Op: "sftp-remove-dir", Path: src, Err: err}
		}
	} else {
		if err := client.Remove(src); err != nil {
			return &pathmodels.PathError{Op: "sftp-remove-file", Path: src, Err: err}
		}
	}

	return nil
}
