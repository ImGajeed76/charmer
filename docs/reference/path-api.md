# Charmer Path API Documentation

The Path API is a powerful component of the Charmer project that provides a unified interface for file system operations
across different storage types including local files, SFTP, and URLs. This API abstracts away the complexities of
handling different storage backends, allowing for seamless operations regardless of where your files are stored.

## Table of Contents

- [Overview](#overview)
- [Creating Path Objects](#creating-path-objects)
- [Path Properties](#path-properties)
- [File Operations](#file-operations)
- [Directory Operations](#directory-operations)
- [Path Manipulation](#path-manipulation)
- [SFTP Operations](#sftp-operations)
- [URL Operations](#url-operations)
- [Error Handling](#error-handling)
- [Examples](#examples)

## Overview

The Path API unifies file operations across different storage systems:

- **Local files**: Work with files on the local file system
- **SFTP**: Securely access remote files via SFTP
- **URLs**: Interact with web resources through HTTP/HTTPS URLs

## Creating Path Objects

### `New(path string, parameter ...*SFTPConfig) *Path`

Creates a new Path object for the specified path.

```go
// Create a local path
localPath := path.New("/path/to/file.txt")

// Create an SFTP path with explicit configuration
sftpConfig := &path.SFTPConfig{
    Host:     "example.com",
    Port:     "22",
    Username: "user",
    Password: "pass",
}
sftpPath := path.New("/remote/path", sftpConfig)

// Create an SFTP path using URL format
sftpUrlPath := path.New("sftp://user:pass@example.com:22/remote/path")

// Create an HTTP URL path
urlPath := path.New("https://example.com/resource")
```

### `Cwd() *Path`

Creates a new Path object for the current working directory.

```go
cwd := path.Cwd()
```

## Path Properties

### Type Checking

```go
// Check if the path is an SFTP path
isSftp := path.IsSftp()

// Check if the path is a URL
isUrl := path.IsUrl()

// Check if the path exists
exists := path.Exists()

// Check if the path is a directory
isDir := path.IsDir()

// Check if the path is a file
isFile := path.IsFile()
```

### Path Information

```go
// Get the string representation of the path
pathStr := path.String()

// Get the SFTP URL representation (if path is SFTP)
sftpUrl := path.SftpPath()

// Get the name component of the path (filename or last directory)
name := path.Name()

// Get the stem (filename without extension)
stem := path.Stem()

// Get the suffix (file extension without the dot)
suffix := path.Suffix()
```

## File Operations

### Reading Files

```go
// Read file as text with specified encoding
content, err := path.ReadText("utf8")

// Read file as bytes
bytes, err := path.ReadBytes()
```

### Writing Files

```go
// Write text to file with specified encoding
err := path.WriteText("Hello, world!", "utf8")

// Write bytes to file
err := path.WriteBytes([]byte{72, 101, 108, 108, 111})
```

### File Information

```go
// Get detailed file information
info, err := path.Stat()
if err == nil {
    fmt.Printf("Name: %s\n", info.Name)
    fmt.Printf("Size: %d bytes\n", info.Size)
    fmt.Printf("Mode: %o\n", info.Mode)
    fmt.Printf("Modified: %s\n", info.ModTime)
    fmt.Printf("Is Directory: %t\n", info.IsDir)
}
```

## Directory Operations

### Listing Directory Contents

```go
// List all items in a directory (non-recursive)
items, err := path.List()

// List all items in a directory and subdirectories (recursive)
allItems, err := path.ListRecursive()
```

### Creating and Removing Directories

```go
// Create a directory
// - parents: Create parent directories if they don't exist
// - existsOk: Don't error if directory already exists
err := path.MakeDir(true, true)

// Remove a directory
// - missingOk: Don't error if directory doesn't exist
// - recursive: Remove all contents recursively (if this is false, directory must be empty)
// - followSymlinks: Follow symbolic links
err := path.RemoveDir(true, true, false)
```

### Finding Files with Patterns

```go
// Find all paths matching a pattern
matches, err := path.Glob("*.txt")
```

## Path Manipulation

### Path Transformation

```go
// Get parent directory
parent := path.Parent()

// Join paths
newPath := path.Join("subdir/file.txt")

// Make a copy of the Path object
pathCopy := path.Copy()

// Set a new path for the Path object
err := path.SetPath("/new/path")
```

### File Operations

```go
// Rename a file
// - newName: The new name for the file (not the full path)
// - followSymlinks: Follow symbolic links
err := path.Rename("newfilename.txt", false)

// Remove a file
// - missingOk: Don't error if file doesn't exist
// - followSymlinks: Follow symbolic links (if true, this will remove the source of the symlink)
err := path.Remove(true, false)
```

### Copying and Moving

```go
// Copy to another path
path1 := path.New("/source/path")
path2 := path.New("/destination/path")
err := path1.CopyTo(path2)

// Copy with options
options := pathmodels.CopyOptions{
	PathOptions: pathmodels.DefaultPathOption(),
    FollowSymlinks: true,
    Recursive: true,
}
err := path1.CopyTo(path2, options)

// Move to another path (with overwrite option)
err := path1.MoveTo(path2, true)
```

## SFTP Operations

### SFTP Configuration

```go
// Get connection details for an SFTP path
conn, err := path1.ConnectionDetails()
if err == nil {
    fmt.Printf("Host: %s\n", conn.Hostname)
    fmt.Printf("Port: %d\n", conn.Port)
    fmt.Printf("Username: %s\n", conn.Username)
}
```

### Cross-System Operations

The API seamlessly handles operations between different storage systems:

- Local to Local
- Local to SFTP
- SFTP to Local
- SFTP to SFTP
- URL to Local
- URL to SFTP

```go
// Copy from local to SFTP
localPath := path.New("/local/file.txt")
sftpPath := path.New("sftp://user:pass@host:22/remote/file.txt")
err := localPath.CopyTo(sftpPath)

// Copy from SFTP to local
err := sftpPath.CopyTo(localPath)

// Copy from URL to local (downloads the URL content)
urlPath := path.New("https://example.com/resource")
err := urlPath.CopyTo(localPath)
```

## URL Operations

URLs have limited operations compared to local paths and SFTP paths. They support:

- Reading basic metadata via `Stat()`
- Copying to local or SFTP paths via `CopyTo()`
- Getting path components via `Name()`, `Parent()`, etc.

In general, URLs are read-only and do not support write operations.

```go
// Check if a URL exists
urlPath := path.New("https://example.com/resource")
exists := urlPath.Exists()

// Get URL metadata
info, err := urlPath.Stat()

// Download a URL to a local path
localPath := path.New("/download/destination.file")
err := urlPath.CopyTo(localPath)
```

## Error Handling

The Path API uses a structured error system with the `pathmodels.PathError` type:

```go
if err != nil {
    if pathErr, ok := err.(*pathmodels.PathError); ok {
        fmt.Printf("Operation: %s\n", pathErr.Op)
        fmt.Printf("Path: %s\n", pathErr.Path)
        fmt.Printf("Error: %s\n", pathErr.Err)
    }
}
```

Common error constants:

- `pathmodels.ErrNotExist`: Path does not exist
- `pathmodels.ErrExist`: Path already exists

## Examples

### Working with Local Files

```go
// Create a new local path
localPath := path.New("/tmp/example.txt")

// Write content to the file
err := localPath.WriteText("Hello, Charmer!", "utf8")
if err != nil {
    log.Fatal(err)
}

// Read the file back
content, err := localPath.ReadText("utf8")
if err != nil {
    log.Fatal(err)
}
fmt.Println(content)

// Get file information
info, err := localPath.Stat()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("File size: %d bytes\n", info.Size)

// Create a directory
dirPath := path.New("/tmp/charmer_example")
err = dirPath.MakeDir(true, true)
if err != nil {
    log.Fatal(err)
}

// Copy the file to the new directory
newPath := dirPath.Join("example_copy.txt")
err = localPath.CopyTo(newPath)
if err != nil {
    log.Fatal(err)
}

// List files in the directory
files, err := dirPath.List()
if err != nil {
    log.Fatal(err)
}
for _, file := range files {
    fmt.Println(file.String())
}

// Clean up
err = localPath.Remove(true, false)
if err != nil {
    log.Fatal(err)
}
err = dirPath.RemoveDir(true, true, false)
if err != nil {
    log.Fatal(err)
}
```

### Working with SFTP

```go
// Create an SFTP path
sftpPath := path.New("sftp://user:pass@example.com:22/remote/example.txt")

// Check if the file exists
if !sftpPath.Exists() {
    // Create parent directories if needed
    parentDir := sftpPath.Parent()
    err := parentDir.MakeDir(true, true)
    if err != nil {
        log.Fatal(err)
    }
    
    // Write content to the file
    err = sftpPath.WriteText("Hello from SFTP!", "utf8")
    if err != nil {
        log.Fatal(err)
    }
}

// Read the file
content, err := sftpPath.ReadText("utf8")
if err != nil {
    log.Fatal(err)
}
fmt.Println(content)

// Copy the file locally
localPath := path.New("/tmp/sftp_example.txt")
err = sftpPath.CopyTo(localPath)
if err != nil {
    log.Fatal(err)
}
```

### Working with URLs

```go
// Create a URL path
urlPath := path.New("https://example.com/index.html")

// Check if the URL exists
if urlPath.Exists() {
    // Download the URL content
    localPath := path.New("/tmp/example_download.html")
    err := urlPath.CopyTo(localPath)
    if err != nil {
        log.Fatal(err)
    }
    
    // Read the downloaded content
    content, err := localPath.ReadText("utf8")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Downloaded %d bytes\n", len(content))
}
```

### Cross-Platform Operations

```go
// Copy from a URL to an SFTP server
urlPath := path.New("https://example.com/resource.zip")
sftpPath := path.New("sftp://user:pass@example.com:22/downloads/resource.zip")

err := urlPath.CopyTo(sftpPath)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Successfully downloaded URL directly to SFTP server")
```

## Path Validation

The Path API includes built-in validation to ensure paths are well-formed and safe to use:

```go
// Validate a path
path := path.New("/some/path")
err := path.Validate()
if err != nil {
    fmt.Printf("Invalid path: %s\n", err)
}
```

Validation checks include:

- Path length limits
- Invalid characters
- Reserved names (on Windows)
- URL format validation
- SFTP host, port, and credential validation

This validation helps prevent path-based security issues and ensures cross-platform compatibility.

!!! note "Auto Validation"

    Path objects are automatically validated when created and when used in operations. This helps catch issues early and
    ensures that paths are safe to use in file operations.

!!! warning "Documentation Errors"

    If you find any errors or inconsistencies in this documentation, please report them by 
    [creating an issue](https://github.com/ImGajeed76/charmer/issues/new) on the Charmer GitHub repository. 
    Your feedback helps improve the project for everyone!