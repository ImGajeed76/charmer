package sftp

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// TransferProgress represents progress information for a file transfer
type TransferProgress struct {
	Filename     string
	BytesWritten int64
	TotalBytes   int64
	Done         bool
	Error        error
}

// TransferCallback is a function that receives transfer progress updates
type TransferCallback func(TransferProgress)

// TransferError represents a structured error for file transfers
type TransferError struct {
	LocalPath  string
	RemotePath string
	Operation  string
	Err        error
}

func (e *TransferError) Error() string {
	return fmt.Sprintf("%s failed - local: %s, remote: %s: %v",
		e.Operation, e.LocalPath, e.RemotePath, e.Err)
}

// ClientConfig holds the configuration for the SFTP client
type ClientConfig struct {
	Host              string
	Port              string
	Username          string
	Password          string
	ConnTimeout       time.Duration
	KeepAliveInterval time.Duration
	KeepAliveMaxCount int
}

type SFTPClient struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	config     ClientConfig
	logger     *log.Logger
}

// NewSFTPClient creates a new SFTP client connection with the given configuration
func NewSFTPClient(config ClientConfig, logger *log.Logger) (*SFTPClient, error) {
	if logger == nil {
		logger = log.New(os.Stderr, "sftp: ", log.LstdFlags)
	}

	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         config.ConnTimeout,
	}

	// Connect to SSH server
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %v", err)
	}

	// Setup keepalive if configured
	if config.KeepAliveInterval > 0 {
		go func() {
			t := time.NewTicker(config.KeepAliveInterval)
			defer t.Stop()

			failCount := 0
			for range t.C {
				_, _, err := sshClient.SendRequest("keepalive@openssh.com", true, nil)
				if err != nil {
					failCount++
					logger.Printf("keepalive failed: %v", err)
					if failCount >= config.KeepAliveMaxCount {
						logger.Printf("max keepalive failures reached, closing connection")
						sshClient.Close()
						return
					}
				} else {
					failCount = 0
				}
			}
		}()
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %v", err)
	}

	return &SFTPClient{
		sshClient:  sshClient,
		sftpClient: sftpClient,
		config:     config,
		logger:     logger,
	}, nil
}

// Close closes the SFTP and SSH connections
func (c *SFTPClient) Close() error {
	if err := c.sftpClient.Close(); err != nil {
		c.logger.Printf("error closing SFTP client: %v", err)
	}
	if err := c.sshClient.Close(); err != nil {
		c.logger.Printf("error closing SSH client: %v", err)
	}
	return nil
}

// FileExists checks if a file exists on the remote server
func (c *SFTPClient) FileExists(remotePath string) (bool, error) {
	_, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// progressReader wraps an io.Reader to track progress
type progressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	filename   string
	onProgress TransferCallback
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.read += int64(n)
		if r.onProgress != nil {
			r.onProgress(TransferProgress{
				Filename:     r.filename,
				BytesWritten: r.read,
				TotalBytes:   r.total,
				Done:         err == io.EOF,
				Error:        err,
			})
		}
	}
	return n, err
}

// UploadFile uploads a local file to the remote server with progress reporting and existence check
func (c *SFTPClient) UploadFile(ctx context.Context, localPath, remotePath string, overwrite bool, callback TransferCallback) error {
	// Check if file exists
	if !overwrite {
		exists, err := c.FileExists(remotePath)
		if err != nil {
			return &TransferError{
				LocalPath:  localPath,
				RemotePath: remotePath,
				Operation:  "check existence",
				Err:        err,
			}
		}
		if exists {
			return &TransferError{
				LocalPath:  localPath,
				RemotePath: remotePath,
				Operation:  "upload",
				Err:        fmt.Errorf("file already exists and overwrite is false"),
			}
		}
	}

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return &TransferError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "open local file",
			Err:        err,
		}
	}
	defer func() {
		if err := localFile.Close(); err != nil {
			c.logger.Printf("failed to close local file %s: %v", localPath, err)
		}
	}()

	// Get file size for progress reporting
	fileInfo, err := localFile.Stat()
	if err != nil {
		return &TransferError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "stat local file",
			Err:        err,
		}
	}

	// Create remote file
	remoteFile, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return &TransferError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "create remote file",
			Err:        err,
		}
	}
	defer func() {
		if err := remoteFile.Close(); err != nil {
			c.logger.Printf("failed to close remote file %s: %v", remotePath, err)
		}
	}()

	// Create progress reader
	reader := &progressReader{
		reader:     localFile,
		total:      fileInfo.Size(),
		filename:   localPath,
		onProgress: callback,
	}

	// Use buffered copy with context
	buf := make([]byte, 1024*1024) // 1MB buffer

	copyDone := make(chan error, 1)
	go func() {
		_, err := io.CopyBuffer(remoteFile, reader, buf)
		copyDone <- err
	}()

	select {
	case err := <-copyDone:
		if err != nil {
			return &TransferError{
				LocalPath:  localPath,
				RemotePath: remotePath,
				Operation:  "copy file contents",
				Err:        err,
			}
		}
	case <-ctx.Done():
		return &TransferError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "copy file contents",
			Err:        ctx.Err(),
		}
	}

	return nil
}

