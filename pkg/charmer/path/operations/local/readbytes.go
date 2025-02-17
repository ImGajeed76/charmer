package pathlocal

import (
	"bufio"
	pathhelpers "github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	"io"
	"io/fs"
	"log"
	"os"
)

func ReadBytes(filePath string) ([]byte, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, &fs.PathError{Op: "local-read-open", Path: filePath, Err: err}
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
		return nil, &fs.PathError{Op: "local-read-stat", Path: filePath, Err: err}
	}

	// Determine optimal buffer size based on system
	bufferSize := pathhelpers.GetOptimalBufferSize(fileInfo.Size())

	// Create a buffered reader for the file
	reader := bufio.NewReaderSize(file, bufferSize)

	// Read the file content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, &fs.PathError{Op: "local-read-read-all", Path: filePath, Err: err}
	}

	return content, nil
}
