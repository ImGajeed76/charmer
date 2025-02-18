package pathsftp

import (
	"context"
	"errors"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"github.com/pkg/sftp"
	"io/fs"
	"path/filepath"
)

func RemoveDir(path string, missingOk bool, followSymlinks bool, recursive bool, connectionDetails sftpmanager.ConnectionDetails) error {
	ctx := context.Background()

	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-removedir-get-client", Path: path, Err: err}
	}
	defer client.Close()

	// Clean the path to ensure consistent formatting
	path = filepath.Clean(path)

	// Get the path to operate on based on symlink preference
	targetPath := path
	if followSymlinks {
		realPath, err := client.ReadLink(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && missingOk {
				return nil
			}
			return &pathmodels.PathError{
				Op:   "sftp-removedir-read-link",
				Path: path,
				Err:  err,
			}
		}
		targetPath = realPath
	}

	// Get file info, using Lstat to not follow symlinks
	info, err := client.Lstat(targetPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && missingOk {
			return nil
		}
		return &pathmodels.PathError{
			Op:   "sftp-removedir-stat",
			Path: path,
			Err:  err,
		}
	}

	// Verify it's a directory
	if !info.IsDir() {
		return &pathmodels.PathError{
			Op:   "sftp-removedir-notdir",
			Path: path,
			Err:  fs.ErrInvalid,
		}
	}

	if recursive {
		// For recursive removal, we need to implement our own RemoveAll
		// since SFTP doesn't have a direct equivalent
		err = removeAllSFTP(client, targetPath)
		if err != nil {
			return &pathmodels.PathError{
				Op:   "sftp-removedir-recursive",
				Path: path,
				Err:  err,
			}
		}
	} else {
		// Try to remove the directory
		err = client.Remove(targetPath)
		if err != nil {
			// Check if directory is not empty
			entries, listErr := client.ReadDir(targetPath)
			if listErr == nil && len(entries) > 0 {
				return &pathmodels.PathError{
					Op:   "sftp-removedir-notempty",
					Path: path,
					Err:  fs.ErrInvalid,
				}
			}
			return &pathmodels.PathError{
				Op:   "sftp-removedir",
				Path: path,
				Err:  err,
			}
		}
	}

	return nil
}

// removeAllSFTP recursively removes a directory and all its contents
func removeAllSFTP(client *sftp.Client, path string) error {
	entries, err := client.ReadDir(path)
	if err != nil {
		return err
	}

	// Remove contents first
	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			err = removeAllSFTP(client, fullPath)
		} else {
			err = client.Remove(fullPath)
		}
		if err != nil {
			return err
		}
	}

	// Remove the directory itself
	return client.Remove(path)
}
