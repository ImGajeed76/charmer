package pathsftpsftp

import (
	"context"
	"fmt"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"path/filepath"
)

func Move(src string, dest string, detailsSrc sftpmanager.ConnectionDetails, detailsDest sftpmanager.ConnectionDetails, overwrite bool, opts ...pathmodels.CopyOptions) error {
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

	// Get source SFTP client
	clientSrc, err := sftpmanager.GetClient(ctx, detailsSrc)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-move-get-client-src", Path: src, Err: err}
	}

	// Get source file info
	srcInfo, err := clientSrc.Stat(src)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-stat", Path: src, Err: err}
	}

	// Check if source and destination are on the same server
	sameServer := detailsSrc.Hostname == detailsDest.Hostname &&
		detailsSrc.Port == detailsDest.Port &&
		detailsSrc.Username == detailsDest.Username

	if sameServer {
		// For same server operations, use rename command
		// First, ensure parent directory exists
		parentDir := filepath.Dir(dest)
		if err := clientSrc.MkdirAll(parentDir); err != nil {
			return &pathmodels.PathError{Op: "sftp-mkdir", Path: parentDir, Err: err}
		}

		// Check if destination exists and handle overwrite
		_, err := clientSrc.Stat(dest)
		if err == nil {
			if !overwrite {
				return &pathmodels.PathError{Op: "sftp-move", Path: dest, Err: fmt.Errorf("destination already exists")}
			}
			// Remove existing destination
			if err := clientSrc.Remove(dest); err != nil {
				return &pathmodels.PathError{Op: "sftp-remove", Path: dest, Err: err}
			}
		}

		// Perform rename operation
		if err := clientSrc.Rename(src, dest); err != nil {
			return &pathmodels.PathError{Op: "sftp-rename", Path: src, Err: err}
		}

		return nil
	}

	// For different servers, copy then delete
	// First copy the file/directory
	if err := Copy(src, dest, detailsSrc, detailsDest, opts...); err != nil {
		return &pathmodels.PathError{Op: "sftp-move-copy", Path: src, Err: err}
	}

	// After successful copy, delete the source
	if srcInfo.IsDir() {
		if err := clientSrc.RemoveAll(src); err != nil {
			return &pathmodels.PathError{Op: "sftp-remove-dir", Path: src, Err: err}
		}
	} else {
		if err := clientSrc.Remove(src); err != nil {
			return &pathmodels.PathError{Op: "sftp-remove-file", Path: src, Err: err}
		}
	}

	return nil
}
