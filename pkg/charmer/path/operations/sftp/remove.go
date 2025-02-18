package pathsftp

import (
	"context"
	"errors"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"os"
	"path/filepath"
)

func Remove(path string, missingOk bool, followSymlinks bool, connectionDetails sftpmanager.ConnectionDetails) error {
	ctx := context.Background()

	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-remove-get-client", Path: path, Err: err}
	}
	defer client.Close()

	// Clean the path to ensure consistent formatting
	path = filepath.Clean(path)

	// Get the path to operate on based on symlink preference
	targetPath := path
	if followSymlinks {
		realPath, err := client.ReadLink(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) && missingOk {
				return nil
			}
			return &pathmodels.PathError{
				Op:   "sftp-remove-read-link",
				Path: path,
				Err:  err,
			}
		}
		// If it's a symlink, use the resolved path
		if realPath != "" {
			targetPath = realPath
		}
	}

	// Get file info without following symlinks
	info, err := client.Lstat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && missingOk {
			return nil
		}
		return &pathmodels.PathError{
			Op:   "sftp-remove-stat",
			Path: path,
			Err:  err,
		}
	}

	// For directories, check if empty
	if info.IsDir() {
		entries, err := client.ReadDir(targetPath)
		if err != nil {
			return &pathmodels.PathError{
				Op:   "sftp-remove-readdir",
				Path: path,
				Err:  err,
			}
		}
		if len(entries) > 0 {
			return &pathmodels.PathError{
				Op:   "sftp-remove-notempty",
				Path: path,
				Err:  os.ErrInvalid,
			}
		}
	}

	// Perform the removal
	err = client.Remove(targetPath)
	if err != nil {
		return &pathmodels.PathError{
			Op:   "sftp-remove",
			Path: path,
			Err:  err,
		}
	}

	return nil
}
