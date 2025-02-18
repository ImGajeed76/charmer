package pathlocalsftp

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

	// Get source file info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return &pathmodels.PathError{Op: "stat", Path: src, Err: err}
	}

	// Get SFTP client
	client, err := sftpmanager.GetClient(ctx, details)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-move-get-client", Path: dest, Err: err}
	}

	// Check if destination exists
	_, err = client.Stat(dest)
	if err == nil {
		if !overwrite {
			return &pathmodels.PathError{
				Op:   "move",
				Path: dest,
				Err:  fmt.Errorf("destination already exists and overwrite is false"),
			}
		}
		// Remove existing destination if overwrite is true
		if err := client.Remove(dest); err != nil {
			return &pathmodels.PathError{Op: "sftp-remove", Path: dest, Err: err}
		}
	}

	// Create parent directory on SFTP server if it doesn't exist
	parentDir := filepath.Dir(dest)
	if err := client.MkdirAll(parentDir); err != nil {
		return &pathmodels.PathError{Op: "sftp-mkdir", Path: parentDir, Err: err}
	}

	// Copy the file/directory to SFTP
	if err := Copy(src, dest, details, opts...); err != nil {
		return &pathmodels.PathError{Op: "local-to-sftp-copy", Path: src, Err: err}
	}

	// After successful copy, delete the source
	if srcInfo.IsDir() {
		if err := os.RemoveAll(src); err != nil {
			return &pathmodels.PathError{Op: "remove-dir", Path: src, Err: err}
		}
	} else {
		if err := os.Remove(src); err != nil {
			return &pathmodels.PathError{Op: "remove-file", Path: src, Err: err}
		}
	}

	return nil
}