// UploadFiles uploads multiple files concurrently with progress reporting
func (c *SFTPClient) UploadFiles(ctx context.Context, transfers []struct{ Local, Remote string }, overwrite bool, callback TransferCallback) []TransferError {
	var wg sync.WaitGroup
	errChan := make(chan TransferError, len(transfers))

	// Create a semaphore channel to limit concurrency
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	// Create a context that's cancelled when the function returns
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start all uploads
	for _, transfer := range transfers {
		wg.Add(1)
		go func(local, remote string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}: // Acquire semaphore
				defer func() { <-sem }() // Release semaphore
			case <-ctx.Done():
				errChan <- TransferError{
					LocalPath:  local,
					RemotePath: remote,
					Operation:  "upload",
					Err:        ctx.Err(),
				}
				return
			}

			if err := c.UploadFile(ctx, local, remote, overwrite, callback); err != nil {
				if transferErr, ok := err.(*TransferError); ok {
					errChan <- *transferErr
				} else {
					errChan <- TransferError{
						LocalPath:  local,
						RemotePath: remote,
						Operation:  "upload",
						Err:        err,
					}
				}
			}
		}(transfer.Local, transfer.Remote)
	}

	// Wait for all uploads to complete and close error channel
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect all errors
	var errors []TransferError
	for err := range errChan {
		errors = append(errors, err)
	}

	return errors
}

// DownloadFile downloads a remote file to the local machine with progress reporting and existence check
func (c *SFTPClient) DownloadFile(ctx context.Context, remotePath, localPath string, overwrite bool, callback TransferCallback) error {
	// Check if local file exists
	if !overwrite {
		if _, err := os.Stat(localPath); err == nil {
			return &TransferError{
				LocalPath:  localPath,
				RemotePath: remotePath,
				Operation:  "download",
				Err:        fmt.Errorf("local file already exists and overwrite is false"),
			}
		}
	}

	// Open remote file
	remoteFile, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return &TransferError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "open remote file",
			Err:        err,
		}
	}
	defer func() {
		if err := remoteFile.Close(); err != nil {
			c.logger.Printf("failed to close remote file %s: %v", remotePath, err)
		}
	}()

	// Get file size for progress reporting
	fileInfo, err := remoteFile.Stat()
	if err != nil {
		return &TransferError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "stat remote file",
			Err:        err,
		}
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return &TransferError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "create local file",
			Err:        err,
		}
	}
	defer func() {
		if err := localFile.Close(); err != nil {
			c.logger.Printf("failed to close local file %s: %v", localPath, err)
		}
	}()

	// Create progress reader
	reader := &progressReader{
		reader:     remoteFile,
		total:      fileInfo.Size(),
		filename:   remotePath,
		onProgress: callback,
	}

	// Use buffered copy with context
	buf := make([]byte, 1024*1024) // 1MB buffer

	copyDone := make(chan error, 1)
	go func() {
		_, err := io.CopyBuffer(localFile, reader, buf)
		copyDone <- err
	}()

	select {
	case err := <-copyDone:
		if err != nil {
			return &TransferError{
				LocalPath:  localPath,
				RemotePath: remotePath,
				Operation:  "copy file contents",
				Err:        err,
			}
		}
	case <-ctx.Done():
		return &TransferError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "copy file contents",
			Err:        ctx.Err(),
		}
	}

	return nil
}

