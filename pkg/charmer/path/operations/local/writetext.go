package pathlocal

import (
	"bufio"
	pathhelpers "github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"io/fs"
	"log"
	"os"
)

func WriteText(filePath string, content string, encodingName string) error {
	// Get the encoding
	enc, err := ianaindex.IANA.Encoding(encodingName)
	if err != nil {
		return &fs.PathError{Op: "local-write-get-encoding", Path: filePath, Err: err}
	}
	if enc == nil {
		enc = encoding.Nop
	}

	// Create an encoder for the specified encoding
	encoder := enc.NewEncoder()

	// Encode the content
	encoded, err := encoder.Bytes([]byte(content))
	if err != nil {
		return &fs.PathError{Op: "local-write-encode", Path: filePath, Err: err}
	}

	// Create or truncate the file
	file, err := os.Create(filePath)
	if err != nil {
		return &fs.PathError{Op: "local-write-create", Path: filePath, Err: err}
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("error closing file: %v", err)
		}
	}(file)

	// Get optimal buffer size based on content length
	bufferSize := pathhelpers.GetOptimalBufferSize(int64(len(encoded)))

	// Create a buffered writer
	writer := bufio.NewWriterSize(file, bufferSize)

	// Write the encoded content
	_, err = writer.Write(encoded)
	if err != nil {
		return &fs.PathError{Op: "local-write-write", Path: filePath, Err: err}
	}

	// Flush the buffer to ensure all data is written to disk
	err = writer.Flush()
	if err != nil {
		return &fs.PathError{Op: "local-write-flush", Path: filePath, Err: err}
	}

	return nil
}
