package pathmodels

import "io/fs"

type File interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
	Name() string
	Stat() (fs.FileInfo, error)
}
