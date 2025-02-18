package pathsftp

import (
	"context"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	"github.com/ImGajeed76/charmer/pkg/charmer/sftp"
)

func Stat(path string, connectionDetails sftpmanager.ConnectionDetails) (*pathmodels.FileInfo, error) {
	ctx := context.Background()

	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return nil, &pathmodels.PathError{Op: "sftp-stat-get-client", Path: path, Err: err}
	}

	info, err := client.Stat(path)
	if err != nil {
		return nil, &pathmodels.PathError{Op: "sftp-stat", Path: path, Err: err}
	}

	return &pathmodels.FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    pathmodels.FileMode(info.Mode()),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}, nil
}
