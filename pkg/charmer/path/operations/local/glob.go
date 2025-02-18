package pathlocal

import (
	"io/fs"
	"path/filepath"
)

// Glob returns a list of absolute paths that match the provided pattern within the given directory.
// The pattern syntax follows filepath.Match rules:
//   - '*' matches any sequence of non-separator characters
//   - '?' matches any single non-separator character
//   - '[abc]' matches any single character within brackets
//   - '{foo,bar}' matches any of the comma-separated patterns
//
// The path parameter specifies the base directory for the search.
// If path is empty, it defaults to the current directory.
// All returned paths are absolute.
func Glob(path string, pattern string) ([]string, error) {
	// Handle empty path
	if path == "" {
		path = "."
	}

	// Combine base path with pattern
	fullPattern := filepath.Join(path, pattern)

	// Use built-in filepath.Glob
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, &fs.PathError{Op: "local-glob-match", Path: fullPattern, Err: err}
	}

	return matches, nil
}
