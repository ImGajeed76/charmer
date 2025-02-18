package path

import (
	"errors"
	"fmt"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	pathlocal "github.com/ImGajeed76/charmer/pkg/charmer/path/operations/local"
	"github.com/ImGajeed76/charmer/pkg/charmer/path/operations/locallocal"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

func New(path string) *Path {
	if path == "" {
		return nil
	}

	// Convert Windows backslashes to forward slashes
	path = strings.ReplaceAll(path, "\\", "/")

	if strings.HasPrefix(path, "sftp://") {
		u, err := url.Parse(path)
		if err != nil {
			return nil
		}

		// Extract authentication info
		var username, password string
		if u.User != nil {
			username = u.User.Username()
			password, _ = u.User.Password()
		}

		// Extract host and port
		host := u.Hostname()
		port := u.Port()
		if port == "" {
			port = "22" // Default SFTP port
		}

		// Clean the path
		cleanPath := filepath.Clean(u.Path)
		if cleanPath == "." {
			cleanPath = "/"
		}

		return &Path{
			path:     cleanPath,
			isSftp:   true,
			host:     host,
			port:     port,
			username: username,
			password: password,
		}
	}

	// Handle local paths
	cleanPath := filepath.Clean(path)
	if cleanPath == "." {
		cleanPath = "/"
	}

	return &Path{
		path:   cleanPath,
		isSftp: false,
	}
}

// MaxPathLength is the maximum allowed length for a path
const MaxPathLength = 4096 // Common Linux PATH_MAX value

func (p *Path) Validate() error {
	if p == nil {
		return errors.New("nil path")
	}

	// Basic path validation
	if p.path == "" {
		return errors.New("empty path")
	}

	if len(p.path) > MaxPathLength {
		return fmt.Errorf("path length exceeds maximum allowed (%d characters)", MaxPathLength)
	}

	// Check for null bytes and control characters
	for _, char := range p.path {
		if char == 0 {
			return errors.New("path contains null byte")
		}
		if char < 32 && char != '\t' { // Allow tabs but no other control characters
			return fmt.Errorf("path contains invalid control character: %#U", char)
		}
	}

	// Check for invalid characters based on platform
	if runtime.GOOS == "windows" {
		// Windows-specific invalid characters
		invalidChars := `<>:"|?*`
		for _, char := range invalidChars {
			if strings.ContainsRune(p.path, char) {
				return fmt.Errorf("path contains invalid character for Windows: %c", char)
			}
		}

		// Check for reserved Windows names (CON, PRN, AUX, etc.)
		segments := strings.Split(p.path, "/")
		for _, segment := range segments {
			upperSegment := strings.ToUpper(segment)
			reserved := []string{"CON", "PRN", "AUX", "NUL",
				"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
				"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}

			for _, name := range reserved {
				if upperSegment == name || strings.HasPrefix(upperSegment, name+".") {
					return fmt.Errorf("path contains reserved Windows name: %s", segment)
				}
			}
		}
	}

	// General path validation
	if !strings.HasPrefix(p.path, "/") {
		return errors.New("path must be absolute (start with /)")
	}

	// Validate path segments
	segments := strings.Split(strings.TrimPrefix(p.path, "/"), "/")
	for _, segment := range segments {
		if segment == "" && len(segments) > 1 {
			return errors.New("path contains empty segment")
		}
		if segment == "." || segment == ".." {
			return errors.New("path contains . or .. segments after normalization")
		}
		if strings.HasSuffix(segment, " ") || strings.HasSuffix(segment, ".") {
			return errors.New("path segment cannot end with space or period")
		}
	}

	// SFTP-specific validation
	if p.isSftp {
		if p.host == "" {
			return errors.New("SFTP path missing host")
		}

		// Validate hostname
		if len(p.host) > 255 {
			return errors.New("SFTP hostname too long")
		}
		for _, label := range strings.Split(p.host, ".") {
			if len(label) > 63 {
				return errors.New("SFTP hostname label too long")
			}
			if !isValidHostnameLabel(label) {
				return fmt.Errorf("invalid SFTP hostname label: %s", label)
			}
		}

		// Validate port
		if p.port != "" {
			port, err := strconv.Atoi(p.port)
			if err != nil {
				return errors.New("invalid SFTP port number")
			}
			if port < 1 || port > 65535 {
				return errors.New("SFTP port number out of range")
			}
		}

		// Validate username if provided
		if p.username != "" {
			if len(p.username) > 255 {
				return errors.New("SFTP username too long")
			}
			for _, char := range p.username {
				if !unicode.IsPrint(char) {
					return errors.New("SFTP username contains non-printable characters")
				}
			}
		}

		// Validate password if provided
		if p.password != "" {
			if len(p.password) > 255 {
				return errors.New("SFTP password too long")
			}
			for _, char := range p.password {
				if !unicode.IsPrint(char) {
					return errors.New("SFTP password contains non-printable characters")
				}
			}
		}
	}

	return nil
}

func isValidHostnameLabel(label string) bool {
	if len(label) == 0 {
		return false
	}

	for i, char := range label {
		if i == 0 {
			if !unicode.IsLetter(char) {
				return false
			}
		} else if i == len(label)-1 {
			if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
				return false
			}
		} else {
			if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '-' {
				return false
			}
		}
	}

	return true
}

func (p *Path) IsSftp() bool {
	return p.isSftp
}

func (p *Path) String() string {
	return p.path
}

func (p *Path) SftpPath() string {
	if !p.isSftp {
		return ""
	}

	var auth string
	if p.username != "" {
		if p.password != "" {
			auth = url.UserPassword(p.username, p.password).String() + "@"
		} else {
			auth = url.User(p.username).String() + "@"
		}
	}

	return fmt.Sprintf("sftp://%s%s:%s%s", auth, p.host, p.port, p.path)
}

func (p *Path) Join(path string) *Path {
	if path == "" {
		return p // Return original path instead of nil
	}

	// Convert Windows backslashes and clean
	path = strings.ReplaceAll(path, "\\", "/")

	var newPath string
	if filepath.IsAbs(path) {
		newPath = filepath.Clean(path)
	} else {
		newPath = filepath.Clean(filepath.Join(p.path, path))
	}

	if p.isSftp {
		return &Path{
			path:     newPath,
			isSftp:   true,
			host:     p.host,
			port:     p.port,
			username: p.username,
			password: p.password,
		}
	}
	return &Path{
		path:   newPath,
		isSftp: false,
	}
}

func (p *Path) Parent() *Path {
	if p.path == "/" {
		return p // Root is its own parent
	}

	parentPath := filepath.Dir(p.path)
	if p.isSftp {
		return &Path{
			path:     parentPath,
			isSftp:   true,
			host:     p.host,
			port:     p.port,
			username: p.username,
			password: p.password,
		}
	}
	return &Path{
		path:   parentPath,
		isSftp: false,
	}
}

func (p *Path) Name() string {
	return filepath.Base(p.path)
}

func (p *Path) Stem() string {
	name := p.Name()
	ext := filepath.Ext(name)
	if ext != "" {
		return name[:len(name)-len(ext)]
	}
	return name
}

func (p *Path) Suffix() string {
	ext := filepath.Ext(p.path)
	if ext != "" {
		return ext[1:] // Remove the leading dot
	}
	return ""
}

// ReadText reads the content of the file with the specified encoding
func (p *Path) ReadText(encoding string) (string, error) {
	if err := p.Validate(); err != nil {
		return "", &pathmodels.PathError{Op: "read", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		// TODO: Implement SFTP ReadText
		return "", &pathmodels.PathError{Op: "read", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.ReadText(p.path, encoding)
	}
}

// WriteText writes text content to the file with the specified encoding
func (p *Path) WriteText(content string, encoding string) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "write", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		// TODO: Implement SFTP WriteText
		return &pathmodels.PathError{Op: "write", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.WriteText(p.path, content, encoding)
	}
}

// ReadBytes reads the content of the file as bytes
func (p *Path) ReadBytes() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, &pathmodels.PathError{Op: "read", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		// TODO: Implement SFTP WriteText
		return nil, &pathmodels.PathError{Op: "read", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.ReadBytes(p.path)
	}
}

// WriteBytes writes byte content to the file
func (p *Path) WriteBytes(content []byte) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "write", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		// TODO: Implement SFTP WriteBytes
		return &pathmodels.PathError{Op: "write", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.WriteBytes(p.path, content)
	}
}

// Exists checks if the path exists
func (p *Path) Exists() bool {
	_, err := p.Stat()
	return err == nil
}

// IsDir checks if the path is a directory
func (p *Path) IsDir() bool {
	info, err := p.Stat()
	if err != nil {
		return false
	}
	return info.IsDir
}

// IsFile checks if the path is a file
func (p *Path) IsFile() bool {
	info, err := p.Stat()
	if err != nil {
		return false
	}
	return !info.IsDir
}

// List returns a list of paths in the directory
func (p *Path) List() ([]*Path, error) {
	if err := p.Validate(); err != nil {
		return nil, &pathmodels.PathError{Op: "list", Path: p.path, Err: err}
	}

	if !p.IsDir() {
		return nil, &pathmodels.PathError{Op: "list", Path: p.path, Err: errors.New("not a directory")}
	}

	switch {
	case p.isSftp:
		return nil, &pathmodels.PathError{Op: "list", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		list, err := pathlocal.List(p.path, false)
		if err != nil {
			return nil, err
		}

		// convert list of strings to list of Paths
		paths := make([]*Path, len(list))
		for i, path := range list {
			paths[i] = New(path)
		}
		return paths, nil
	}
}

// ListRecursive returns a list of paths in the directory and all subdirectories
func (p *Path) ListRecursive() ([]*Path, error) {
	if err := p.Validate(); err != nil {
		return nil, &pathmodels.PathError{Op: "list", Path: p.path, Err: err}
	}

	if !p.IsDir() {
		return nil, &pathmodels.PathError{Op: "list", Path: p.path, Err: errors.New("not a directory")}
	}

	switch {
	case p.isSftp:
		return nil, &pathmodels.PathError{Op: "list", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		list, err := pathlocal.List(p.path, true)
		if err != nil {
			return nil, err
		}

		// convert list of strings to list of Paths
		paths := make([]*Path, len(list))
		for i, path := range list {
			paths[i] = New(path)
		}
		return paths, nil
	}
}

// CopyTo copies the path to a destination
func (p *Path) CopyTo(dest *Path, opts ...pathmodels.CopyOptions) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "copy", Path: p.path, Err: err}
	}

	if !p.Exists() {
		return &pathmodels.PathError{Op: "copy", Path: p.path, Err: pathmodels.ErrNotExist}
	}

	opt := pathmodels.CopyOptions{PathOption: pathmodels.DefaultPathOption()}
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Handle different combinations of local and SFTP paths
	switch {
	case p.isSftp && dest.isSftp:
		// TODO: Implement SFTP to SFTP copy
		return &pathmodels.PathError{Op: "copy", Path: p.path, Err: errors.New("SFTP to SFTP copy not implemented")}

	case p.isSftp && !dest.isSftp:
		// TODO: Implement SFTP to local copy
		return &pathmodels.PathError{Op: "copy", Path: p.path, Err: errors.New("SFTP to local copy not implemented")}

	case !p.isSftp && dest.isSftp:
		// TODO: Implement local to SFTP copy
		return &pathmodels.PathError{Op: "copy", Path: p.path, Err: errors.New("Local to SFTP copy not implemented")}

	default: // both local
		return pathlocallocal.Copy(p.path, dest.path, opt)
	}
}

