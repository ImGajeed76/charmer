package pathmodels

import (
	"io/fs"
	"time"
)

type FileMode uint32

type FileInfo struct {
	Name    string    // base Name of the file
	Size    int64     // length in bytes
	Mode    FileMode  // file Mode bits
	ModTime time.Time // modification time
	IsDir   bool      // is a directory
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
	// Download Options
	Headers map[string]string
}

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

type HTTPError struct {
	Op   string
	Code int
	Msg  string
	Err  error
}

func (e *HTTPError) Error() string {
	if e.Err == nil {
		return e.Op + " " + e.Msg + "[" + string(rune(e.Code)) + "]"
	}
	return e.Op + " " + e.Msg + "[" + string(rune(e.Code)) + "]: " + e.Err.Error()
}
