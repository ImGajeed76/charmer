package pathsftp

import (
	"context"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"path/filepath"
)

func RenameFile(oldPath string, newName string, connectionDetails sftpmanager.ConnectionDetails, followSymlinks bool) error {
	ctx := context.Background()

	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-renamefile-get-client", Path: oldPath, Err: err}
	}
	defer client.Close()

	// Clean paths to ensure consistent formatting
	oldPath = filepath.ToSlash(filepath.Clean(oldPath))
	newName = filepath.Clean(newName)

	// Check if newName contains path separators
	if filepath.Base(newName) != newName {
		return &pathmodels.PathError{
			Op:   "sftp-renamefile-validate",
			Path: oldPath,
			Err:  pathmodels.ErrInvalid,
		}
	}

	var pathToRename string
	if followSymlinks {
		// Get the real path by evaluating symlinks
		realPath, err := client.RealPath(oldPath)
		if err != nil {
			return &pathmodels.PathError{
				Op:   "sftp-renamefile-realpath",
				Path: oldPath,
				Err:  err,
			}
		}
		pathToRename = realPath
	} else {
		// Use the original path without resolving symlinks
		pathToRename = oldPath
	}

	// Get the directory of the path we're renaming
	dir := filepath.Dir(pathToRename)

	// Construct the new full path
	newPath := filepath.ToSlash(filepath.Join(dir, newName))

	// Perform the rename operation
	err = client.Rename(pathToRename, newPath)
	if err != nil {
		return &pathmodels.PathError{
			Op:   "sftp-renamefile",
			Path: oldPath,
			Err:  err,
		}
	}

	return nil
}