// DownloadFiles downloads multiple files concurrently with progress reporting
func (c *SFTPClient) DownloadFiles(ctx context.Context, transfers []struct{ Remote, Local string }, overwrite bool, callback TransferCallback) []TransferError {
	var wg sync.WaitGroup
	errChan := make(chan TransferError, len(transfers))

	// Create a semaphore channel to limit concurrency
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	// Create a context that's cancelled when the function returns
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start all downloads
	for _, transfer := range transfers {
		wg.Add(1)
		go func(remote, local string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}: // Acquire semaphore
				defer func() { <-sem }() // Release semaphore
			case <-ctx.Done():
				errChan <- TransferError{
					LocalPath:  local,
					RemotePath: remote,
					Operation:  "download",
					Err:        ctx.Err(),
				}
				return
			}

			if err := c.DownloadFile(ctx, remote, local, overwrite, callback); err != nil {
				if transferErr, ok := err.(*TransferError); ok {
					errChan <- *transferErr
				} else {
					errChan <- TransferError{
						LocalPath:  local,
						RemotePath: remote,
						Operation:  "download",
						Err:        err,
					}
				}
			}
		}(transfer.Remote, transfer.Local)
	}

	// Wait for all downloads to complete and close error channel
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect all errors
	var errors []TransferError
	for err := range errChan {
		errors = append(errors, err)
	}

	return errors
}

type FileInfo struct {
	Path    string
	IsDir   bool
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
}

func (c *SFTPClient) ListFilesRecursive(ctx context.Context, remotePath string, pattern string) ([]FileInfo, error) {
	var files []FileInfo
	var regex *regexp.Regexp
	var err error

	if pattern != "" {
		regex, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %v", err)
		}
	}

	var walk func(path string) error
	walk = func(path string) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		entries, err := c.sftpClient.ReadDir(path)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %v", path, err)
		}

		for _, entry := range entries {
			fullPath := filepath.Join(path, entry.Name())

			// Create FileInfo struct
			info := FileInfo{
				Path:    fullPath,
				IsDir:   entry.IsDir(),
				Size:    entry.Size(),
				Mode:    entry.Mode(),
				ModTime: entry.ModTime(),
			}

			// Apply regex filter if pattern is provided
			if regex != nil {
				if !regex.MatchString(fullPath) {
					// If it's a directory, we still need to traverse it
					if entry.IsDir() {
						err := walk(fullPath)
						if err != nil {
							return err
						}
					}
					continue
				}
			}

			// Add the file/directory to our results
			files = append(files, info)

			// Recursively walk directories
			if entry.IsDir() {
				err := walk(fullPath)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	err = walk(remotePath)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// Example usage in main
func main() {
	logger := log.New(os.Stdout, "sftp: ", log.LstdFlags)

	config := ClientConfig{
		Host:              "example.com",
		Port:              "22",
		Username:          "username",
		Password:          "password",
		ConnTimeout:       30 * time.Second,
		KeepAliveInterval: 30 * time.Second,
		KeepAliveMaxCount: 4,
	}

	client, err := NewSFTPClient(config, logger)
	if err != nil {
		log.Fatalf("Failed to create SFTP client: %v", err)
	}
	defer client.Close()

	// Example progress callback
	progressCb := func(progress TransferProgress) {
		if progress.TotalBytes > 0 {
			percentage := float64(progress.BytesWritten) / float64(progress.TotalBytes) * 100
			log.Printf("Transfer progress for %s: %.2f%%", progress.Filename, percentage)
		}
	}

	// Upload with context and progress reporting
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Example upload
	err = client.UploadFile(ctx, "local/path/file.txt", "/remote/path/file.txt", false, progressCb)
	if err != nil {
		if transferErr, ok := err.(*TransferError); ok {
			log.Printf("Structured upload error: %+v", transferErr)
		} else {
			log.Printf("Failed to upload file: %v", err)
		}
	}

	// Example download
	err = client.DownloadFile(ctx, "/remote/path/file.txt", "local/path/downloaded.txt", false, progressCb)
	if err != nil {
		if transferErr, ok := err.(*TransferError); ok {
			log.Printf("Structured download error: %+v", transferErr)
		} else {
			log.Printf("Failed to download file: %v", err)
		}
	}

	// Example multiple transfers
	transfers := []struct{ Remote, Local string }{
		{Remote: "/remote/file1.txt", Local: "local/file1.txt"},
		{Remote: "/remote/file2.txt", Local: "local/file2.txt"},
	}

	errors := client.DownloadFiles(ctx, transfers, false, progressCb)
	if len(errors) > 0 {
		log.Printf("Some downloads failed: %+v", errors)
	}
}
