package pathlocallocal

import (
	"context"
	"fmt"
	"github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	"io"
	"os"
	"path/filepath"
	"time"
)

func Move(src string, dest string, overwrite bool, opts ...pathmodels.CopyOptions) error {
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

	// Check if destination exists
	_, err = os.Stat(dest)
	if err == nil {
		if !overwrite {
			return &pathmodels.PathError{
				Op:   "move",
				Path: dest,
				Err:  fmt.Errorf("destination already exists and overwrite is false"),
			}
		}
	}

	// Try atomic rename first
	err = os.Rename(src, dest)
	if err == nil {
		return nil // Successful atomic move
	}

	// If rename fails (e.g., across devices), fall back to copy-and-delete
	if srcInfo.IsDir() {
		if !options.Recursive {
			return &pathmodels.PathError{Op: "move", Path: src, Err: pathmodels.ErrInvalid}
		}
		err = moveDir(ctx, src, dest, srcInfo, overwrite, options)
	} else {
		err = moveFile(ctx, src, dest, srcInfo, overwrite, options)
	}

	if err != nil {
		return err
	}

	// If copy was successful, remove the source
	return os.RemoveAll(src)
}

func moveFile(ctx context.Context, src, dest string, srcInfo os.FileInfo, overwrite bool, options pathmodels.CopyOptions) error {
	// Handle symbolic links
	if (srcInfo.Mode()&os.ModeSymlink != 0) && !options.FollowSymlinks {
		return moveSymlink(src, dest, overwrite)
	}

	// If overwrite is true and destination exists, create a temporary file
	tempDest := dest
	if overwrite {
		tempDest = dest + ".tmp"
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return &pathmodels.PathError{Op: "open", Path: src, Err: err}
	}
	defer srcFile.Close()

	// Create destination file with proper permissions
	destFile, err := os.OpenFile(tempDest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(options.Permissions))
	if err != nil {
		return &pathmodels.PathError{Op: "create", Path: tempDest, Err: err}
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
			os.Remove(tempDest) // Clean up temporary file
			return ctx.Err()
		default:
		}

		nr, err := srcFile.Read(buf)
		if err != nil && err != io.EOF {
			os.Remove(tempDest) // Clean up temporary file
			return &pathmodels.PathError{Op: "read", Path: src, Err: err}
		}
		if nr == 0 {
			break
		}

		nw, err := destFile.Write(buf[:nr])
		if err != nil {
			os.Remove(tempDest) // Clean up temporary file
			return &pathmodels.PathError{Op: "write", Path: tempDest, Err: err}
		}
		if nw != nr {
			os.Remove(tempDest) // Clean up temporary file
			return &pathmodels.PathError{Op: "write", Path: tempDest, Err: io.ErrShortWrite}
		}

		copied += int64(nw)
		if options.ProgressFunc != nil {
			options.ProgressFunc(srcInfo.Size(), copied)
		}
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		os.Remove(tempDest) // Clean up temporary file
		return &pathmodels.PathError{Op: "sync", Path: tempDest, Err: err}
	}

	// Close files before performing the rename
	srcFile.Close()
	destFile.Close()

	// If we're using a temporary file for overwrite, perform the atomic rename
	if overwrite && tempDest != dest {
		if err := os.Rename(tempDest, dest); err != nil {
			os.Remove(tempDest) // Clean up temporary file
			return &pathmodels.PathError{Op: "rename", Path: dest, Err: err}
		}
	}

	// Preserve attributes if requested
	if options.PreserveAttributes {
		if err := os.Chtimes(dest, time.Now(), srcInfo.ModTime()); err != nil {
			return &pathmodels.PathError{Op: "chtimes", Path: dest, Err: err}
		}
	}

	return nil
}

func moveDir(ctx context.Context, src, dest string, srcInfo os.FileInfo, overwrite bool, options pathmodels.CopyOptions) error {
	// If overwrite is true and destination exists, create a temporary directory
	tempDest := dest
	if overwrite {
		tempDest = dest + ".tmp"
	}

	// Create destination directory
	if err := os.MkdirAll(tempDest, os.FileMode(options.Permissions)); err != nil {
		return &pathmodels.PathError{Op: "mkdir", Path: tempDest, Err: err}
	}

	// Read directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		os.RemoveAll(tempDest) // Clean up temporary directory
		return &pathmodels.PathError{Op: "readdir", Path: src, Err: err}
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(tempDest, entry.Name())

		select {
		case <-ctx.Done():
			os.RemoveAll(tempDest) // Clean up temporary directory
			return ctx.Err()
		default:
		}

		info, err := entry.Info()
		if err != nil {
			os.RemoveAll(tempDest) // Clean up temporary directory
			return &pathmodels.PathError{Op: "stat", Path: srcPath, Err: err}
		}

		if info.IsDir() {
			if err := moveDir(ctx, srcPath, destPath, info, overwrite, options); err != nil {
				os.RemoveAll(tempDest) // Clean up temporary directory
				return err
			}
		} else {
			if err := moveFile(ctx, srcPath, destPath, info, overwrite, options); err != nil {
				os.RemoveAll(tempDest) // Clean up temporary directory
				return err
			}
		}
	}

	// If we're using a temporary directory for overwrite, perform the atomic rename
	if overwrite && tempDest != dest {
		if err := os.Rename(tempDest, dest); err != nil {
			os.RemoveAll(tempDest) // Clean up temporary directory
			return &pathmodels.PathError{Op: "rename", Path: dest, Err: err}
		}
	}

	// Preserve directory attributes if requested
	if options.PreserveAttributes {
		if err := os.Chtimes(dest, time.Now(), srcInfo.ModTime()); err != nil {
			return &pathmodels.PathError{Op: "chtimes", Path: dest, Err: err}
		}
	}

	return nil
}

func moveSymlink(src, dest string, overwrite bool) error {
	// Read the target of the symlink
	target, err := os.Readlink(src)
	if err != nil {
		return &pathmodels.PathError{Op: "readlink", Path: src, Err: err}
	}

	// If overwrite is true and destination exists, remove it first
	if overwrite {
		os.Remove(dest)
	}

	// Create the symlink
	if err := os.Symlink(target, dest); err != nil {
		return &pathmodels.PathError{Op: "symlink", Path: dest, Err: err}
	}

	return nil
}
