package pathlocal

import (
	"bufio"
	pathhelpers "github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"io"
	"io/fs"
	"log"
	"os"
)

func ReadText(filePath string, encodingName string) (string, error) {
	enc, err := ianaindex.IANA.Encoding(encodingName)
	if err != nil {
		return "", &fs.PathError{Op: "local-read-get-encoding", Path: filePath, Err: err}
	}
	if enc == nil {
		enc = encoding.Nop
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", &fs.PathError{Op: "local-read-open", Path: filePath, Err: err}
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("error closing file: %v", err)
		}
	}(file)

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", &fs.PathError{Op: "local-read-stat", Path: filePath, Err: err}
	}

	// Determine optimal buffer size based on system
	bufferSize := pathhelpers.GetOptimalBufferSize(fileInfo.Size())

	// Create a buffered reader for the file
	reader := bufio.NewReaderSize(file, bufferSize)

	// Create a decoder for the specified encoding
	decoder := enc.NewDecoder()

	// Read the file content
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", &fs.PathError{Op: "local-read-read-all", Path: filePath, Err: err}
	}

	// Decode the content
	decoded, err := decoder.Bytes(content)
	if err != nil {
		return "", &fs.PathError{Op: "local-read-decode", Path: filePath, Err: err}
	}

	return string(decoded), nil
}
