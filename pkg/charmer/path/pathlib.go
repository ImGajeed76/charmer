package path

import (
	"errors"
	"fmt"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	pathlocal "github.com/ImGajeed76/charmer/pkg/charmer/path/operations/local"
	"github.com/ImGajeed76/charmer/pkg/charmer/path/operations/locallocal"
	pathlocalsftp "github.com/ImGajeed76/charmer/pkg/charmer/path/operations/localsftp"
	pathsftp "github.com/ImGajeed76/charmer/pkg/charmer/path/operations/sftp"
	pathsftplocal "github.com/ImGajeed76/charmer/pkg/charmer/path/operations/sftplocal"
	pathsftpsftp "github.com/ImGajeed76/charmer/pkg/charmer/path/operations/sftpsftp"
	sftpmanager "github.com/ImGajeed76/charmer/pkg/charmer/sftp"
	"log"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

func New(path string, parameter ...*SFTPConfig) *Path {
	if path == "" {
		return nil
	}

	// Get config
	var sftpConf *SFTPConfig = nil
	if len(parameter) > 0 {
		sftpConf = parameter[0]
	}

	// Convert Windows backslashes to forward slashes
	path = strings.ReplaceAll(path, "\\", "/")

	if strings.HasPrefix(path, "sftp://") && sftpConf == nil {
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

	// Use config if available
	if sftpConf != nil {
		if !strings.HasPrefix(cleanPath, "/") {
			log.Fatal("SFTP path must be absolute")
			return nil
		}

		return &Path{
			path:     cleanPath,
			isSftp:   true,
			host:     sftpConf.Host,
			port:     sftpConf.Port,
			username: sftpConf.Username,
			password: sftpConf.Password,
		}
	}

	// If path is relative, convert to absolute
	absPath := cleanPath
	if strings.HasPrefix(cleanPath, ".") {
		var err error
		absPath, err = filepath.Abs(cleanPath)
		if err != nil {
			log.Fatal(err)
			return nil
		}
	}

	return &Path{
		path:   absPath,
		isSftp: false,
	}
}

func (p *Path) ConnectionDetails() (*sftpmanager.ConnectionDetails, error) {
	if !p.isSftp {
		return nil, &pathmodels.PathError{Op: "connection-details", Path: p.path, Err: errors.New("Path is no sftp path")}
	}

	portI, convErr := strconv.Atoi(p.port)
	if convErr != nil {
		return nil, &pathmodels.PathError{Op: "connection-details", Path: p.path, Err: errors.New("Cannot convert port to int")}
	}

	return &sftpmanager.ConnectionDetails{
		Hostname: p.host,
		Port:     portI,
		Username: p.username,
		Password: p.password,
	}, nil
}

func (p *Path) Copy() *Path {
	return &Path{
		path:     p.path,
		isSftp:   p.isSftp,
		host:     p.host,
		port:     p.port,
		username: p.username,
		password: p.password,
	}
}

func (p *Path) SetPath(path string) error {
	if path == "" {
		return errors.New("empty path")
	}

	// Convert Windows backslashes to forward slashes
	path = strings.ReplaceAll(path, "\\", "/")

	if strings.HasPrefix(path, "sftp://") {
		return errors.New("cannot change path to SFTP path. please create a new path instead")
	}

	cleanPath := filepath.Clean(path)
	if cleanPath == "." {
		cleanPath = "/"
	}

	p.path = cleanPath
	return nil
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
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return "", connErr
		}
		return pathsftp.ReadText(p.path, encoding, *conn)
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
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return connErr
		}
		return pathsftp.WriteText(p.path, content, encoding, *conn)
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
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return nil, connErr
		}
		return pathsftp.ReadBytes(p.path, *conn)
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
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return connErr
		}
		return pathsftp.WriteBytes(p.path, content, *conn)
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
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return nil, connErr
		}
		list, err := pathsftp.List(p.path, false, *conn)
		if err != nil {
			return nil, err
		}

		// convert list of strings to list of Paths
		paths := make([]*Path, len(list))
		for i, path := range list {
			// make sure the new path is also sftp
			paths[i] = p.Copy()
			err := paths[i].SetPath(path)
			if err != nil {
				return nil, err
			}
		}
		return paths, nil
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
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return nil, connErr
		}
		list, err := pathsftp.List(p.path, true, *conn)
		if err != nil {
			return nil, err
		}

		// convert list of strings to list of Paths
		paths := make([]*Path, len(list))
		for i, path := range list {
			// make sure the new path is also sftp
			paths[i] = p.Copy()
			err := paths[i].SetPath(path)
			if err != nil {
				return nil, err
			}
		}
		return paths, nil
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
		connSrc, connSrcErr := p.ConnectionDetails()
		if connSrcErr != nil {
			return connSrcErr
		}
		connDest, connDestErr := dest.ConnectionDetails()
		if connDestErr != nil {
			return connDestErr
		}
		return pathsftpsftp.Copy(p.path, dest.path, *connSrc, *connDest, opt)

	case p.isSftp && !dest.isSftp:
		connSrc, connSrcErr := p.ConnectionDetails()
		if connSrcErr != nil {
			return connSrcErr
		}
		return pathsftplocal.Copy(p.path, dest.path, *connSrc, opt)

	case !p.isSftp && dest.isSftp:
		connDest, connDestErr := dest.ConnectionDetails()
		if connDestErr != nil {
			return connDestErr
		}
		return pathlocalsftp.Copy(p.path, dest.path, *connDest, opt)

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
		connSrc, connSrcErr := p.ConnectionDetails()
		if connSrcErr != nil {
			return connSrcErr
		}
		connDest, connDestErr := dest.ConnectionDetails()
		if connDestErr != nil {
			return connDestErr
		}
		return pathsftpsftp.Move(p.path, dest.path, *connSrc, *connDest, overwrite)

	case p.isSftp && !dest.isSftp:
		connSrc, connSrcErr := p.ConnectionDetails()
		if connSrcErr != nil {
			return connSrcErr
		}
		return pathsftplocal.Move(p.path, dest.path, *connSrc, overwrite)

	case !p.isSftp && dest.isSftp:
		connDest, connDestErr := dest.ConnectionDetails()
		if connDestErr != nil {
			return connDestErr
		}
		return pathlocalsftp.Move(p.path, dest.path, *connDest, overwrite)

	default: // both local
		return pathlocallocal.Move(p.path, dest.path, overwrite)
	}
}

