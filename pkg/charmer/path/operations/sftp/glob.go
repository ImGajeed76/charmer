package pathsftp

import (
	"context"
	"path/filepath"

	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
)

// Glob returns a list of absolute paths that match the provided pattern within the given directory over SFTP.
// The pattern syntax follows filepath.Match rules:
//   - '*' matches any sequence of non-separator characters
//   - '?' matches any single non-separator character
//   - '[abc]' matches any single character within brackets
//   - '{foo,bar}' matches any of the comma-separated patterns
//
// The path parameter specifies the base directory for the search.
// If path is empty, it defaults to the current directory.
func Glob(path string, pattern string, connectionDetails sftpmanager.ConnectionDetails) ([]string, error) {
	ctx := context.Background()

	// Handle empty path
	if path == "" {
		path = "."
	}

	// Get SFTP client
	client, err := sftpmanager.GetClient(ctx, connectionDetails)
	if err != nil {
		return nil, &pathmodels.PathError{Op: "sftp-glob-get-client", Path: path, Err: err}
	}
	defer client.Close()

	// Combine base path with pattern
	fullPattern := filepath.ToSlash(filepath.Join(path, pattern))

	// Use the built-in SFTP Glob function
	matches, err := client.Glob(fullPattern)
	if err != nil {
		return nil, &pathmodels.PathError{Op: "sftp-glob-match", Path: fullPattern, Err: err}
	}

	return matches, nil
}
