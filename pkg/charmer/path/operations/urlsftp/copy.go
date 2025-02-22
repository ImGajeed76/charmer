package pathurlsftp

import (
	"context"
	"github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Copy downloads a file from a URL and uploads it to an SFTP destination
func Copy(url string, dest string, details sftpmanager.ConnectionDetails, opts ...pathmodels.CopyOptions) error {
	// Apply default options if none provided
	options := pathmodels.CopyOptions{
		PathOption: pathmodels.DefaultPathOption(),
	}
	if len(opts) > 0 {
		options = opts[0]
	}

	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	// Create a new HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &pathmodels.PathError{Op: "request", Path: url, Err: err}
	}

	// Add headers to the request
	for key, value := range options.Headers {
		req.Header.Add(key, value)
	}

	// Perform the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &pathmodels.PathError{Op: "get", Path: url, Err: err}
	}
	defer resp.Body.Close()

	// Check if the response status code is successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &pathmodels.PathError{Op: "get", Path: url, Err: &pathmodels.HTTPError{Code: resp.StatusCode, Msg: resp.Status}}
	}

	// Get SFTP client
	sftpClient, err := sftpmanager.GetClient(ctx, details)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-get-client", Path: dest, Err: err}
	}

	// Create the destination directory on SFTP server if it doesn't exist
	destDir := filepath.Dir(dest)
	if err := sftpClient.MkdirAll(destDir); err != nil {
		return &pathmodels.PathError{Op: "sftp-mkdir", Path: destDir, Err: err}
	}

	// Create destination file on SFTP server
	destFile, err := sftpClient.Create(dest)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-create", Path: dest, Err: err}
	}
	defer destFile.Close()

	// Get optimal buffer size or use the one specified in options
	bufferSize := helpers.GetOptimalBufferSize(resp.ContentLength)
	if options.BufferSize > 0 {
		bufferSize = options.BufferSize
	}

	// Create buffer for downloading/uploading
	buf := make([]byte, bufferSize)

	// Get total file size for progress calculation (if available)
	contentLength := resp.ContentLength
	transferred := int64(0)

	// Download from URL and upload to SFTP in chunks
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		nr, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return &pathmodels.PathError{Op: "read", Path: url, Err: err}
		}
		if nr == 0 {
			break
		}

		nw, err := destFile.Write(buf[:nr])
		if err != nil {
			return &pathmodels.PathError{Op: "sftp-write", Path: dest, Err: err}
		}
		if nw != nr {
			return &pathmodels.PathError{Op: "sftp-write", Path: dest, Err: io.ErrShortWrite}
		}

		transferred += int64(nw)
		if options.ProgressFunc != nil && contentLength > 0 {
			options.ProgressFunc(contentLength, transferred)
		}
	}

	// Set file permissions on SFTP server
	if err := sftpClient.Chmod(dest, os.FileMode(options.PathOption.Permissions)); err != nil {
		return &pathmodels.PathError{Op: "sftp-chmod", Path: dest, Err: err}
	}

	return nil
}
