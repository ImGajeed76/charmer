package path

import (
	"io/fs"
	"time"
)

type FileMode uint32

type FileInfo struct {
	name    string    // base name of the file
	size    int64     // length in bytes
	mode    FileMode  // file mode bits
	modTime time.Time // modification time
	isDir   bool      // is a directory
}

func (fi *FileInfo) Name() string       { return fi.name }
func (fi *FileInfo) Size() int64        { return fi.size }
func (fi *FileInfo) Mode() FileMode     { return fi.mode }
func (fi *FileInfo) ModTime() time.Time { return fi.modTime }
func (fi *FileInfo) IsDir() bool        { return fi.isDir }
func (fi *FileInfo) Sys() interface{}   { return nil }

type FileSystem interface {
	Open(name string) (File, error)
	Stat(name string) (fs.FileInfo, error)
	Remove(name string) error
	Rename(oldpath, newpath string) error
	MkdirAll(path string, perm FileMode) error
}

type File interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
	Name() string
	Stat() (fs.FileInfo, error)
}

type PathOption struct {
	// Permissions for new files/directories
	Permissions FileMode
	// Whether to preserve file attributes during copy
	PreserveAttributes bool
	// Buffer size for copy operations
	BufferSize int
	// Timeout for operations
	Timeout time.Duration
}

func DefaultPathOption() PathOption {
	return PathOption{
		Permissions:        0644,
		PreserveAttributes: true,
		BufferSize:         1024 * 1024, // 1MB
		Timeout:            60 * time.Second,
	}
}

type CopyOptions struct {
	PathOption
	// Whether to follow symlinks
	FollowSymlinks bool
	// Whether to copy recursively
	Recursive bool
	// ProgressFunc callback
	ProgressFunc func(total, copied int64)
}

type GlobOptions struct {
	PathOption
	// Whether to include hidden files
	IncludeHidden bool
	// Maximum depth for recursive glob
	MaxDepth int
	// File patterns to ignore
	IgnorePatterns []string
}

type SftpConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	KeyFile  string
}

const (
	// The single letters are the abbreviations
	// used by ls(1) and understood by chmod(1)
	ModeDir        FileMode = 1 << (32 - 1 - iota) // d: is a directory
	ModeAppend                                     // a: append-only
	ModeExclusive                                  // l: exclusive use
	ModeTemporary                                  // T: temporary file
	ModeSymlink                                    // L: symbolic link
	ModeDevice                                     // D: device file
	ModeNamedPipe                                  // p: named pipe (FIFO)
	ModeSocket                                     // S: Unix domain socket
	ModeSetuid                                     // u: setuid
	ModeSetgid                                     // g: setgid
	ModeCharDevice                                 // c: Unix character device
	ModeSticky                                     // t: sticky
	ModeIrregular                                  // ?: non-regular file

	// Mask for the type bits. For regular files, none will be set.
	ModeType = ModeDir | ModeSymlink | ModeNamedPipe | ModeSocket | ModeDevice | ModeCharDevice | ModeIrregular

	ModePerm FileMode = 0777 // Unix permission bits
)

var (
	ErrNotExist   = fs.ErrNotExist   // Item does not exist
	ErrExist      = fs.ErrExist      // Item already exists
	ErrPermission = fs.ErrPermission // Permission denied
	ErrInvalid    = fs.ErrInvalid    // Invalid operation
	ErrClosed     = fs.ErrClosed     // File already closed
)

type PathError struct {
	Op   string
	Path string
	Err  error
}

func (e *PathError) Error() string {
	if e.Err == nil {
		return e.Op + " " + e.Path
	}
	return e.Op + " " + e.Path + ": " + e.Err.Error()
}

type Path struct {
	path   string
	isSftp bool
}
