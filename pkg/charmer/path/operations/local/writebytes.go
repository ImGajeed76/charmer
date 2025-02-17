package pathlocal

import (
	"bufio"
	pathhelpers "github.com/ImGajeed76/charmer/pkg/charmer/path/helpers"
	"io"
	"io/fs"
	"log"
	"os"
)

func WriteBytes(filePath string, data []byte) error {
	// Create or truncate the file
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return &fs.PathError{Op: "local-write-create", Path: filePath, Err: err}
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("error closing file: %v", err)
		}
	}(file)

	// Determine optimal buffer size based on data length
	bufferSize := pathhelpers.GetOptimalBufferSize(int64(len(data)))

	// Create a buffered writer
	writer := bufio.NewWriterSize(file, bufferSize)

	// Write the data
	_, err = io.Copy(writer, NewByteReader(data))
	if err != nil {
		return &fs.PathError{Op: "local-write-copy", Path: filePath, Err: err}
	}

	// Flush the buffered writer to ensure all data is written
	err = writer.Flush()
	if err != nil {
		return &fs.PathError{Op: "local-write-flush", Path: filePath, Err: err}
	}

	return nil
}

// ByteReader implements io.Reader for a byte slice
type ByteReader struct {
	data []byte
	pos  int
}

// NewByteReader creates a new ByteReader
func NewByteReader(data []byte) *ByteReader {
	return &ByteReader{data: data}
}

// Read implements io.Reader
func (r *ByteReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
