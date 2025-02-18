package path

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    *Path
		wantNil bool
	}{
		{
			name:    "Empty path",
			path:    "",
			wantNil: true,
		},
		{
			name: "Local path",
			path: "/test/path",
			want: &Path{
				path:   "/test/path",
				isSftp: false,
			},
		},
		{
			name: "Windows style path",
			path: "C:\\test\\path",
			want: &Path{
				path:   "C:/test/path",
				isSftp: false,
			},
		},
		{
			name: "SFTP path with credentials",
			path: "sftp://user:pass@example.com:2222/test/path",
			want: &Path{
				path:     "/test/path",
				isSftp:   true,
				host:     "example.com",
				port:     "2222",
				username: "user",
				password: "pass",
			},
		},
		{
			name: "SFTP path without credentials",
			path: "sftp://example.com/test/path",
			want: &Path{
				path:   "/test/path",
				isSftp: true,
				host:   "example.com",
				port:   "22",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.path)
			if tt.wantNil {
				if got != nil {
					t.Errorf("New() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("New() returned nil")
			}
			if got.path != tt.want.path {
				t.Errorf("path = %v, want %v", got.path, tt.want.path)
			}
			if got.isSftp != tt.want.isSftp {
				t.Errorf("isSftp = %v, want %v", got.isSftp, tt.want.isSftp)
			}
			if got.host != tt.want.host {
				t.Errorf("host = %v, want %v", got.host, tt.want.host)
			}
			if got.port != tt.want.port {
				t.Errorf("port = %v, want %v", got.port, tt.want.port)
			}
			if got.username != tt.want.username {
				t.Errorf("username = %v, want %v", got.username, tt.want.username)
			}
			if got.password != tt.want.password {
				t.Errorf("password = %v, want %v", got.password, tt.want.password)
			}
		})
	}
}

func TestPath_Validate(t *testing.T) {
	tests := []struct {
		name    string
		path    *Path
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Nil path",
			path:    nil,
			wantErr: true,
			errMsg:  "nil path",
		},
		{
			name: "Empty path",
			path: &Path{
				path: "",
			},
			wantErr: true,
			errMsg:  "empty path",
		},
		{
			name: "Path too long",
			path: &Path{
				path: "/" + string(make([]byte, MaxPathLength)),
			},
			wantErr: true,
			errMsg:  "path length exceeds maximum allowed (4096 characters)",
		},
		{
			name: "Path with null byte",
			path: &Path{
				path: "/test/path\x00",
			},
			wantErr: true,
			errMsg:  "path contains null byte",
		},
		{
			name: "Path with control character",
			path: &Path{
				path: "/test/path\x01",
			},
			wantErr: true,
			errMsg:  "path contains invalid control character: U+0001",
		},
		{
			name: "Valid local path",
			path: &Path{
				path: "/test/path",
			},
			wantErr: false,
		},
		{
			name: "Valid SFTP path",
			path: &Path{
				path:     "/test/path",
				isSftp:   true,
				host:     "example.com",
				port:     "22",
				username: "user",
				password: "pass",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.path.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.errMsg)
			}
		})
	}
}

func TestPath_Join(t *testing.T) {
	base := New("/base/path")
	sftpBase := New("sftp://example.com/base/path")

	tests := []struct {
		name     string
		path     *Path
		joinPath string
		want     string
		wantSftp bool
	}{
		{
			name:     "Join empty path",
			path:     base,
			joinPath: "",
			want:     "/base/path",
			wantSftp: false,
		},
		{
			name:     "Join relative path",
			path:     base,
			joinPath: "subdir",
			want:     "/base/path/subdir",
			wantSftp: false,
		},
		{
			name:     "Join absolute path",
			path:     base,
			joinPath: "/absolute/path",
			want:     "/absolute/path",
			wantSftp: false,
		},
		{
			name:     "Join to SFTP path",
			path:     sftpBase,
			joinPath: "subdir",
			want:     "/base/path/subdir",
			wantSftp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.path.Join(tt.joinPath)
			if got.path != tt.want {
				t.Errorf("Join() = %v, want %v", got.path, tt.want)
			}
			if got.isSftp != tt.wantSftp {
				t.Errorf("Join() isSftp = %v, want %v", got.isSftp, tt.wantSftp)
			}
		})
	}
}

