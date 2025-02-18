package pathsftp

import (
	"context"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"io/fs"
	"path/filepath"
)

func MakeDir(path string, parents bool, existsOk bool, connectionDetails sftpmanager.ConnectionDetails) error {
	ctx := context.Background()

	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-mkdir-get-client", Path: path, Err: err}
	}
	defer client.Close()

	// Clean the path to ensure consistent formatting
	path = filepath.Clean(path)

	// Check if path exists
	info, err := client.Stat(path)
	if err == nil {
		// Path exists
		if info.IsDir() {
			// It's a directory
			if existsOk {
				return nil
			}
			return &pathmodels.PathError{
				Op:   "sftp-mkdir-exists",
				Path: path,
				Err:  fs.ErrExist,
			}
		}
		// Path exists but is not a directory
		return &pathmodels.PathError{
			Op:   "sftp-mkdir-notdir",
			Path: path,
			Err:  fs.ErrExist,
		}
	}

	if err.Error() != "file does not exist" {
		// Error is not about non-existence
		return &pathmodels.PathError{
			Op:   "sftp-mkdir-stat",
			Path: path,
			Err:  err,
		}
	}

	if !parents {
		// Single directory creation
		err = client.Mkdir(path)
		if err != nil {
			return &pathmodels.PathError{
				Op:   "sftp-mkdir",
				Path: path,
				Err:  err,
			}
		}
		return nil
	}

	// Create parent directories
	// We need to implement MkdirAll functionality since SFTP doesn't provide it
	current := "/"
	for _, part := range filepath.SplitList(path) {
		current = filepath.Join(current, part)
		err := client.Mkdir(current)
		if err != nil {
			// Ignore already exists error for parent directories
			if info, statErr := client.Stat(current); statErr == nil && info.IsDir() {
				continue
			}
			return &pathmodels.PathError{
				Op:   "sftp-mkdir-all",
				Path: current,
				Err:  err,
			}
		}
	}

	return nil
}
