package pathsftpsftp

import (
	"context"
	"fmt"
	"github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"github.com/pkg/sftp"
	"io"
	"os"
	"path/filepath"
)

func Copy(src string, dest string, detailsSrc sftpmanager.ConnectionDetails, detailsDest sftpmanager.ConnectionDetails, opts ...pathmodels.CopyOptions) error {
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

	// Check if source and destination are on the same server
	sameServer := detailsSrc.Hostname == detailsDest.Hostname &&
		detailsSrc.Port == detailsDest.Port &&
		detailsSrc.Username == detailsDest.Username

	// Handle directory copy if source is a directory
	if srcInfo.IsDir() {
		if !options.Recursive {
			return &pathmodels.PathError{Op: "sftp-copy", Path: src, Err: pathmodels.ErrInvalid}
		}
		return copyDir(ctx, src, dest, clientSrc, detailsSrc, detailsDest, srcInfo, sameServer, options)
	}

	return copyFile(ctx, src, dest, clientSrc, detailsSrc, detailsDest, srcInfo, sameServer, options)
}

func copyFile(ctx context.Context, src, dest string, clientSrc *sftp.Client, detailsSrc, detailsDest sftpmanager.ConnectionDetails, srcInfo os.FileInfo, sameServer bool, options pathmodels.CopyOptions) error {
	if sameServer {
		// Use server-side copy for files on the same server
		session, err := sftpmanager.GetSSHSession(ctx, detailsSrc)
		if err != nil {
			return &pathmodels.PathError{Op: "sftp-copy-get-session", Path: src, Err: err}
		}
		defer session.Close()

		// Use cp with preserve attributes flag
		cmd := fmt.Sprintf("cp -p %s %s", src, dest)
		if err := session.Run(cmd); err == nil {
			return nil
		}

		// If server-side copy fails, fallback to downloading and uploading
	}

	// For different servers, we need to download and upload
	// Get destination SFTP client
	clientDest, err := sftpmanager.GetClient(ctx, detailsDest)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-copy-get-client-dest", Path: dest, Err: err}
	}

	// Open source file
	srcFile, err := clientSrc.Open(src)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-open", Path: src, Err: err}
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := clientDest.Create(dest)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-create", Path: dest, Err: err}
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
			return &pathmodels.PathError{Op: "sftp-write", Path: dest, Err: err}
		}
		if nw != nr {
			return &pathmodels.PathError{Op: "sftp-write", Path: dest, Err: err}
		}

		copied += int64(nw)
		if options.ProgressFunc != nil {
			options.ProgressFunc(srcInfo.Size(), copied)
		}
	}

	// Preserve file mode
	if err := clientDest.Chmod(dest, srcInfo.Mode()); err != nil {
		return &pathmodels.PathError{Op: "sftp-chmod", Path: dest, Err: err}
	}

	// Preserve modification and access times
	mTime := srcInfo.ModTime()
	aTime := mTime // Since os.FileInfo doesn't provide access time, we'll use mTime as a fallback
	if err := clientDest.Chtimes(dest, aTime, mTime); err != nil {
		return &pathmodels.PathError{Op: "sftp-chtimes", Path: dest, Err: err}
	}

	return nil
}

func copyDir(ctx context.Context, src, dest string, clientSrc *sftp.Client, detailsSrc, detailsDest sftpmanager.ConnectionDetails, srcInfo os.FileInfo, sameServer bool, options pathmodels.CopyOptions) error {
	// Get destination client if needed
	var clientDest *sftp.Client
	var err error
	if !sameServer {
		clientDest, err = sftpmanager.GetClient(ctx, detailsDest)
		if err != nil {
			return &pathmodels.PathError{Op: "sftp-copy-get-client-dest", Path: dest, Err: err}
		}
	}

	// Create destination directory
	if !sameServer {
		if err := clientDest.MkdirAll(dest); err != nil {
			return &pathmodels.PathError{Op: "sftp-mkdir", Path: dest, Err: err}
		}
		// Preserve directory mode
		if err := clientDest.Chmod(dest, srcInfo.Mode()); err != nil {
			return &pathmodels.PathError{Op: "sftp-chmod", Path: dest, Err: err}
		}
	} else {
		if err := clientSrc.MkdirAll(dest); err != nil {
			return &pathmodels.PathError{Op: "sftp-mkdir", Path: dest, Err: err}
		}
		// Preserve directory mode
		if err := clientSrc.Chmod(dest, srcInfo.Mode()); err != nil {
			return &pathmodels.PathError{Op: "sftp-chmod", Path: dest, Err: err}
		}
	}

	// Read directory entries
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
			if err := copyDir(ctx, srcPath, destPath, clientSrc, detailsSrc, detailsDest, entry, sameServer, options); err != nil {
				return err
			}
		} else {
			if err := copyFile(ctx, srcPath, destPath, clientSrc, detailsSrc, detailsDest, entry, sameServer, options); err != nil {
				return err
			}
		}
	}

	// Preserve directory timestamps after all contents have been copied
	mTime := srcInfo.ModTime()
	aTime := mTime // Since os.FileInfo doesn't provide access time, we'll use mTime as a fallback
	if !sameServer {
		if err := clientDest.Chtimes(dest, aTime, mTime); err != nil {
			return &pathmodels.PathError{Op: "sftp-chtimes", Path: dest, Err: err}
		}
	} else {
		if err := clientSrc.Chtimes(dest, aTime, mTime); err != nil {
			return &pathmodels.PathError{Op: "sftp-chtimes", Path: dest, Err: err}
		}
	}

	return nil
}
