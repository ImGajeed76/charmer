package pathsftp

import (
	"bytes"
	"context"
	"errors"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"io/fs"
)

func WriteText(filePath string, content string, encodingName string, connectionDetails sftpmanager.ConnectionDetails) error {
	ctx := context.Background()

	// Get SFTP client
	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-write-get-client", Path: filePath, Err: err}
	}

	// Get encoding
	enc, err := ianaindex.IANA.Encoding(encodingName)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-write-get-encoding", Path: filePath, Err: err}
	}
	if enc == nil {
		enc = encoding.Nop
	}

	// Create encoder and encode content
	encoder := enc.NewEncoder()
	decoder := enc.NewDecoder()

	// First encode the content
	encoded, err := encoder.Bytes([]byte(content))
	if err != nil {
		return &fs.PathError{Op: "sftp-write-encode", Path: filePath, Err: err}
	}

	// Then try to decode it back - this validates that the encoding is correct
	var decoded []byte
	decoded, err = decoder.Bytes(encoded)
	if err != nil {
		return &fs.PathError{Op: "sftp-write-validate", Path: filePath,
			Err: errors.New("content cannot be represented in specified encoding: " + err.Error())}
	}

	if string(decoded) != content {
		return &fs.PathError{Op: "sftp-write-validate", Path: filePath,
			Err: errors.New("content cannot be represented in specified encoding")}
	}

	// Create or truncate the remote file
	file, err := client.Create(filePath)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-write-create", Path: filePath, Err: err}
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return it since we're in defer
			// Consider using a proper logging framework
			println("error closing SFTP file:", err.Error())
		}
	}()

	// Create a buffer with the encoded content
	contentBuffer := bytes.NewBuffer(encoded)

	// Write the entire content
	_, err = contentBuffer.WriteTo(file)
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-write-content", Path: filePath, Err: err}
	}

	return nil
}
