package path

import (
	"errors"
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
	if p.isSftp {
		// TODO: Implement SFTP ReadText
		return "", &PathError{Op: "read", Path: p.path, Err: errors.New("SFTP not implemented")}
	}

	bytes, err := p.ReadBytes()
	if err != nil {
		return "", &PathError{Op: "read", Path: p.path, Err: err}
	}
	return string(bytes), nil
}

// WriteText writes text content to the file with the specified encoding
func (p *Path) WriteText(content string, encoding string) error {
	if p.isSftp {
		// TODO: Implement SFTP WriteText
		return &PathError{Op: "write", Path: p.path, Err: errors.New("SFTP not implemented")}
	}

	return p.WriteBytes([]byte(content))
}

// ReadBytes reads the content of the file as bytes
func (p *Path) ReadBytes() ([]byte, error) {
	if p.isSftp {
		// TODO: Implement SFTP ReadBytes
		return nil, &PathError{Op: "read", Path: p.path, Err: errors.New("SFTP not implemented")}
	}

	file, err := os.Open(p.path)
	if err != nil {
		return nil, &PathError{Op: "read", Path: p.path, Err: err}
	}
	defer file.Close()

	return io.ReadAll(file)
}

// WriteBytes writes byte content to the file
func (p *Path) WriteBytes(content []byte) error {
	if p.isSftp {
		// TODO: Implement SFTP WriteBytes
		return &PathError{Op: "write", Path: p.path, Err: errors.New("SFTP not implemented")}
	}

	err := os.WriteFile(p.path, content, os.FileMode(DefaultPathOption().Permissions))
	if err != nil {
		return &PathError{Op: "write", Path: p.path, Err: err}
	}
	return nil
}

// Exists checks if the path exists
func (p *Path) Exists() bool {
	if p.isSftp {
		// TODO: Implement SFTP Exists
		return false
	}

	_, err := os.Stat(p.path)
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
	if p.isSftp {
		// TODO: Implement SFTP List
		return nil, &PathError{Op: "list", Path: p.path, Err: errors.New("SFTP not implemented")}
	}

	if !p.IsDir() {
		return nil, &PathError{Op: "list", Path: p.path, Err: errors.New("not a directory")}
	}

	entries, err := os.ReadDir(p.path)
	if err != nil {
		return nil, &PathError{Op: "list", Path: p.path, Err: err}
	}

	paths := make([]*Path, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		childPath := p.Join(name)
		if childPath != nil {
			paths = append(paths, childPath)
		}
	}
	return paths, nil
}

// CopyTo copies the path to a destination
func (p *Path) CopyTo(dest *Path, opts ...CopyOptions) error {
	if !p.Exists() {
		return &PathError{Op: "copy", Path: p.path, Err: ErrNotExist}
	}

	opt := CopyOptions{PathOption: DefaultPathOption()}
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Handle different combinations of local and SFTP paths
	switch {
	case p.isSftp && dest.isSftp:
		// TODO: Implement SFTP to SFTP copy
		return &PathError{Op: "copy", Path: p.path, Err: errors.New("SFTP to SFTP copy not implemented")}

	case p.isSftp && !dest.isSftp:
		// TODO: Implement SFTP to local copy
		return &PathError{Op: "copy", Path: p.path, Err: errors.New("SFTP to local copy not implemented")}

	case !p.isSftp && dest.isSftp:
		// TODO: Implement local to SFTP copy
		return &PathError{Op: "copy", Path: p.path, Err: errors.New("Local to SFTP copy not implemented")}

	default: // both local
		if p.IsDir() {
			if !opt.Recursive {
				return &PathError{Op: "copy", Path: p.path, Err: errors.New("source is a directory, recursive copy required")}
			}
			return p.copyDir(dest, opt)
		}
		return p.copyFile(dest, opt)
	}
}

// MoveTo moves the path to a destination
func (p *Path) MoveTo(dest *Path, overwrite bool) error {
	if !p.Exists() {
		return &PathError{Op: "move", Path: p.path, Err: ErrNotExist}
	}

	if dest.Exists() && !overwrite {
		return &PathError{Op: "move", Path: dest.path, Err: ErrExist}
	}

	switch {
	case p.isSftp && dest.isSftp:
		// TODO: Implement SFTP to SFTP move
		return &PathError{Op: "move", Path: p.path, Err: errors.New("SFTP to SFTP move not implemented")}

	case p.isSftp && !dest.isSftp:
		// TODO: Implement SFTP to local move
		return &PathError{Op: "move", Path: p.path, Err: errors.New("SFTP to local move not implemented")}

	case !p.isSftp && dest.isSftp:
		// TODO: Implement local to SFTP move
		return &PathError{Op: "move", Path: p.path, Err: errors.New("Local to SFTP move not implemented")}

	default: // both local
		if !overwrite {
			// Try to create destination file exclusively
			_, err := os.OpenFile(dest.path, os.O_CREATE|os.O_EXCL, 0666)
			if err == nil {
				// If successful, close and remove the probe file
				os.Remove(dest.path)
			} else if !os.IsExist(err) {
				return &PathError{Op: "move", Path: dest.path, Err: err}
			} else {
				return &PathError{Op: "move", Path: dest.path, Err: ErrExist}
			}
		}

		// Try rename first (atomic if possible)
		err := os.Rename(p.path, dest.path)
		if err == nil {
			return nil
		}

		// If rename failed, try copy and delete with proper error handling
		if err := p.CopyTo(dest, CopyOptions{
			PathOption: DefaultPathOption(),
			Recursive:  true,
		}); err != nil {
			// Clean up partial destination on failure
			dest.Remove(true)
			return &PathError{Op: "move", Path: p.path, Err: err}
		}

		return p.Remove(false)
	}
}

