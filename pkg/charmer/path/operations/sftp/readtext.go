package pathsftp

import (
	"bufio"
	"bytes"
	"context"
	pathhelpers "github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"io"
)

func ReadText(filePath string, encodingName string, connectionDetails sftpmanager.ConnectionDetails) (string, error) {
	ctx := context.Background()

	// Get SFTP client
	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return "", &pathmodels.PathError{Op: "sftp-read-get-client", Path: filePath, Err: err}
	}

	// Get encoding
	enc, err := ianaindex.IANA.Encoding(encodingName)
	if err != nil {
		return "", &pathmodels.PathError{Op: "sftp-read-get-encoding", Path: filePath, Err: err}
	}
	if enc == nil {
		enc = encoding.Nop
	}

	// Open the remote file
	file, err := client.Open(filePath)
	if err != nil {
		return "", &pathmodels.PathError{Op: "sftp-read-open", Path: filePath, Err: err}
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return it since we're in defer
			// Consider using a proper logging framework
			println("error closing SFTP file:", err.Error())
		}
	}()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", &pathmodels.PathError{Op: "sftp-read-stat", Path: filePath, Err: err}
	}

	// Get optimal buffer size
	bufferSize := pathhelpers.GetOptimalBufferSize(fileInfo.Size())

	// Create buffered reader with optimal size
	reader := bufio.NewReaderSize(file, bufferSize)

	// Create a buffer to store the file content
	// Using bytes.Buffer for better memory efficiency
	var contentBuffer bytes.Buffer
	contentBuffer.Grow(int(fileInfo.Size())) // Preallocate buffer to avoid resizing

	// Copy data in chunks
	if _, err := io.Copy(&contentBuffer, reader); err != nil {
		return "", &pathmodels.PathError{Op: "sftp-read-copy", Path: filePath, Err: err}
	}

	// Create decoder and decode content
	decoder := enc.NewDecoder()
	decoded, err := decoder.Bytes(contentBuffer.Bytes())
	if err != nil {
		return "", &pathmodels.PathError{Op: "sftp-read-decode", Path: filePath, Err: err}
	}

	return string(decoded), nil
}