// MoveTo moves the path to a destination
func (p *Path) MoveTo(dest *Path, overwrite bool) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "move", Path: p.path, Err: err}
	}

	if !p.Exists() {
		return &pathmodels.PathError{Op: "move", Path: p.path, Err: pathmodels.ErrNotExist}
	}

	if dest.Exists() && !overwrite {
		return &pathmodels.PathError{Op: "move", Path: dest.path, Err: pathmodels.ErrExist}
	}

	switch {
	case p.isSftp && dest.isSftp:
		// TODO: Implement SFTP to SFTP move
		return &pathmodels.PathError{Op: "move", Path: p.path, Err: errors.New("SFTP to SFTP move not implemented")}

	case p.isSftp && !dest.isSftp:
		// TODO: Implement SFTP to local move
		return &pathmodels.PathError{Op: "move", Path: p.path, Err: errors.New("SFTP to local move not implemented")}

	case !p.isSftp && dest.isSftp:
		// TODO: Implement local to SFTP move
		return &pathmodels.PathError{Op: "move", Path: p.path, Err: errors.New("Local to SFTP move not implemented")}

	default: // both local
		return pathlocallocal.Move(p.path, dest.path, overwrite)
	}
}

// Rename renames the path
func (p *Path) Rename(newName string) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "rename", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		// TODO: Implement SFTP Rename
		return &pathmodels.PathError{Op: "rename", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.RenameFile(p.path, newName)
	}
}

// MakeDir creates a directory
func (p *Path) MakeDir(parents bool, existsOk bool) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "mkdir", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		// TODO: Implement SFTP MakeDir
		return &pathmodels.PathError{Op: "mkdir", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.MakeDir(p.path, parents, existsOk)
	}
}

// Remove removes a file
func (p *Path) Remove(missingOk bool, followSymlinks bool) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "remove", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		// TODO: Implement SFTP Remove
		return &pathmodels.PathError{Op: "remove", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.Remove(p.path, missingOk, followSymlinks)
	}
}

// RemoveDir removes a directory
func (p *Path) RemoveDir(missingOk bool, recursive bool, followSymlinks bool) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "rmdir", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		// TODO: Implement SFTP RemoveDir
		return &pathmodels.PathError{Op: "rmdir", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.RemoveDir(p.path, missingOk, followSymlinks, recursive)
	}
}

// Stat returns file information
func (p *Path) Stat() (*pathmodels.FileInfo, error) {
	if err := p.Validate(); err != nil {
		return nil, &pathmodels.PathError{Op: "stat", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		// TODO: Implement SFTP Stat
		return nil, &pathmodels.PathError{Op: "stat", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.Stat(p.path)
	}
}