func (p *Path) Rename()

// MakeDir creates a directory
func (p *Path) MakeDir(parents bool, existsOk bool) error {
	if p.isSftp {
		// TODO: Implement SFTP MakeDir
		return &PathError{Op: "mkdir", Path: p.path, Err: errors.New("SFTP not implemented")}
	}

	if p.Exists() {
		if !existsOk {
			return &PathError{Op: "mkdir", Path: p.path, Err: ErrExist}
		}
		return nil
	}

	var err error
	if parents {
		err = os.MkdirAll(p.path, os.FileMode(DefaultPathOption().Permissions))
	} else {
		err = os.Mkdir(p.path, os.FileMode(DefaultPathOption().Permissions))
	}

	if err != nil {
		return &PathError{Op: "mkdir", Path: p.path, Err: err}
	}
	return nil
}

// Remove removes a file
func (p *Path) Remove(missingOk bool) error {
	if p.isSftp {
		// TODO: Implement SFTP Remove
		return &PathError{Op: "remove", Path: p.path, Err: errors.New("SFTP not implemented")}
	}

	if !p.Exists() {
		if missingOk {
			return nil
		}
		return &PathError{Op: "remove", Path: p.path, Err: ErrNotExist}
	}

	if err := os.Remove(p.path); err != nil {
		return &PathError{Op: "remove", Path: p.path, Err: err}
	}
	return nil
}

// RemoveDir removes a directory
func (p *Path) RemoveDir(missingOk bool, recursive bool) error {
	if p.isSftp {
		// TODO: Implement SFTP RemoveDir
		return &PathError{Op: "rmdir", Path: p.path, Err: errors.New("SFTP not implemented")}
	}

	if !p.Exists() {
		if missingOk {
			return nil
		}
		return &PathError{Op: "rmdir", Path: p.path, Err: ErrNotExist}
	}

	if !p.IsDir() {
		return &PathError{Op: "rmdir", Path: p.path, Err: errors.New("not a directory")}
	}

	var err error
	if recursive {
		err = os.RemoveAll(p.path)
	} else {
		err = os.Remove(p.path)
	}

	if err != nil {
		return &PathError{Op: "rmdir", Path: p.path, Err: err}
	}
	return nil
}

// Stat returns file information
func (p *Path) Stat() (*FileInfo, error) {
	if p.isSftp {
		// TODO: Implement SFTP Stat
		return nil, &PathError{Op: "stat", Path: p.path, Err: errors.New("SFTP not implemented")}
	}

	info, err := os.Stat(p.path)
	if err != nil {
		return nil, &PathError{Op: "stat", Path: p.path, Err: err}
	}

	return &FileInfo{
		name:    info.Name(),
		size:    info.Size(),
		mode:    FileMode(info.Mode()),
		modTime: info.ModTime(),
		isDir:   info.IsDir(),
	}, nil
}

func (p *Path) copyFile(dest *Path, opts CopyOptions) error {
	src, err := os.Open(p.path)
	if err != nil {
		return &PathError{Op: "copy", Path: p.path, Err: err}
	}
	defer src.Close()

	if dest.Exists() {
		return &PathError{Op: "copy", Path: dest.path, Err: ErrExist}
	}

	dst, err := os.OpenFile(dest.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(opts.Permissions))
	if err != nil {
		return &PathError{Op: "copy", Path: dest.path, Err: err}
	}
	defer dst.Close()

	// Get file size for progress calculation
	srcInfo, err := src.Stat()
	if err != nil {
		return &PathError{Op: "copy", Path: p.path, Err: err}
	}
	totalSize := srcInfo.Size()

	buf := make([]byte, opts.BufferSize)
	var written int64

	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			// Clean up the destination file on error
			os.Remove(dest.path)
			return &PathError{Op: "copy", Path: p.path, Err: err}
		}
		if n == 0 {
			break
		}

		if _, err := dst.Write(buf[:n]); err != nil {
			// Clean up the destination file on error
			os.Remove(dest.path)
			return &PathError{Op: "copy", Path: dest.path, Err: err}
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
		if err := os.Chmod(dest.path, os.FileMode(srcInfo.Mode())); err != nil {
			return &PathError{Op: "copy", Path: dest.path, Err: err}
		}

		// Preserve timestamps
		atime := srcInfo.ModTime() // Note: Access time might not be available in all systems
		mtime := srcInfo.ModTime()
		if err := os.Chtimes(dest.path, atime, mtime); err != nil {
			return &PathError{Op: "copy", Path: dest.path, Err: err}
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
		return info.Size(), nil
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

func (p *Path) copyDir(dest *Path, opts CopyOptions) error {
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
			return &PathError{Op: "copy", Path: entry.path, Err: errors.New("invalid destination path")}
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

		if err := os.Chmod(dest.path, os.FileMode(srcInfo.Mode())); err != nil {
			cleanup()
			return &PathError{Op: "copy", Path: dest.path, Err: err}
		}
	}

	return nil
}