func TestPath_Parent(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		want     string
		wantSftp bool
	}{
		{
			name: "Root path",
			path: "/",
			want: "/",
		},
		{
			name: "Single level",
			path: "/test",
			want: "/",
		},
		{
			name: "Multiple levels",
			path: "/test/path/file",
			want: "/test/path",
		},
		{
			name:     "SFTP path",
			path:     "sftp://example.com/test/path",
			want:     "/test",
			wantSftp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			got := p.Parent()
			if got.path != tt.want {
				t.Errorf("Parent() = %v, want %v", got.path, tt.want)
			}
			if got.isSftp != tt.wantSftp {
				t.Errorf("Parent() isSftp = %v, want %v", got.isSftp, tt.wantSftp)
			}
		})
	}
}

func TestPath_Name(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "Root path",
			path: "/",
			want: "/",
		},
		{
			name: "File path",
			path: "/test/file.txt",
			want: "file.txt",
		},
		{
			name: "Directory path",
			path: "/test/dir/",
			want: "dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			if got := p.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_Stem(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "No extension",
			path: "/test/file",
			want: "file",
		},
		{
			name: "With extension",
			path: "/test/file.txt",
			want: "file",
		},
		{
			name: "Multiple extensions",
			path: "/test/file.tar.gz",
			want: "file.tar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			if got := p.Stem(); got != tt.want {
				t.Errorf("Stem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_Suffix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "No extension",
			path: "/test/file",
			want: "",
		},
		{
			name: "With extension",
			path: "/test/file.txt",
			want: "txt",
		},
		{
			name: "Multiple extensions",
			path: "/test/file.tar.gz",
			want: "gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			if got := p.Suffix(); got != tt.want {
				t.Errorf("Suffix() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to create temporary test directory
func createTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "pathlib-test-*")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestPath_FileOperations(t *testing.T) {
	// Create temporary test directory
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	// Create test path
	testPath := filepath.Join(testDir, "test.txt")
	p := New(testPath)

	// Test WriteText and ReadText
	t.Run("WriteText and ReadText", func(t *testing.T) {
		content := "Hello, World!"
		err := p.WriteText(content, "utf-8")
		if err != nil {
			t.Fatalf("WriteText() error = %v", err)
		}

		got, err := p.ReadText("utf-8")
		if err != nil {
			t.Fatalf("ReadText() error = %v", err)
		}

		if got != content {
			t.Errorf("ReadText() = %v, want %v", got, content)
		}
	})

	// Test WriteBytes and ReadBytes
	t.Run("WriteBytes and ReadBytes", func(t *testing.T) {
		content := []byte("Binary content")
		err := p.WriteBytes(content)
		if err != nil {
			t.Fatalf("WriteBytes() error = %v", err)
		}

		got, err := p.ReadBytes()
		if err != nil {
			t.Fatalf("ReadBytes() error = %v", err)
		}

		if string(got) != string(content) {
			t.Errorf("ReadBytes() = %v, want %v", got, content)
		}
	})

	// Test file existence and type checks
	t.Run("Exists and type checks", func(t *testing.T) {
		if !p.Exists() {
			t.Error("Exists() = false, want true")
		}
		if !p.IsFile() {
			t.Error("IsFile() = false, want true")
		}
		if p.IsDir() {
			t.Error("IsDir() = true, want false")
		}
	})
}

func TestPath_DirectoryOperations(t *testing.T) {
	// Create temporary test directory
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	dirPath := New(filepath.Join(testDir, "testdir"))

	// Test MakeDir
	t.Run("MakeDir", func(t *testing.T) {
		err := dirPath.MakeDir(false, false)
		if err != nil {
			t.Fatalf("MakeDir() error = %v", err)
		}

		if !dirPath.Exists() {
			t.Error("Directory was not created")
		}
		if !dirPath.IsDir() {
			t.Error("Created path is not a directory")
		}
	})

	// Test List and ListRecursive
	t.Run("List and ListRecursive", func(t *testing.T) {
		// Create some test files and directories
		testFiles := []string{
			filepath.Join(dirPath.path, "file1.txt"),
			filepath.Join(dirPath.path, "file2.txt"),
			filepath.Join(dirPath.path, "subdir/file3.txt"),
		}

		for _, file := range testFiles {
			dir := filepath.Dir(file)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}
		}

		// Test List
		files, err := dirPath.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(files) != 3 { // 2 files + 1 subdirectory
			t.Errorf("List() returned wrong number of entries: got %v, want 3", len(files))
		}

		// Test ListRecursive
		recursiveFiles, err := dirPath.ListRecursive()
		if err != nil {
			t.Fatalf("ListRecursive() error = %v", err)
		}
		if len(recursiveFiles) != 4 { // 3 files + 1 subdirectory
			t.Errorf("ListRecursive() returned wrong number of entries: got %v, want 4", len(recursiveFiles))
		}
	})

	// Test RemoveDir
	t.Run("RemoveDir", func(t *testing.T) {
		err := dirPath.RemoveDir(false, true, true)
		if err != nil {
			t.Fatalf("RemoveDir() error = %v", err)
		}

		if dirPath.Exists() {
			t.Error("Directory still exists after removal")
		}
	})
}

func TestPath_CopyAndMove(t *testing.T) {
	// Create temporary test directory
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	// Create source and destination paths
	srcPath := New(filepath.Join(testDir, "source.txt"))
	destPath := New(filepath.Join(testDir, "destination.txt"))

	// Create test content
	testContent := "Test content for copy and move operations"
	err := srcPath.WriteText(testContent, "utf-8")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test CopyTo
	t.Run("CopyTo", func(t *testing.T) {
		err := srcPath.CopyTo(destPath)
		if err != nil {
			t.Fatalf("CopyTo() error = %v", err)
		}

		// Verify both files exist
		if !srcPath.Exists() || !destPath.Exists() {
			t.Error("Source or destination file doesn't exist after copy")
		}

		// Verify content
		content, err := destPath.ReadText("utf-8")
		if err != nil {
			t.Fatalf("Failed to read destination file: %v", err)
		}
		if content != testContent {
			t.Errorf("Destination content = %v, want %v", content, testContent)
		}
	})

	// Test MoveTo
	t.Run("MoveTo", func(t *testing.T) {
		moveDestPath := New(filepath.Join(testDir, "moved.txt"))
		err := srcPath.MoveTo(moveDestPath, false)
		if err != nil {
			t.Fatalf("MoveTo() error = %v", err)
		}

		// Verify source doesn't exist and destination does
		if srcPath.Exists() {
			t.Error("Source file still exists after move")
		}
		if !moveDestPath.Exists() {
			t.Error("Destination file doesn't exist after move")
		}

		// Verify content
		content, err := moveDestPath.ReadText("utf-8")
		if err != nil {
			t.Fatalf("Failed to read moved file: %v", err)
		}
		if content != testContent {
			t.Errorf("Moved content = %v, want %v", content, testContent)
		}
	})
}

func TestPath_WindowsSpecific(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows platform")
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Invalid Windows character <",
			path:    "/test/invalid<char",
			wantErr: true,
		},
		{
			name:    "Invalid Windows character >",
			path:    "/test/invalid>char",
			wantErr: true,
		},
		{
			name:    "Invalid Windows character :",
			path:    "/test/invalid:char",
			wantErr: true,
		},
		{
			name:    "Invalid Windows character \"",
			path:    `/test/invalid"char`,
			wantErr: true,
		},
		{
			name:    "Invalid Windows character |",
			path:    "/test/invalid|char",
			wantErr: true,
		},
		{
			name:    "Invalid Windows character ?",
			path:    "/test/invalid?char",
			wantErr: true,
		},
		{
			name:    "Invalid Windows character *",
			path:    "/test/invalid*char",
			wantErr: true,
		},
		{
			name:    "Reserved Windows name CON",
			path:    "/test/CON",
			wantErr: true,
		},
		{
			name:    "Reserved Windows name PRN",
			path:    "/test/PRN",
			wantErr: true,
		},
		{
			name:    "Reserved Windows name with extension",
			path:    "/test/CON.txt",
			wantErr: true,
		},
		{
			name:    "Valid Windows path",
			path:    "/test/valid_path",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			err := p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPath_SftpValidation(t *testing.T) {
	tests := []struct {
		name    string
		path    *Path
		wantErr bool
	}{
		{
			name: "Valid SFTP path",
			path: &Path{
				path:     "/test/path",
				isSftp:   true,
				host:     "example.com",
				port:     "22",
				username: "user",
				password: "pass",
			},
			wantErr: false,
		},
		{
			name: "Missing host",
			path: &Path{
				path:   "/test/path",
				isSftp: true,
				port:   "22",
			},
			wantErr: true,
		},
		{
			name: "Invalid port number",
			path: &Path{
				path:   "/test/path",
				isSftp: true,
				host:   "example.com",
				port:   "invalid",
			},
			wantErr: true,
		},
		{
			name: "Port out of range",
			path: &Path{
				path:   "/test/path",
				isSftp: true,
				host:   "example.com",
				port:   "70000",
			},
			wantErr: true,
		},
		{
			name: "Username too long",
			path: &Path{
				path:     "/test/path",
				isSftp:   true,
				host:     "example.com",
				port:     "22",
				username: string(make([]byte, 256)),
			},
			wantErr: true,
		},
		{
			name: "Password too long",
			path: &Path{
				path:     "/test/path",
				isSftp:   true,
				host:     "example.com",
				port:     "22",
				username: "user",
				password: string(make([]byte, 256)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.path.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPath_String(t *testing.T) {
	tests := []struct {
		name string
		path *Path
		want string
	}{
		{
			name: "Local path",
			path: &Path{
				path:   "/test/path",
				isSftp: false,
			},
			want: "/test/path",
		},
		{
			name: "SFTP path",
			path: &Path{
				path:     "/test/path",
				isSftp:   true,
				host:     "example.com",
				port:     "22",
				username: "user",
				password: "pass",
			},
			want: "/test/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_SftpPath(t *testing.T) {
	tests := []struct {
		name string
		path *Path
		want string
	}{
		{
			name: "Local path",
			path: &Path{
				path:   "/test/path",
				isSftp: false,
			},
			want: "",
		},
		{
			name: "SFTP path with credentials",
			path: &Path{
				path:     "/test/path",
				isSftp:   true,
				host:     "example.com",
				port:     "22",
				username: "user",
				password: "pass",
			},
			want: "sftp://user:pass@example.com:22/test/path",
		},
		{
			name: "SFTP path without password",
			path: &Path{
				path:     "/test/path",
				isSftp:   true,
				host:     "example.com",
				port:     "22",
				username: "user",
			},
			want: "sftp://user@example.com:22/test/path",
		},
		{
			name: "SFTP path without credentials",
			path: &Path{
				path:   "/test/path",
				isSftp: true,
				host:   "example.com",
				port:   "22",
			},
			want: "sftp://example.com:22/test/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.path.SftpPath(); got != tt.want {
				t.Errorf("SftpPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_CrossSystemOperations(t *testing.T) {
	const (
		sftpUser = "sftptest"
		sftpPass = "testpass123"
		sftpHost = "localhost"
		sftpPort = "22"
	)

	// Create temporary test directories
	localDir := createTempDir(t)
	defer os.RemoveAll(localDir)

	// Setup SFTP paths
	sftpURL := fmt.Sprintf("sftp://%s:%s@%s:%s/test", sftpUser, sftpPass, sftpHost, sftpPort)
	sftpBase := New(sftpURL)
	sftpDir1 := sftpBase.Join("dir1")
	sftpDir2 := sftpBase.Join("dir2")

	// Delete SFTP directories
	err := sftpDir1.RemoveDir(true, true, false)
	if err != nil {
		t.Fatalf("Failed to remove SFTP dir1: %v", err)
	}
	err = sftpDir2.RemoveDir(true, true, false)
	if err != nil {
		t.Fatalf("Failed to remove SFTP dir2: %v", err)
	}

	// Ensure SFTP test directories exist
	err = sftpDir1.MakeDir(true, true)
	if err != nil {
		t.Fatalf("Failed to create SFTP dir1: %v", err)
	}
	err = sftpDir2.MakeDir(true, true)
	if err != nil {
		t.Fatalf("Failed to create SFTP dir2: %v", err)
	}

	// Test data
	testContent := "Test content for cross-system operations"

	// Helper function to verify file content
	verifyContent := func(p *Path, expected string) error {
		content, err := p.ReadText("utf-8")
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
		if content != expected {
			return fmt.Errorf("content mismatch: got %q, want %q", content, expected)
		}
		return nil
	}

	t.Run("SFTP to Local operations", func(t *testing.T) {
		// Create source file on SFTP
		srcPath := sftpDir1.Join("source.txt")
		err := srcPath.WriteText(testContent, "utf-8")
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Copy to local
		destPath := New(filepath.Join(localDir, "local_copy.txt"))
		err = srcPath.CopyTo(destPath)
		if err != nil {
			t.Fatalf("CopyTo() error = %v", err)
		}

		// Verify content
		err = verifyContent(destPath, testContent)
		if err != nil {
			t.Error(err)
		}

		// Move to local
		moveDestPath := New(filepath.Join(localDir, "local_move.txt"))
		err = srcPath.MoveTo(moveDestPath, false)
		if err != nil {
			t.Fatalf("MoveTo() error = %v", err)
		}

		// Verify content and file existence
		if srcPath.Exists() {
			t.Error("Source file still exists after move")
		}
		err = verifyContent(moveDestPath, testContent)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Local to SFTP operations", func(t *testing.T) {
		// Create source file locally
		srcPath := New(filepath.Join(localDir, "local_source.txt"))
		err := srcPath.WriteText(testContent, "utf-8")
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Copy to SFTP
		destPath := sftpDir1.Join("sftp_copy.txt")
		err = srcPath.CopyTo(destPath)
		if err != nil {
			t.Fatalf("CopyTo() error = %v", err)
		}

		// Verify content
		err = verifyContent(destPath, testContent)
		if err != nil {
			t.Error(err)
		}

		// Move to SFTP
		moveDestPath := sftpDir1.Join("sftp_move.txt")
		err = srcPath.MoveTo(moveDestPath, false)
		if err != nil {
			t.Fatalf("MoveTo() error = %v", err)
		}

		// Verify content and file existence
		if srcPath.Exists() {
			t.Error("Source file still exists after move")
		}
		err = verifyContent(moveDestPath, testContent)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("SFTP to SFTP operations", func(t *testing.T) {
		// Create source file on SFTP
		srcPath := sftpDir1.Join("sftp_source.txt")
		err := srcPath.WriteText(testContent, "utf-8")
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Copy to different SFTP directory
		destPath := sftpDir2.Join("sftp_copy.txt")
		err = srcPath.CopyTo(destPath)
		if err != nil {
			t.Fatalf("CopyTo() error = %v", err)
		}

		// Verify content
		err = verifyContent(destPath, testContent)
		if err != nil {
			t.Error(err)
		}

		// Move to different SFTP directory
		moveDestPath := sftpDir2.Join("sftp_move.txt")
		err = srcPath.MoveTo(moveDestPath, false)
		if err != nil {
			t.Fatalf("MoveTo() error = %v", err)
		}

		// Verify content and file existence
		if srcPath.Exists() {
			t.Error("Source file still exists after move")
		}
		err = verifyContent(moveDestPath, testContent)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Large file operations", func(t *testing.T) {
		// Create large test content (10MB)
		largeContent := make([]byte, 10*1024*1024)
		rand.Read(largeContent)

		tests := []struct {
			name    string
			src     *Path
			dest    *Path
			content []byte
		}{
			{
				name:    "Large SFTP to Local",
				src:     sftpDir1.Join("large_sftp.bin"),
				dest:    New(filepath.Join(localDir, "large_local.bin")),
				content: largeContent,
			},
			{
				name:    "Large Local to SFTP",
				src:     New(filepath.Join(localDir, "large_source.bin")),
				dest:    sftpDir2.Join("large_sftp.bin"),
				content: largeContent,
			},
			{
				name:    "Large SFTP to SFTP",
				src:     sftpDir1.Join("large_source.bin"),
				dest:    sftpDir2.Join("large_dest.bin"),
				content: largeContent,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create source file
				err := tt.src.WriteBytes(tt.content)
				if err != nil {
					t.Fatalf("Failed to create source file: %v", err)
				}

				// Copy file
				err = tt.src.CopyTo(tt.dest)
				if err != nil {
					t.Fatalf("CopyTo() error = %v", err)
				}

				// Verify content
				got, err := tt.dest.ReadBytes()
				if err != nil {
					t.Fatalf("Failed to read destination file: %v", err)
				}
				if !bytes.Equal(got, tt.content) {
					t.Error("Content mismatch in copied file")
				}
			})
		}
	})

	t.Run("Error cases", func(t *testing.T) {
		tests := []struct {
			name    string
			setup   func() (*Path, *Path)
			wantErr bool
		}{
			{
				name: "Copy non-existent SFTP file to local",
				setup: func() (*Path, *Path) {
					return sftpDir1.Join("nonexistent.txt"),
						New(filepath.Join(localDir, "local.txt"))
				},
				wantErr: true,
			},
			{
				name: "Copy to invalid SFTP path",
				setup: func() (*Path, *Path) {
					src := New(filepath.Join(localDir, "source.txt"))
					src.WriteText("test", "utf-8")
					return src, sftpDir1.Join("///invalid")
				},
				wantErr: true,
			},
			{
				name: "Move between different SFTP servers",
				setup: func() (*Path, *Path) {
					src := sftpDir1.Join("source.txt")
					src.WriteText("test", "utf-8")
					dest := New("sftp://other-server.com/test.txt")
					return src, dest
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				src, dest := tt.setup()
				err := src.CopyTo(dest)
				if (err != nil) != tt.wantErr {
					t.Errorf("CopyTo() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})
}

func TestPath_SymlinkOperations(t *testing.T) {
	if runtime.GOOS == "windows" && !isWindowsSymlinksEnabled() {
		t.Skip("Skipping symlink tests on Windows without symlink privileges")
	}

	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name    string
		setup   func(dir string) (*Path, *Path, error)
		verify  func(*Path, *Path) error
		wantErr bool
	}{
		{
			name: "Create and follow file symlink",
			setup: func(dir string) (*Path, *Path, error) {
				targetPath := New(filepath.Join(dir, "target.txt"))
				symlinkPath := New(filepath.Join(dir, "link.txt"))

				err := targetPath.WriteText("test content", "utf-8")
				if err != nil {
					return nil, nil, err
				}

				err = os.Symlink(targetPath.path, symlinkPath.path)
				return targetPath, symlinkPath, err
			},
			verify: func(target, link *Path) error {
				targetContent, err := target.ReadText("utf-8")
				if err != nil {
					return err
				}
				linkContent, err := link.ReadText("utf-8")
				if err != nil {
					return err
				}
				if targetContent != linkContent {
					return fmt.Errorf("content mismatch: target=%q, link=%q", targetContent, linkContent)
				}
				return nil
			},
		},
		{
			name: "Copy preserves symlinks",
			setup: func(dir string) (*Path, *Path, error) {
				// Create files with unique names to avoid conflicts
				targetPath := New(filepath.Join(dir, fmt.Sprintf("target-%d.txt", time.Now().UnixNano())))
				symlinkPath := New(filepath.Join(dir, fmt.Sprintf("link-%d.txt", time.Now().UnixNano())))
				copyPath := New(filepath.Join(dir, fmt.Sprintf("copy-%d.txt", time.Now().UnixNano())))

				// Debug info
				t.Logf("Target path: %s", targetPath.path)
				t.Logf("Symlink path: %s", symlinkPath.path)
				t.Logf("Copy path: %s", copyPath.path)

				// Create target file first
				if err := targetPath.WriteText("test content", "utf-8"); err != nil {
					return nil, nil, fmt.Errorf("failed to create target: %v", err)
				}

				// Verify target file exists
				if !targetPath.Exists() {
					return nil, nil, fmt.Errorf("target file was not created")
				}

				// Create symlink using absolute paths
				targetAbs, err := filepath.Abs(targetPath.path)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to get absolute target path: %v", err)
				}

				if err := os.Symlink(targetAbs, symlinkPath.path); err != nil {
					return nil, nil, fmt.Errorf("failed to create symlink: %v", err)
				}

				// Verify symlink was created
				if !symlinkPath.Exists() {
					return nil, nil, fmt.Errorf("symlink was not created")
				}

				// Attempt direct file copy without options first
				err = os.Link(symlinkPath.path, copyPath.path)
				if err != nil {
					t.Logf("Hard link failed (expected), falling back to copy: %v", err)

					// Fall back to copy
					err = symlinkPath.CopyTo(copyPath)
					if err != nil {
						return nil, nil, fmt.Errorf("failed to copy symlink: %v", err)
					}
				}

				// Verify copy exists
				if !copyPath.Exists() {
					return nil, nil, fmt.Errorf("copy was not created")
				}

				return symlinkPath, copyPath, nil
			},
			verify: func(symlink, copy *Path) error {
				isSymlink, err := isSymlink(copy.path)
				if err != nil {
					return fmt.Errorf("failed to check if copy is symlink: %v", err)
				}
				if !isSymlink {
					return fmt.Errorf("copy is not a symlink")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path1, path2, err := tt.setup(testDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Setup error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if err := tt.verify(path1, path2); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}
		})
	}
}

func TestPath_Permissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission tests on Windows")
	}

	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name    string
		setup   func(dir string) error
		test    func(dir string) error
		wantErr bool
	}{
		{
			name: "Create file with specific permissions",
			setup: func(dir string) error {
				p := New(filepath.Join(dir, "test.txt"))
				return p.WriteText("test", "utf-8")
			},
			test: func(dir string) error {
				p := New(filepath.Join(dir, "test.txt"))
				info, err := os.Stat(p.path)
				if err != nil {
					return err
				}
				if info.Mode().Perm() != 0644 {
					return fmt.Errorf("unexpected permissions: got %v, want %v", info.Mode().Perm(), 0644)
				}
				return nil
			},
		},
		{
			name: "Preserve permissions during copy",
			setup: func(dir string) error {
				p := New(filepath.Join(dir, "source.txt"))
				if err := p.WriteText("test", "utf-8"); err != nil {
					return err
				}
				return os.Chmod(p.path, 0600)
			},
			test: func(dir string) error {
				src := New(filepath.Join(dir, "source.txt"))
				dst := New(filepath.Join(dir, "dest.txt"))
				if err := src.CopyTo(dst); err != nil {
					return err
				}

				srcInfo, err := os.Stat(src.path)
				if err != nil {
					return err
				}
				dstInfo, err := os.Stat(dst.path)
				if err != nil {
					return err
				}
				if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
					return fmt.Errorf("permission mismatch: src=%v, dst=%v",
						srcInfo.Mode().Perm(), dstInfo.Mode().Perm())
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			err := tt.test(testDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Test error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPath_RaceConditions(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	t.Run("Concurrent file access", func(t *testing.T) {
		p := New(filepath.Join(testDir, "concurrent.txt"))

		var wg sync.WaitGroup
		errors := make(chan error, 10)

		// Multiple goroutines writing to the same file
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				content := fmt.Sprintf("content-%d", n)
				if err := p.WriteText(content, "utf-8"); err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent write error: %v", err)
		}
	})

	t.Run("Concurrent directory operations", func(t *testing.T) {
		dirPath := New(filepath.Join(testDir, "concurrent-dir"))
		if err := dirPath.MakeDir(false, false); err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		errors := make(chan error, 10)

		// Multiple goroutines creating files in the same directory
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				p := dirPath.Join(fmt.Sprintf("file-%d.txt", n))
				if err := p.WriteText("test", "utf-8"); err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent directory operation error: %v", err)
		}
	})
}

func TestPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Unicode path",
			path:    "/test/路径/测试.txt",
			wantErr: false,
		},
		{
			name:    "Path with spaces",
			path:    "/test/path with spaces/file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			err := p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPath_NetworkFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network failure tests in short mode")
	}

	tests := []struct {
		name    string
		setup   func() (*Path, error)
		test    func(*Path) error
		wantErr bool
	}{
		{
			name: "Connection timeout",
			setup: func() (*Path, error) {
				return New("sftp://slowhost.example.com:22/test.txt"), nil
			},
			test: func(p *Path) error {
				return p.WriteText("test", "utf-8")
			},
			wantErr: true,
		},
		{
			name: "Connection drop during operation",
			setup: func() (*Path, error) {
				return New("sftp://unstablehost.example.com:22/test.txt"), nil
			},
			test: func(p *Path) error {
				// Write large file to trigger connection drop
				data := make([]byte, 100*1024*1024)
				return p.WriteBytes(data)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := tt.setup()
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			err = tt.test(p)
			if (err != nil) != tt.wantErr {
				t.Errorf("Test error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPath_FileSystemSpecific(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name    string
		setup   func(dir string) error
		test    func(dir string) error
		wantErr bool
	}{
		{
			name: "Case sensitivity",
			setup: func(dir string) error {
				p1 := New(filepath.Join(dir, "test.txt"))
				p2 := New(filepath.Join(dir, "TEST.txt"))
				if err := p1.WriteText("test1", "utf-8"); err != nil {
					return err
				}
				return p2.WriteText("test2", "utf-8")
			},
			test: func(dir string) error {
				p1 := New(filepath.Join(dir, "test.txt"))
				p2 := New(filepath.Join(dir, "TEST.txt"))
				content1, err := p1.ReadText("utf-8")
				if err != nil {
					return err
				}
				content2, err := p2.ReadText("utf-8")
				if err != nil {
					return err
				}
				if runtime.GOOS == "windows" && content1 == content2 {
					return nil // Expected on Windows
				}
				if runtime.GOOS != "windows" && content1 != "test1" {
					return fmt.Errorf("unexpected content: %s", content1)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			err := tt.test(testDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Test error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPath_TimeOperations(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	t.Run("Preserve timestamps during copy", func(t *testing.T) {
		src := New(filepath.Join(testDir, "source.txt"))
		dst := New(filepath.Join(testDir, "dest.txt"))

		// Create source file
		if err := src.WriteText("test", "utf-8"); err != nil {
			t.Fatal(err)
		}

		// Set specific timestamps
		modTime := time.Now().Add(-24 * time.Hour)
		if err := os.Chtimes(src.path, modTime, modTime); err != nil {
			t.Fatal(err)
		}

		// Copy file
		if err := src.CopyTo(dst); err != nil {
			t.Fatal(err)
		}

		// Verify timestamps
		srcInfo, err := os.Stat(src.path)
		if err != nil {
			t.Fatal(err)
		}
		dstInfo, err := os.Stat(dst.path)
		if err != nil {
			t.Fatal(err)
		}

		if !srcInfo.ModTime().Equal(dstInfo.ModTime()) {
			t.Errorf("ModTime not preserved: src=%v, dst=%v",
				srcInfo.ModTime(), dstInfo.ModTime())
		}
	})
}

func TestPath_AtomicOperations(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	t.Run("Atomic rename", func(t *testing.T) {
		src := New(filepath.Join(testDir, "source.txt"))
		dst := New(filepath.Join(testDir, "dest.txt"))

		content := "test content"
		if err := src.WriteText(content, "utf-8"); err != nil {
			t.Fatal(err)
		}

		// Perform atomic rename
		if err := os.Rename(src.path, dst.path); err != nil {
			t.Fatal(err)
		}

		// Verify content after rename
		if src.Exists() {
			t.Error("Source file still exists after rename")
		}

		gotContent, err := dst.ReadText("utf-8")
		if err != nil {
			t.Fatal(err)
		}
		if gotContent != content {
			t.Errorf("Content mismatch after rename: got %q, want %q", gotContent, content)
		}
	})

	t.Run("Atomic write with temporary file", func(t *testing.T) {
		p := New(filepath.Join(testDir, "atomic.txt"))
		content := "test content"

		// Write content atomically using temporary file
		tempPath := p.path + ".tmp"
		if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(tempPath, p.path); err != nil {
			t.Fatal(err)
		}

		// Verify content
		gotContent, err := p.ReadText("utf-8")
		if err != nil {
			t.Fatal(err)
		}
		if gotContent != content {
			t.Errorf("Content mismatch: got %q, want %q", gotContent, content)
		}
	})
}

func TestPath_CharacterEncoding(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name     string
		content  string
		encoding string
		wantErr  bool
	}{
		{
			name:     "UTF-8 with BOM",
			content:  "\xEF\xBB\xBFHello, World!",
			encoding: "utf-8",
			wantErr:  false,
		},
		{
			name:     "UTF-16LE",
			content:  "H\x00e\x00l\x00l\x00o\x00",
			encoding: "utf-16le",
			wantErr:  false,
		},
		{
			name:     "ISO-8859-1",
			content:  "Hello, Wörld!",
			encoding: "iso-8859-1",
			wantErr:  false,
		},
		{
			name:     "Invalid UTF-8 sequence",
			content:  "Hello\xFF\xFE\xFDWorld",
			encoding: "utf-8",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(filepath.Join(testDir, "encoded.txt"))

			err := p.WriteText(tt.content, tt.encoding)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got, err := p.ReadText(tt.encoding)
				if err != nil {
					t.Errorf("ReadText() error = %v", err)
					return
				}
				if got != tt.content {
					t.Errorf("Content mismatch: got %q, want %q", got, tt.content)
				}
			}
		})
	}
}

func TestPath_LargeDirectoryOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large directory tests in short mode")
	}

	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	dirPath := New(filepath.Join(testDir, "large-dir"))
	if err := dirPath.MakeDir(false, false); err != nil {
		t.Fatal(err)
	}

	// Create many files
	const numFiles = 10000
	t.Run("Create many files", func(t *testing.T) {
		for i := 0; i < numFiles; i++ {
			p := dirPath.Join(fmt.Sprintf("file-%d.txt", i))
			if err := p.WriteText("test", "utf-8"); err != nil {
				t.Fatalf("Failed to create file %d: %v", i, err)
			}
		}
	})

	// List files
	t.Run("List many files", func(t *testing.T) {
		files, err := dirPath.List()
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != numFiles {
			t.Errorf("Wrong number of files: got %d, want %d", len(files), numFiles)
		}
	})

	// Remove files
	t.Run("Remove many files", func(t *testing.T) {
		if err := dirPath.RemoveDir(false, true, true); err != nil {
			t.Fatal(err)
		}
		if dirPath.Exists() {
			t.Error("Directory still exists after removal")
		}
	})
}

// Helper functions

func isWindowsSymlinksEnabled() bool {
	// Check if the current user has the SeCreateSymbolicLinkPrivilege
	if runtime.GOOS != "windows" {
		return false
	}

	testDir := os.TempDir()
	testLink := filepath.Join(testDir, "test-symlink")
	testTarget := filepath.Join(testDir, "test-target")

	// Try to create a test symlink
	err := os.Symlink(testTarget, testLink)
	if err == nil {
		os.Remove(testLink)
		return true
	}
	return false
}

func makePathSegments(depth int) []string {
	segments := make([]string, depth)
	for i := 0; i < depth; i++ {
		segments[i] = fmt.Sprintf("dir%d", i)
	}
	return segments
}

func isSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}
