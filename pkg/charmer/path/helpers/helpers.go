package helpers

import "runtime"

// GetOptimalBufferSize returns the optimal buffer size based on the file size and system
func GetOptimalBufferSize(fileSize int64) int {
	// Base buffer size (4KB)
	baseSize := 4 * 1024

	// For small files, use file size as buffer size
	if fileSize < int64(baseSize) {
		return int(fileSize)
	}

	// Scale buffer size based on available CPU cores
	cpuCount := runtime.GOMAXPROCS(0)
	scaledSize := baseSize * cpuCount

	// Cap maximum buffer size at 1MB
	maxSize := 1 * 1024 * 1024
	if scaledSize > maxSize {
		return maxSize
	}

	return scaledSize
}
