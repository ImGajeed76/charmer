package pathsftplocal

import (
	"context"
	"github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"github.com/pkg/sftp"
	"io"
	"os"
	"path/filepath"
)

func Copy(src string, dest string, detailsSrc sftpmanager.ConnectionDetails, opts ...pathmodels.CopyOptions) error {
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

	// Get source SFTP client
	clientSrc, err := sftpmanager.GetClient(ctx, detailsSrc)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-copy-get-client-src", Path: src, Err: err}
	}

	// Get source file info
	srcInfo, err := clientSrc.Stat(src)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-stat", Path: src, Err: err}
	}

	// Handle directory copy if source is a directory
	if srcInfo.IsDir() {
		if !options.Recursive {
			return &pathmodels.PathError{Op: "sftp-copy", Path: src, Err: pathmodels.ErrInvalid}
		}
		return copyDir(ctx, src, dest, clientSrc, srcInfo, options)
	}

	return copyFile(ctx, src, dest, clientSrc, srcInfo, options)
}

func copyFile(ctx context.Context, src, dest string, clientSrc *sftp.Client, srcInfo os.FileInfo, options pathmodels.CopyOptions) error {
	// Open source file from SFTP
	srcFile, err := clientSrc.Open(src)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-open", Path: src, Err: err}
	}
	defer srcFile.Close()

	// Create destination file with temporary permissions
	// We'll set the correct permissions after writing the file
	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		nr, err := srcFile.Read(buf)
		if err != nil && err != io.EOF {
			return &pathmodels.PathError{Op: "sftp-read", Path: src, Err: err}
		}
		if nr == 0 {
			break
		}

		nw, err := destFile.Write(buf[:nr])
		if err != nil {
			return &pathmodels.PathError{Op: "write", Path: dest, Err: err}
		}
		if nw != nr {
			return &pathmodels.PathError{Op: "write", Path: dest, Err: err}
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

	// Close the file before changing attributes
	destFile.Close()

	// Preserve attributes if requested
	if options.PreserveAttributes {
		// Set the original mode (permission bits)
		if err := os.Chmod(dest, srcInfo.Mode()); err != nil {
			return &pathmodels.PathError{Op: "chmod", Path: dest, Err: err}
		}

		// Set access and modification times
		if err := os.Chtimes(dest, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
			return &pathmodels.PathError{Op: "chtimes", Path: dest, Err: err}
		}
	} else {
		// If not preserving attributes, set the permissions from options
		if err := os.Chmod(dest, os.FileMode(options.Permissions)); err != nil {
			return &pathmodels.PathError{Op: "chmod", Path: dest, Err: err}
		}
	}

	return nil
}

func copyDir(ctx context.Context, src, dest string, clientSrc *sftp.Client, srcInfo os.FileInfo, options pathmodels.CopyOptions) error {
	// Create destination directory with temporary permissions
	if err := os.MkdirAll(dest, 0700); err != nil {
		return &pathmodels.PathError{Op: "mkdir", Path: dest, Err: err}
	}

	// Read directory entries from SFTP
	entries, err := clientSrc.ReadDir(src)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-readdir", Path: src, Err: err}
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if entry.IsDir() {
			if err := copyDir(ctx, srcPath, destPath, clientSrc, entry, options); err != nil {
				return err
			}
		} else {
			if err := copyFile(ctx, srcPath, destPath, clientSrc, entry, options); err != nil {
				return err
			}
		}
	}

	// Preserve directory attributes if requested
	if options.PreserveAttributes {
		// Set the original mode (permission bits)
		if err := os.Chmod(dest, srcInfo.Mode()); err != nil {
			return &pathmodels.PathError{Op: "chmod", Path: dest, Err: err}
		}

		// Set access and modification times
		if err := os.Chtimes(dest, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
			return &pathmodels.PathError{Op: "chtimes", Path: dest, Err: err}
		}
	} else {
		// If not preserving attributes, set the permissions from options
		if err := os.Chmod(dest, os.FileMode(options.Permissions)); err != nil {
			return &pathmodels.PathError{Op: "chmod", Path: dest, Err: err}
		}
	}

	return nil
}
