package pathlocallocal

import (
	"context"
	"github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	"io"
	"os"
	"path/filepath"
	"time"
)

func Copy(src string, dest string, opts ...pathmodels.CopyOptions) error {
	// Apply default options if none provided
	options := pathmodels.CopyOptions{
		PathOption: pathmodels.DefaultPathOption(),
	}
	if len(opts) > 0 {
		options = opts[0]
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	// Get source file info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return &pathmodels.PathError{Op: "stat", Path: src, Err: err}
	}

	// Handle directory copy if source is a directory
	if srcInfo.IsDir() {
		// Automatically enable recursive for directory copies
		options.Recursive = true
		return copyDir(ctx, src, dest, srcInfo, options)
	}

	return copyFile(ctx, src, dest, srcInfo, options)
}

func copyFile(ctx context.Context, src, dest string, srcInfo os.FileInfo, options pathmodels.CopyOptions) error {
	// Handle symbolic links
	if (srcInfo.Mode()&os.ModeSymlink != 0) && !options.FollowSymlinks {
		return copySymlink(src, dest)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return &pathmodels.PathError{Op: "open", Path: src, Err: err}
	}
	defer srcFile.Close()

	var permissions os.FileMode
	if options.PreserveAttributes {
		// Preserve all mode bits, not just permission bits
		permissions = srcInfo.Mode()
	} else {
		permissions = os.FileMode(options.Permissions)
	}

	// Create destination file with proper permissions
	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permissions)
	if err != nil {
		return &pathmodels.PathError{Op: "create", Path: dest, Err: err}
	}
	defer destFile.Close()

	// Get optimal buffer size
	bufferSize := helpers.GetOptimalBufferSize(srcInfo.Size())
	if options.BufferSize > 0 {
		bufferSize = options.BufferSize
	}

	// Create buffer for copying
	buf := make([]byte, bufferSize)
	copied := int64(0)

	// Copy the file contents
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		nr, err := srcFile.Read(buf)
		if err != nil && err != io.EOF {
			return &pathmodels.PathError{Op: "read", Path: src, Err: err}
		}
		if nr == 0 {
			break
		}

		nw, err := destFile.Write(buf[:nr])
		if err != nil {
			return &pathmodels.PathError{Op: "write", Path: dest, Err: err}
		}
		if nw != nr {
			return &pathmodels.PathError{Op: "write", Path: dest, Err: io.ErrShortWrite}
		}

		copied += int64(nw)
		if options.ProgressFunc != nil {
			options.ProgressFunc(srcInfo.Size(), copied)
		}
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return &pathmodels.PathError{Op: "sync", Path: dest, Err: err}
	}

	// Preserve attributes if requested
	if options.PreserveAttributes {
		// Preserve modification and access times
		if err := os.Chtimes(dest, time.Now(), srcInfo.ModTime()); err != nil {
			return &pathmodels.PathError{Op: "chtimes", Path: dest, Err: err}
		}

		// Ensure all mode bits are preserved
		if err := os.Chmod(dest, srcInfo.Mode()); err != nil {
			return &pathmodels.PathError{Op: "chmod", Path: dest, Err: err}
		}
	}

	return nil
}

func copyDir(ctx context.Context, src, dest string, srcInfo os.FileInfo, options pathmodels.CopyOptions) error {
	// Get original directory permissions if preserving attributes
	var dirMode os.FileMode
	if options.PreserveAttributes {
		dirMode = srcInfo.Mode()
	} else {
		dirMode = os.FileMode(options.Permissions)
	}

	// Create destination directory with proper permissions
	if err := os.MkdirAll(dest, dirMode); err != nil {
		return &pathmodels.PathError{Op: "mkdir", Path: dest, Err: err}
	}

	// Read directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		return &pathmodels.PathError{Op: "readdir", Path: src, Err: err}
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		info, err := entry.Info()
		if err != nil {
			return &pathmodels.PathError{Op: "stat", Path: srcPath, Err: err}
		}

		if info.IsDir() {
			if err := copyDir(ctx, srcPath, destPath, info, options); err != nil {
				return err
			}
		} else {
			if err := copyFile(ctx, srcPath, destPath, info, options); err != nil {
				return err
			}
		}
	}

	// Preserve directory attributes if requested
	if options.PreserveAttributes {
		// Preserve modification and access times
		if err := os.Chtimes(dest, time.Now(), srcInfo.ModTime()); err != nil {
			return &pathmodels.PathError{Op: "chtimes", Path: dest, Err: err}
		}

		// Ensure directory mode is preserved (in case umask affected MkdirAll)
		if err := os.Chmod(dest, dirMode); err != nil {
			return &pathmodels.PathError{Op: "chmod", Path: dest, Err: err}
		}
	}

	return nil
}

func copySymlink(src, dest string) error {
	// Read the target of the symlink
	target, err := os.Readlink(src)
	if err != nil {
		return &pathmodels.PathError{Op: "readlink", Path: src, Err: err}
	}

	// Create the symlink
	if err := os.Symlink(target, dest); err != nil {
		return &pathmodels.PathError{Op: "symlink", Path: dest, Err: err}
	}

	return nil
}
