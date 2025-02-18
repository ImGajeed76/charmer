package pathsftp

import (
	"bufio"
	"bytes"
	"context"
	pathhelpers "github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"io"
)

func WriteBytes(filePath string, data []byte, connectionDetails sftpmanager.ConnectionDetails) error {
	ctx := context.Background()

	// Get SFTP client
	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-write-get-client", Path: filePath, Err: err}
	}

	// Create the remote file
	file, err := client.Create(filePath)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-write-create", Path: filePath, Err: err}
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return it since we're in defer
			println("error closing SFTP file:", err.Error())
		}
	}()

	// Get optimal buffer size based on data length
	bufferSize := pathhelpers.GetOptimalBufferSize(int64(len(data)))

	// Create buffered writer with optimal size
	writer := bufio.NewWriterSize(file, bufferSize)

	// Create a bytes reader for the input data
	reader := bytes.NewReader(data)

	// Copy data in chunks
	if _, err := io.Copy(writer, reader); err != nil {
		return &pathmodels.PathError{Op: "sftp-write-copy", Path: filePath, Err: err}
	}

	// Flush any buffered data
	if err := writer.Flush(); err != nil {
		return &pathmodels.PathError{Op: "sftp-write-flush", Path: filePath, Err: err}
	}

	return nil
}
