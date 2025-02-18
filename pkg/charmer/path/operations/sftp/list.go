package pathsftp

import (
	"context"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"path/filepath"
)

func List(dirPath string, recursive bool, connectionDetails sftpmanager.ConnectionDetails) ([]string, error) {
	ctx := context.Background()

	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return nil, &pathmodels.PathError{Op: "sftp-list-get-client", Path: dirPath, Err: err}
	}
	defer client.Close()

	// Check if path exists and is a directory
	info, err := client.Stat(dirPath)
	if err != nil {
		return nil, &pathmodels.PathError{Op: "sftp-list-stat", Path: dirPath, Err: err}
	}
	if !info.IsDir() {
		return nil, &pathmodels.PathError{
			Op:   "sftp-list-check",
			Path: dirPath,
			Err:  pathmodels.ErrInvalid,
		}
	}

	var paths []string

	if recursive {
		// Walk through all subdirectories
		walker := client.Walk(dirPath)
		for walker.Step() {
			if err := walker.Err(); err != nil {
				return nil, &pathmodels.PathError{Op: "sftp-list-walk", Path: dirPath, Err: err}
			}
			path := walker.Path()
			if path != dirPath { // Skip the root directory itself
				paths = append(paths, path)
			}
		}
	} else {
		// Read only the immediate directory
		entries, err := client.ReadDir(dirPath)
		if err != nil {
			return nil, &pathmodels.PathError{Op: "sftp-list-read", Path: dirPath, Err: err}
		}

		// Convert entries to full paths
		for _, entry := range entries {
			paths = append(paths, filepath.Join(dirPath, entry.Name()))
		}
	}

	return paths, nil
}
