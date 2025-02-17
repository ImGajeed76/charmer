package pathlocal

import (
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	"os"
)

func Stat(path string) (*pathmodels.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, &pathmodels.PathError{Op: "stat", Path: path, Err: err}
	}

	return &pathmodels.FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    pathmodels.FileMode(info.Mode()),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}, nil
}
