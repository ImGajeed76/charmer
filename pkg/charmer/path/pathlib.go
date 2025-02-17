package path

import (
	"errors"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	pathlocal "github.com/ImGajeed76/charmer/pkg/charmer/path/operations/local"
	"github.com/ImGajeed76/charmer/pkg/charmer/path/operations/locallocal"
	"io"
	"os"
	"strings"
)

func New(path string) *Path {
	if path == "" {
		return nil
	}

	if strings.HasPrefix(path, "sftp://") {
		modifiedPath := strings.TrimPrefix(path, "sftp://")
		if modifiedPath == "" {
			return nil
		}
		if !strings.Contains(modifiedPath, "/") {
			return nil
		}

		if strings.Contains(modifiedPath, "@") {
			segments := strings.Split(modifiedPath, "/")
			if len(segments) < 2 {
				return nil
			}

			modifiedPath = strings.Join(segments[1:], "/")
		}

		return &Path{
			path:   modifiedPath,
			isSftp: true,
		}
	}

	return &Path{
		path:   path,
		isSftp: false,
	}
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

	return "sftp://" + p.path
}

func (p *Path) Join(path string) *Path {
	if path == "" {
		return nil
	}

	path = strings.TrimPrefix(path, "/")
	if p.isSftp {
		return New(p.SftpPath() + "/" + path)
	}
	return New(p.path + "/" + path)
}

func (p *Path) Parent() *Path {
	if p.isSftp {
		segments := strings.Split(p.path, "/")
		if len(segments) < 2 {
			return nil
		}

		return New("sftp://" + strings.Join(segments[:len(segments)-1], "/"))
	}

	segments := strings.Split(p.path, "/")
	if len(segments) < 2 {
		return nil
	}

	return New(strings.Join(segments[:len(segments)-1], "/"))
}

func (p *Path) Name() string {
	segments := strings.Split(p.path, "/")
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}

func (p *Path) Stem() string {
	name := p.Name()
	if strings.Contains(name, ".") {
		return strings.Split(name, ".")[0]
	}

	return name
}

func (p *Path) Suffix() string {
	name := p.Name()
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

// ReadText reads the content of the file with the specified encoding
func (p *Path) ReadText(encoding string) (string, error) {
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
	if p.isSftp {
		// TODO: Implement SFTP IsDir
		return false
	}

	info, err := os.Stat(p.path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile checks if the path is a file
func (p *Path) IsFile() bool {
	if p.isSftp {
		// TODO: Implement SFTP IsFile
		return false
	}

	info, err := os.Stat(p.path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// List returns a list of paths in the directory
func (p *Path) List() ([]*Path, error) {
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

func (p *Path) Rename(newName string) error {
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
	switch {
	case p.isSftp:
		// TODO: Implement SFTP Stat
		return nil, &pathmodels.PathError{Op: "stat", Path: p.path, Err: errors.New("SFTP not implemented")}
	default:
		return pathlocal.Stat(p.path)
	}
}

func (p *Path) copyFile(dest *Path, opts pathmodels.CopyOptions) error {
	src, err := os.Open(p.path)
	if err != nil {
		return &pathmodels.PathError{Op: "copy", Path: p.path, Err: err}
	}
	defer src.Close()

	if dest.Exists() {
		return &pathmodels.PathError{Op: "copy", Path: dest.path, Err: pathmodels.ErrExist}
	}

	dst, err := os.OpenFile(dest.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(opts.Permissions))
	if err != nil {
		return &pathmodels.PathError{Op: "copy", Path: dest.path, Err: err}
	}
	defer dst.Close()

	// Get file size for progress calculation
	srcInfo, err := src.Stat()
	if err != nil {
		return &pathmodels.PathError{Op: "copy", Path: p.path, Err: err}
	}
	totalSize := srcInfo.Size()

	buf := make([]byte, opts.BufferSize)
	var written int64

	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			// Clean up the destination file on error
			os.Remove(dest.path)
			return &pathmodels.PathError{Op: "copy", Path: p.path, Err: err}
		}
		if n == 0 {
			break
		}

		if _, err := dst.Write(buf[:n]); err != nil {
			// Clean up the destination file on error
			os.Remove(dest.path)
			return &pathmodels.PathError{Op: "copy", Path: dest.path, Err: err}
		}

		written += int64(n)

		// Call progress function if provided
		if opts.ProgressFunc != nil {
			opts.ProgressFunc(totalSize, written)
		}
	}

	if opts.PreserveAttributes {
		srcInfo, err := p.Stat()
		if err != nil {
			return err
		}

		// Preserve mode
		if err := os.Chmod(dest.path, os.FileMode(srcInfo.Mode)); err != nil {
			return &pathmodels.PathError{Op: "copy", Path: dest.path, Err: err}
		}

		// Preserve timestamps
		atime := srcInfo.ModTime // Note: Access time might not be available in all systems
		mtime := srcInfo.ModTime
		if err := os.Chtimes(dest.path, atime, mtime); err != nil {
			return &pathmodels.PathError{Op: "copy", Path: dest.path, Err: err}
		}
	}

	return nil
}

// calculateDirSize calculates total size of directory
func (p *Path) calculateDirSize() (int64, error) {
	if !p.IsDir() {
		info, err := p.Stat()
		if err != nil {
			return 0, err
		}
		return info.Size, nil
	}

	var size int64
	entries, err := p.List()
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		entrySize, err := entry.calculateDirSize()
		if err != nil {
			return 0, err
		}
		size += entrySize
	}
	return size, nil
}

func (p *Path) copyDir(dest *Path, opts pathmodels.CopyOptions) error {
	var copiedFiles []string

	cleanup := func() {
		for _, file := range copiedFiles {
			os.Remove(file)
		}
	}

	if err := dest.MakeDir(true, true); err != nil {
		return err
	}

	// Calculate total size for progress tracking
	totalSize, err := p.calculateDirSize()
	if err != nil {
		return err
	}

	var totalWritten int64
	entries, err := p.List()
	if err != nil {
		return err
	}

	// Create a wrapped progress function that adjusts for the overall directory progress
	wrappedProgressFunc := func(fileTotal, fileWritten int64) {
		if opts.ProgressFunc != nil {
			opts.ProgressFunc(totalSize, totalWritten+fileWritten)
		}
	}

	// Create modified options for sub-operations
	subOpts := opts
	subOpts.ProgressFunc = wrappedProgressFunc

	for _, entry := range entries {
		destPath := dest.Join(entry.Name())
		if destPath == nil {
			cleanup()
			return &pathmodels.PathError{Op: "copy", Path: entry.path, Err: errors.New("invalid destination path")}
		}

		// Get size of current entry
		entrySize, err := entry.calculateDirSize()
		if err != nil {
			cleanup()
			return err
		}

		if entry.IsDir() {
			if err := entry.copyDir(destPath, subOpts); err != nil {
				cleanup()
				return err
			}
		} else {
			if err := entry.copyFile(destPath, subOpts); err != nil {
				cleanup()
				return err
			}
			copiedFiles = append(copiedFiles, destPath.path)
		}

		totalWritten += entrySize
		if opts.ProgressFunc != nil {
			opts.ProgressFunc(totalSize, totalWritten)
		}
	}

	if opts.PreserveAttributes {
		srcInfo, err := p.Stat()
		if err != nil {
			cleanup()
			return err
		}

		if err := os.Chmod(dest.path, os.FileMode(srcInfo.Mode)); err != nil {
			cleanup()
			return &pathmodels.PathError{Op: "copy", Path: dest.path, Err: err}
		}
	}

	return nil
}