// Rename renames the path
func (p *Path) Rename(newName string, followSymlinks bool) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "rename", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return connErr
		}
		return pathsftp.RenameFile(p.path, newName, *conn, followSymlinks)
	default:
		return pathlocal.RenameFile(p.path, newName, followSymlinks)
	}
}

// MakeDir creates a directory
func (p *Path) MakeDir(parents bool, existsOk bool) error {
	if err := p.Validate(); err != nil {
		return &pathmodels.PathError{Op: "mkdir", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return connErr
		}
		return pathsftp.MakeDir(p.path, parents, existsOk, *conn)
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
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return connErr
		}
		return pathsftp.Remove(p.path, missingOk, followSymlinks, *conn)
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
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return connErr
		}
		return pathsftp.RemoveDir(p.path, missingOk, followSymlinks, recursive, *conn)
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
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return nil, connErr
		}
		return pathsftp.Stat(p.path, *conn)
	default:
		return pathlocal.Stat(p.path)
	}
}

// Glob returns a list of paths matching the pattern
func (p *Path) Glob(pattern string) ([]*Path, error) {
	if err := p.Validate(); err != nil {
		return nil, &pathmodels.PathError{Op: "glob", Path: p.path, Err: err}
	}

	switch {
	case p.isSftp:
		conn, connErr := p.ConnectionDetails()
		if connErr != nil {
			return nil, connErr
		}
		stringPaths, err := pathsftp.Glob(p.path, pattern, *conn)
		if err != nil {
			return nil, err
		}
		// map stringPaths to Paths
		paths := make([]*Path, len(stringPaths))
		for i, str := range stringPaths {
			paths[i] = p.Copy()
			err := paths[i].SetPath(str)
			if err != nil {
				return nil, err
			}
		}
		return paths, nil
	default:
		stringPaths, err := pathlocal.Glob(p.path, pattern)
		if err != nil {
			return nil, err
		}
		// map stringPaths to Paths
		paths := make([]*Path, len(stringPaths))
		for i, str := range stringPaths {
			paths[i] = New(str)
		}
		return paths, nil
	}
}
