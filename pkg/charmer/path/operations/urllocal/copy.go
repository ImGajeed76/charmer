package pathurllocal

import (
	"context"
	"github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Copy downloads a file from a URL to a local destination path
func Copy(url string, dest string, opts ...pathmodels.CopyOptions) error {
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

	// Create the destination directory if it doesn't exist
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, os.FileMode(options.Permissions)); err != nil {
		return &pathmodels.PathError{Op: "mkdir", Path: destDir, Err: err}
	}

	// Create destination file with proper permissions
	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(options.Permissions))
	if err != nil {
		return &pathmodels.PathError{Op: "create", Path: dest, Err: err}
	}
	defer destFile.Close()

	// Get optimal buffer size or use the one specified in options
	bufferSize := helpers.GetOptimalBufferSize(resp.ContentLength)
	if options.BufferSize > 0 {
		bufferSize = options.BufferSize
	}

	// Create buffer for downloading
	buf := make([]byte, bufferSize)

	// Get total file size for progress calculation (if available)
	contentLength := resp.ContentLength
	downloaded := int64(0)

	// Download the file contents
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
			return &pathmodels.PathError{Op: "write", Path: dest, Err: err}
		}
		if nw != nr {
			return &pathmodels.PathError{Op: "write", Path: dest, Err: io.ErrShortWrite}
		}

		downloaded += int64(nw)
		if options.ProgressFunc != nil && contentLength > 0 {
			options.ProgressFunc(contentLength, downloaded)
		}
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return &pathmodels.PathError{Op: "sync", Path: dest, Err: err}
	}

	return nil
}
