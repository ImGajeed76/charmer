package pathsftp

import (
	"bytes"
	"context"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
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
	encoded, err := encoder.Bytes([]byte(content))
	if err != nil {
		return &pathmodels.PathError{Op: "sftp-write-encode", Path: filePath, Err: err}
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
