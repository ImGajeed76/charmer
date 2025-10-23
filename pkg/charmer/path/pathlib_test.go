package path

import (
	"bytes"
	"errors"
	"fmt"
	pathmodels "github.com/ImGajeed76/charmer/pkg/charmer/path/models"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// ==================== Test Configuration ====================

const (
	sftpTestUser = "testuser"
	sftpTestPass = "testpass"
	sftpTestHost = "localhost"
	sftpTestPort = "2222"
)

// getSFTPTestPath returns a new SFTP path for testing
func getSFTPTestPath(subpath string) *Path {
	url := fmt.Sprintf("sftp://%s:%s@%s:%s/config/upload/%s",
		sftpTestUser, sftpTestPass, sftpTestHost, sftpTestPort, subpath)
	return New(url)
}

// createTempDir creates a temporary test directory
func createTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "pathlib-test-*")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// cleanupSFTPTestDir removes SFTP test directory
func cleanupSFTPTestDir(t *testing.T, path *Path) {
	if err := path.RemoveDir(true, true, false); err != nil {
		t.Logf("Warning: failed to cleanup SFTP directory: %v", err)
	}
}

// isSFTPAvailable checks if SFTP server is available
func isSFTPAvailable() bool {
	testPath := getSFTPTestPath("test-connection")
	defer testPath.RemoveDir(true, true, false)

	if err := testPath.MakeDir(true, true); err != nil {
		return false
	}
	return true
}

// ==================== Docker SFTP Setup ====================

var (
	dockerComposeFile  = "../../../docker-sftp/docker-compose.yml"
	sftpContainerName  = "charmer-sftp-testing"
	weStartedContainer = false
)

func init() {
	// Setup SFTP container for testing
	setupSFTPContainer()
}

func setupSFTPContainer() {
	if runtime.GOOS == "windows" {
		log.Println("Skipping SFTP container setup on Windows (Docker networking limitations)")
		return
	}

	// Check if docker-compose is available
	if !isDockerComposeAvailable() {
		log.Println("Docker Compose not available, SFTP tests will be skipped")
		return
	}

	// Get absolute path to docker-compose file
	absPath, err := filepath.Abs(dockerComposeFile)
	if err != nil {
		log.Printf("Failed to get absolute path for docker-compose file: %v\n", err)
		return
	}

	// Check if container is already running
	if isContainerRunning() {
		log.Println("SFTP container already running")
		return
	}

	// Start the container
	log.Println("Starting SFTP container for testing...")
	var cmd *exec.Cmd
	if exec.Command("docker", "compose", "version").Run() == nil {
		cmd = exec.Command("docker", "compose", "-f", absPath, "up", "-d")
	} else {
		cmd = exec.Command("docker-compose", "-f", absPath, "up", "-d")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to start SFTP container: %v\nOutput: %s\n", err, output)
		return
	}

	weStartedContainer = true
	log.Println("SFTP container started successfully")

	// Wait for container to be ready
	waitForSFTP()
}

func teardownSFTPContainer() {
	if !weStartedContainer {
		return
	}

	log.Println("Stopping SFTP container...")
	absPath, err := filepath.Abs(dockerComposeFile)
	if err != nil {
		log.Printf("Failed to get absolute path for docker-compose file: %v\n", err)
		return
	}

	var cmd *exec.Cmd
	if exec.Command("docker", "compose", "version").Run() == nil {
		cmd = exec.Command("docker", "compose", "-f", absPath, "down")
	} else {
		cmd = exec.Command("docker-compose", "-f", absPath, "down")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to stop SFTP container: %v\nOutput: %s\n", err, output)
		return
	}

	log.Println("SFTP container stopped successfully")
}

func isDockerComposeAvailable() bool {
	// Try docker-compose
	cmd := exec.Command("docker-compose", "version")
	if err := cmd.Run(); err == nil {
		return true
	}

	// Try docker compose (v2 syntax)
	cmd = exec.Command("docker", "compose", "version")
	return cmd.Run() == nil
}

func isContainerRunning() bool {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", sftpContainerName), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == sftpContainerName
}

func waitForSFTP() {
	// Wait up to 30 seconds for SFTP to be ready
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		if isSFTPAvailable() {
			log.Println("SFTP server is ready")
			return
		}
		time.Sleep(1 * time.Second)
	}
	log.Println("Warning: SFTP server may not be ready")
}

// ==================== Path Creation Tests ====================

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup
	teardownSFTPContainer()

	os.Exit(code)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		want            *Path
		wantNil         bool
		windowsSpecific bool
	}{
		{
			name:    "Empty path returns nil",
			path:    "",
			wantNil: true,
		},
		{
			name: "Absolute local path",
			path: "/test/path",
			want: &Path{
				path:   "/test/path",
				isSftp: false,
				isUrl:  false,
			},
		},
		{
			name: "Windows path with backslashes",
			path: "C:\\test\\path",
			want: &Path{
				path:   "C:/test/path",
				isSftp: false,
			},
			windowsSpecific: true,
		},
		{
			name: "SFTP URL with full credentials",
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
			name: "SFTP URL with username only",
			path: "sftp://user@example.com/test/path",
			want: &Path{
				path:     "/test/path",
				isSftp:   true,
				host:     "example.com",
				port:     "22",
				username: "user",
			},
		},
		{
			name: "SFTP URL without credentials",
			path: "sftp://example.com/test/path",
			want: &Path{
				path:   "/test/path",
				isSftp: true,
				host:   "example.com",
				port:   "22",
			},
		},
		{
			name: "HTTP URL",
			path: "http://example.com/test/file.txt",
			want: &Path{
				path:  "http://example.com/test/file.txt",
				isUrl: true,
			},
		},
		{
			name: "HTTPS URL",
			path: "https://raw.githubusercontent.com/user/repo/master/file.txt",
			want: &Path{
				path:  "https://raw.githubusercontent.com/user/repo/master/file.txt",
				isUrl: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.windowsSpecific && runtime.GOOS != "windows" {
				t.Skip("Skipping Windows-specific test")
			}

			got := New(tt.path)
			if tt.wantNil {
				if got != nil {
					t.Errorf("New() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("New() returned nil unexpectedly")
			}
			if got.path != tt.want.path {
				t.Errorf("path = %v, want %v", got.path, tt.want.path)
			}
			if got.isSftp != tt.want.isSftp {
				t.Errorf("isSftp = %v, want %v", got.isSftp, tt.want.isSftp)
			}
			if got.isUrl != tt.want.isUrl {
				t.Errorf("isUrl = %v, want %v", got.isUrl, tt.want.isUrl)
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

func TestNewWithSFTPConfig(t *testing.T) {
	config := &SFTPConfig{
		Host:     "example.com",
		Port:     "2222",
		Username: "testuser",
		Password: "testpass",
	}

	p := New("/test/path", config)
	if p == nil {
		t.Fatal("New() returned nil")
	}
	if !p.isSftp {
		t.Error("Expected SFTP path")
	}
	if p.host != config.Host {
		t.Errorf("host = %v, want %v", p.host, config.Host)
	}
	if p.port != config.Port {
		t.Errorf("port = %v, want %v", p.port, config.Port)
	}
	if p.username != config.Username {
		t.Errorf("username = %v, want %v", p.username, config.Username)
	}
	if p.password != config.Password {
		t.Errorf("password = %v, want %v", p.password, config.Password)
	}
}

func TestCwd(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	p := Cwd()
	if p == nil {
		t.Fatal("Cwd() returned nil")
	}
	if p.path != wd {
		t.Errorf("Cwd() = %v, want %v", p.path, wd)
	}
	if p.isSftp {
		t.Error("Cwd() should not be SFTP path")
	}
}

// ==================== Path Validation Tests ====================

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
			name: "Path exceeds maximum length",
			path: &Path{
				path: "/" + strings.Repeat("a", MaxPathLength),
			},
			wantErr: true,
		},
		{
			name: "Path with null byte",
			path: &Path{
				path: "/test/path\x00",
			},
			wantErr: true,
		},
		{
			name: "Path with control character",
			path: &Path{
				path: "/test/path\x01",
			},
			wantErr: true,
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
		{
			name: "SFTP path missing host",
			path: &Path{
				path:   "/test/path",
				isSftp: true,
				port:   "22",
			},
			wantErr: true,
		},
		{
			name: "SFTP path with invalid port",
			path: &Path{
				path:   "/test/path",
				isSftp: true,
				host:   "example.com",
				port:   "invalid",
			},
			wantErr: true,
		},
		{
			name: "SFTP path with port out of range",
			path: &Path{
				path:   "/test/path",
				isSftp: true,
				host:   "example.com",
				port:   "70000",
			},
			wantErr: true,
		},
		{
			name: "Valid URL path",
			path: &Path{
				path:  "https://example.com/file.txt",
				isUrl: true,
			},
			wantErr: false,
		},
		{
			name: "Path cannot be both SFTP and URL",
			path: &Path{
				path:   "/test",
				isSftp: true,
				isUrl:  true,
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

// ==================== Path Manipulation Tests ====================

func TestPath_Join(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		joinPath string
		want     string
		wantType string // "local", "sftp", "url"
	}{
		{
			name:     "Join empty string",
			basePath: "/base/path",
			joinPath: "",
			want:     "/base/path",
			wantType: "local",
		},
		{
			name:     "Join relative path",
			basePath: "/base/path",
			joinPath: "subdir",
			want:     "/base/path/subdir",
			wantType: "local",
		},
		{
			name:     "Join with backslashes",
			basePath: "/base/path",
			joinPath: "sub\\dir",
			want:     "/base/path/sub/dir",
			wantType: "local",
		},
		{
			name:     "Join to SFTP path",
			basePath: "sftp://example.com/base",
			joinPath: "subdir/file.txt",
			want:     "/base/subdir/file.txt",
			wantType: "sftp",
		},
		{
			name:     "Join to URL",
			basePath: "https://example.com/base",
			joinPath: "file.txt",
			want:     "https://example.com/base/file.txt",
			wantType: "url",
		},
		{
			name:     "Join URL with trailing slash",
			basePath: "https://example.com/base/",
			joinPath: "file.txt",
			want:     "https://example.com/base/file.txt",
			wantType: "url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := New(tt.basePath)
			got := base.Join(tt.joinPath)
			if got.path != tt.want {
				t.Errorf("Join() path = %v, want %v", got.path, tt.want)
			}
			switch tt.wantType {
			case "sftp":
				if !got.isSftp {
					t.Error("Expected SFTP path")
				}
			case "url":
				if !got.isUrl {
					t.Error("Expected URL path")
				}
			case "local":
				if got.isSftp || got.isUrl {
					t.Error("Expected local path")
				}
			}
		})
	}
}

func TestPath_Parent(t *testing.T) {
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
			name: "SFTP path",
			path: "sftp://example.com/test/path",
			want: "/test",
		},
		{
			name: "URL path",
			path: "https://example.com/path/to/file.txt",
			want: "https://example.com/path/to",
		},
		{
			name: "URL at root",
			path: "https://example.com/file.txt",
			want: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			got := p.Parent()
			if got.path != tt.want {
				t.Errorf("Parent() = %v, want %v", got.path, tt.want)
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
			name: "File with extension",
			path: "/test/file.txt",
			want: "file.txt",
		},
		{
			name: "Directory",
			path: "/test/dir/",
			want: "dir",
		},
		{
			name: "URL path",
			path: "https://example.com/path/file.txt",
			want: "file.txt",
		},
		{
			name: "Complex filename",
			path: "/path/file.tar.gz",
			want: "file.tar.gz",
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
			name: "Single extension",
			path: "/test/file.txt",
			want: "file",
		},
		{
			name: "Multiple extensions",
			path: "/test/file.tar.gz",
			want: "file.tar",
		},
		{
			name: "Hidden file",
			path: "/test/.gitignore",
			want: "",
		},
		{
			name: "Directory",
			path: "/test/dir/",
			want: "dir",
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
			name: "Single extension",
			path: "/test/file.txt",
			want: "txt",
		},
		{
			name: "Multiple extensions",
			path: "/test/file.tar.gz",
			want: "gz",
		},
		{
			name: "Hidden file with extension",
			path: "/test/.config.json",
			want: "json",
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

func TestPath_WithName(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		newName string
		want    string
	}{
		{
			name:    "Change filename",
			path:    "/test/file.txt",
			newName: "newfile.txt",
			want:    "/test/newfile.txt",
		},
		{
			name:    "Change with different extension",
			path:    "/test/file.txt",
			newName: "newfile.json",
			want:    "/test/newfile.json",
		},
		{
			name:    "SFTP path",
			path:    "sftp://example.com/test/file.txt",
			newName: "newfile.txt",
			want:    "/test/newfile.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			got := p.WithName(tt.newName)
			if got.path != tt.want {
				t.Errorf("WithName() = %v, want %v", got.path, tt.want)
			}
		})
	}
}

func TestPath_WithSuffix(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		suffix string
		want   string
	}{
		{
			name:   "Change extension",
			path:   "/test/file.txt",
			suffix: "json",
			want:   "/test/file.json",
		},
		{
			name:   "Change extension with dot",
			path:   "/test/file.txt",
			suffix: ".json",
			want:   "/test/file.json",
		},
		{
			name:   "Add extension",
			path:   "/test/file",
			suffix: "txt",
			want:   "/test/file.txt",
		},
		{
			name:   "Complex extension",
			path:   "/test/file.tar.gz",
			suffix: "bz2",
			want:   "/test/file.tar.bz2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			got := p.WithSuffix(tt.suffix)
			if got.path != tt.want {
				t.Errorf("WithSuffix() = %v, want %v", got.path, tt.want)
			}
		})
	}
}

func TestPath_Parts(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "Root path",
			path: "/",
			want: []string{"/"},
		},
		{
			name: "Simple path",
			path: "/test/path",
			want: []string{"/", "test", "path"},
		},
		{
			name: "URL path",
			path: "https://example.com/path/to/file",
			want: []string{"https://example.com", "path", "to", "file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			got := p.Parts()
			if len(got) != len(tt.want) {
				t.Errorf("Parts() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Parts()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPath_Match(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
		wantErr bool
	}{
		{
			name:    "Exact match",
			path:    "/test/file.txt",
			pattern: "file.txt",
			want:    true,
		},
		{
			name:    "Wildcard match",
			path:    "/test/file.txt",
			pattern: "*.txt",
			want:    true,
		},
		{
			name:    "No match",
			path:    "/test/file.txt",
			pattern: "*.json",
			want:    false,
		},
		{
			name:    "Question mark wildcard",
			path:    "/test/file1.txt",
			pattern: "file?.txt",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			got, err := p.Match(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("Match() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_IsAbsolute(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "Absolute path",
			path: "/test/path",
			want: true,
		},
		{
			name: "SFTP path always absolute",
			path: "sftp://example.com/test",
			want: true,
		},
		{
			name: "URL always absolute",
			path: "https://example.com/test",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			if got := p.IsAbsolute(); got != tt.want {
				t.Errorf("IsAbsolute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_IsRelative(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "Absolute path",
			path: "/test/path",
			want: false,
		},
		{
			name: "SFTP path never relative",
			path: "sftp://example.com/test",
			want: false,
		},
		{
			name: "URL never relative",
			path: "https://example.com/test",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.path)
			if got := p.IsRelative(); got != tt.want {
				t.Errorf("IsRelative() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_Copy(t *testing.T) {
	original := New("sftp://user:pass@example.com:2222/test/path")
	copy := original.Copy()

	if copy.path != original.path {
		t.Errorf("Copy() path = %v, want %v", copy.path, original.path)
	}
	if copy.isSftp != original.isSftp {
		t.Errorf("Copy() isSftp = %v, want %v", copy.isSftp, original.isSftp)
	}
	if copy.host != original.host {
		t.Errorf("Copy() host = %v, want %v", copy.host, original.host)
	}

	// Verify it's a deep copy
	copy.path = "/different/path"
	if original.path == copy.path {
		t.Error("Copy() did not create a deep copy")
	}
}

func TestPath_SetPath(t *testing.T) {
	tests := []struct {
		name    string
		initial *Path
		newPath string
		wantErr bool
	}{
		{
			name:    "Set new path",
			initial: New("/old/path"),
			newPath: "/new/path",
			wantErr: false,
		},
		{
			name:    "Cannot set empty path",
			initial: New("/old/path"),
			newPath: "",
			wantErr: true,
		},
		{
			name:    "Cannot change to SFTP URL",
			initial: New("/old/path"),
			newPath: "sftp://example.com/path",
			wantErr: true,
		},
		{
			name:    "Convert backslashes",
			initial: New("/old/path"),
			newPath: "new\\path",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.SetPath(tt.newPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ==================== File Operation Tests ====================

func TestPath_FileOperations(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	t.Run("WriteText and ReadText", func(t *testing.T) {
		p := New(filepath.Join(testDir, "test.txt"))
		content := "Hello, World! 你好世界"

		if err := p.WriteText(content, "utf-8"); err != nil {
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

	t.Run("WriteBytes and ReadBytes", func(t *testing.T) {
		p := New(filepath.Join(testDir, "binary.bin"))
		content := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}

		if err := p.WriteBytes(content); err != nil {
			t.Fatalf("WriteBytes() error = %v", err)
		}

		got, err := p.ReadBytes()
		if err != nil {
			t.Fatalf("ReadBytes() error = %v", err)
		}

		if !bytes.Equal(got, content) {
			t.Errorf("ReadBytes() = %v, want %v", got, content)
		}
	})

	t.Run("AppendText", func(t *testing.T) {
		p := New(filepath.Join(testDir, "append.txt"))

		if err := p.WriteText("Line 1\n", "utf-8"); err != nil {
			t.Fatal(err)
		}
		if err := p.AppendText("Line 2\n", "utf-8"); err != nil {
			t.Fatal(err)
		}

		got, err := p.ReadText("utf-8")
		if err != nil {
			t.Fatal(err)
		}

		want := "Line 1\nLine 2\n"
		if got != want {
			t.Errorf("AppendText() result = %v, want %v", got, want)
		}
	})

	t.Run("AppendBytes", func(t *testing.T) {
		p := New(filepath.Join(testDir, "append.bin"))

		data1 := []byte{0x01, 0x02}
		data2 := []byte{0x03, 0x04}

		if err := p.WriteBytes(data1); err != nil {
			t.Fatal(err)
		}
		if err := p.AppendBytes(data2); err != nil {
			t.Fatal(err)
		}

		got, err := p.ReadBytes()
		if err != nil {
			t.Fatal(err)
		}

		want := append(data1, data2...)
		if !bytes.Equal(got, want) {
			t.Errorf("AppendBytes() result = %v, want %v", got, want)
		}
	})

	t.Run("ReadLines and WriteLines", func(t *testing.T) {
		p := New(filepath.Join(testDir, "lines.txt"))
		lines := []string{"Line 1", "Line 2", "Line 3"}

		if err := p.WriteLines(lines, "utf-8"); err != nil {
			t.Fatal(err)
		}

		got, err := p.ReadLines("utf-8")
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != len(lines) {
			t.Fatalf("ReadLines() length = %v, want %v", len(got), len(lines))
		}

		for i := range lines {
			if got[i] != lines[i] {
				t.Errorf("ReadLines()[%d] = %v, want %v", i, got[i], lines[i])
			}
		}
	})
}

func TestPath_FileInfo(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	t.Run("Exists, IsFile, IsDir", func(t *testing.T) {
		filePath := New(filepath.Join(testDir, "file.txt"))
		dirPath := New(filepath.Join(testDir, "dir"))

		// File doesn't exist yet
		if filePath.Exists() {
			t.Error("File should not exist yet")
		}

		// Create file
		if err := filePath.WriteText("test", "utf-8"); err != nil {
			t.Fatal(err)
		}

		if !filePath.Exists() {
			t.Error("File should exist")
		}
		if !filePath.IsFile() {
			t.Error("Should be a file")
		}
		if filePath.IsDir() {
			t.Error("Should not be a directory")
		}

		// Create directory
		if err := dirPath.MakeDir(false, false); err != nil {
			t.Fatal(err)
		}

		if !dirPath.Exists() {
			t.Error("Directory should exist")
		}
		if !dirPath.IsDir() {
			t.Error("Should be a directory")
		}
		if dirPath.IsFile() {
			t.Error("Should not be a file")
		}
	})

	t.Run("Size", func(t *testing.T) {
		p := New(filepath.Join(testDir, "sized.txt"))
		content := "Hello, World!"

		if err := p.WriteText(content, "utf-8"); err != nil {
			t.Fatal(err)
		}

		size, err := p.Size()
		if err != nil {
			t.Fatal(err)
		}

		if size != int64(len(content)) {
			t.Errorf("Size() = %v, want %v", size, len(content))
		}
	})

	t.Run("IsEmpty", func(t *testing.T) {
		emptyFile := New(filepath.Join(testDir, "empty.txt"))
		nonEmptyFile := New(filepath.Join(testDir, "nonempty.txt"))
		emptyDir := New(filepath.Join(testDir, "emptydir"))

		// Empty file
		if err := emptyFile.WriteText("", "utf-8"); err != nil {
			t.Fatal(err)
		}
		isEmpty, err := emptyFile.IsEmpty()
		if err != nil {
			t.Fatal(err)
		}
		if !isEmpty {
			t.Error("Empty file should be empty")
		}

		// Non-empty file
		if err := nonEmptyFile.WriteText("content", "utf-8"); err != nil {
			t.Fatal(err)
		}
		isEmpty, err = nonEmptyFile.IsEmpty()
		if err != nil {
			t.Fatal(err)
		}
		if isEmpty {
			t.Error("Non-empty file should not be empty")
		}

		// Empty directory
		if err := emptyDir.MakeDir(false, false); err != nil {
			t.Fatal(err)
		}
		isEmpty, err = emptyDir.IsEmpty()
		if err != nil {
			t.Fatal(err)
		}
		if !isEmpty {
			t.Error("Empty directory should be empty")
		}
	})

	t.Run("Stat", func(t *testing.T) {
		p := New(filepath.Join(testDir, "stat.txt"))
		if err := p.WriteText("test", "utf-8"); err != nil {
			t.Fatal(err)
		}

		info, err := p.Stat()
		if err != nil {
			t.Fatal(err)
		}

		if info.Name != "stat.txt" {
			t.Errorf("Name = %v, want %v", info.Name, "stat.txt")
		}
		if info.IsDir {
			t.Error("Should not be a directory")
		}
		if info.Size != 4 {
			t.Errorf("Size = %v, want 4", info.Size)
		}
	})
}

// ==================== Directory Operation Tests ====================

func TestPath_DirectoryOperations(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	t.Run("MakeDir", func(t *testing.T) {
		dirPath := New(filepath.Join(testDir, "testdir"))

		if err := dirPath.MakeDir(false, false); err != nil {
			t.Fatal(err)
		}

		if !dirPath.Exists() || !dirPath.IsDir() {
			t.Error("Directory was not created")
		}

		// Try creating again with existsOk=false
		err := dirPath.MakeDir(false, false)
		if err == nil {
			t.Error("Should error when directory exists and existsOk=false")
		}

		// Try creating again with existsOk=true
		if err := dirPath.MakeDir(false, true); err != nil {
			t.Errorf("Should not error when existsOk=true: %v", err)
		}
	})

	t.Run("MakeDir with parents", func(t *testing.T) {
		deepPath := New(filepath.Join(testDir, "a/b/c/d"))

		if err := deepPath.MakeDir(true, false); err != nil {
			t.Fatal(err)
		}

		if !deepPath.Exists() || !deepPath.IsDir() {
			t.Error("Deep directory was not created")
		}
	})

	t.Run("List", func(t *testing.T) {
		dirPath := New(filepath.Join(testDir, "listdir"))
		if err := dirPath.MakeDir(true, true); err != nil {
			t.Fatal(err)
		}

		// Create test files
		for i := 1; i <= 3; i++ {
			p := dirPath.Join(fmt.Sprintf("file%d.txt", i))
			if err := p.WriteText("test", "utf-8"); err != nil {
				t.Fatal(err)
			}
		}

		files, err := dirPath.List()
		if err != nil {
			t.Fatal(err)
		}

		if len(files) != 3 {
			t.Errorf("List() returned %d files, want 3", len(files))
		}
	})

	t.Run("ListRecursive", func(t *testing.T) {
		dirPath := New(filepath.Join(testDir, "recursedir"))
		if err := dirPath.MakeDir(true, true); err != nil {
			t.Fatal(err)
		}

		// Create nested structure
		dirPath.Join("file1.txt").WriteText("test", "utf-8")
		dirPath.Join("subdir").MakeDir(true, true)
		dirPath.Join("subdir/file2.txt").WriteText("test", "utf-8")
		dirPath.Join("subdir/deeper").MakeDir(true, true)
		dirPath.Join("subdir/deeper/file3.txt").WriteText("test", "utf-8")

		files, err := dirPath.ListRecursive()
		if err != nil {
			t.Fatal(err)
		}

		// Should include: file1.txt, subdir (dir), subdir/file2.txt,
		// subdir/deeper (dir), subdir/deeper/file3.txt
		if len(files) < 3 {
			t.Errorf("ListRecursive() returned %d items, want at least 3", len(files))
		}
	})

	t.Run("Remove and RemoveDir", func(t *testing.T) {
		// Test Remove (file)
		filePath := New(filepath.Join(testDir, "remove.txt"))
		if err := filePath.WriteText("test", "utf-8"); err != nil {
			t.Fatal(err)
		}

		if err := filePath.Remove(false, false); err != nil {
			t.Fatal(err)
		}

		if filePath.Exists() {
			t.Error("File still exists after Remove()")
		}

		// Test RemoveDir (empty directory)
		emptyDir := New(filepath.Join(testDir, "emptyremove"))
		if err := emptyDir.MakeDir(true, true); err != nil {
			t.Fatal(err)
		}

		if err := emptyDir.RemoveDir(false, false, false); err != nil {
			t.Fatal(err)
		}

		if emptyDir.Exists() {
			t.Error("Directory still exists after RemoveDir()")
		}

		// Test RemoveDir with recursive
		nonEmptyDir := New(filepath.Join(testDir, "nonemptyremove"))
		nonEmptyDir.MakeDir(true, true)
		nonEmptyDir.Join("file.txt").WriteText("test", "utf-8")
		nonEmptyDir.Join("subdir").MakeDir(true, true)

		if err := nonEmptyDir.RemoveDir(false, true, false); err != nil {
			t.Fatal(err)
		}

		if nonEmptyDir.Exists() {
			t.Error("Directory still exists after recursive RemoveDir()")
		}
	})
}

// ==================== Glob Tests ====================

func TestPath_Glob(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	dirPath := New(testDir)

	// Create test structure
	files := []string{
		"file1.txt",
		"file2.txt",
		"file3.log",
		"test.json",
		"subdir/file4.txt",
		"subdir/file5.log",
	}

	for _, file := range files {
		p := dirPath.Join(file)
		p.Parent().MakeDir(true, true)
		p.WriteText("test", "utf-8")
	}

	tests := []struct {
		name     string
		pattern  string
		minCount int
		maxCount int
	}{
		{
			name:     "All txt files",
			pattern:  "*.txt",
			minCount: 2,
			maxCount: 2,
		},
		{
			name:     "All log files",
			pattern:  "*.log",
			minCount: 1,
			maxCount: 1,
		},
		{
			name:     "Files in subdir",
			pattern:  "subdir/*",
			minCount: 2,
			maxCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := dirPath.Glob(tt.pattern)
			if err != nil {
				t.Fatal(err)
			}

			count := len(matches)
			if count < tt.minCount || count > tt.maxCount {
				t.Errorf("Glob() returned %d matches, want between %d and %d",
					count, tt.minCount, tt.maxCount)
			}
		})
	}
}

// ==================== Copy and Move Tests ====================

func TestPath_LocalCopyAndMove(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	testContent := "Test content for copy and move"

	t.Run("CopyTo file", func(t *testing.T) {
		src := New(filepath.Join(testDir, "copy_src.txt"))
		dst := New(filepath.Join(testDir, "copy_dst.txt"))

		if err := src.WriteText(testContent, "utf-8"); err != nil {
			t.Fatal(err)
		}

		if err := src.CopyTo(dst); err != nil {
			t.Fatal(err)
		}

		// Both should exist
		if !src.Exists() || !dst.Exists() {
			t.Error("Source or destination missing after copy")
		}

		// Content should match
		dstContent, err := dst.ReadText("utf-8")
		if err != nil {
			t.Fatal(err)
		}
		if dstContent != testContent {
			t.Errorf("Content mismatch: got %q, want %q", dstContent, testContent)
		}
	})

	t.Run("CopyTo directory", func(t *testing.T) {
		srcDir := New(filepath.Join(testDir, "copy_src_dir"))
		dstDir := New(filepath.Join(testDir, "copy_dst_dir"))

		srcDir.MakeDir(true, true)
		srcDir.Join("file1.txt").WriteText("content1", "utf-8")
		srcDir.Join("file2.txt").WriteText("content2", "utf-8")
		srcDir.Join("subdir").MakeDir(true, true)
		srcDir.Join("subdir/file3.txt").WriteText("content3", "utf-8")

		if err := srcDir.CopyTo(dstDir); err != nil {
			t.Fatal(err)
		}

		// Verify files were copied
		if !dstDir.Join("file1.txt").Exists() {
			t.Error("file1.txt not copied")
		}
		if !dstDir.Join("subdir/file3.txt").Exists() {
			t.Error("subdir/file3.txt not copied")
		}
	})

	t.Run("MoveTo file", func(t *testing.T) {
		src := New(filepath.Join(testDir, "move_src.txt"))
		dst := New(filepath.Join(testDir, "move_dst.txt"))

		if err := src.WriteText(testContent, "utf-8"); err != nil {
			t.Fatal(err)
		}

		if err := src.MoveTo(dst, false); err != nil {
			t.Fatal(err)
		}

		// Source should not exist, destination should
		if src.Exists() {
			t.Error("Source still exists after move")
		}
		if !dst.Exists() {
			t.Error("Destination doesn't exist after move")
		}

		// Content should match
		dstContent, err := dst.ReadText("utf-8")
		if err != nil {
			t.Fatal(err)
		}
		if dstContent != testContent {
			t.Errorf("Content mismatch: got %q, want %q", dstContent, testContent)
		}
	})

	t.Run("MoveTo with overwrite", func(t *testing.T) {
		src := New(filepath.Join(testDir, "move_overwrite_src.txt"))
		dst := New(filepath.Join(testDir, "move_overwrite_dst.txt"))

		src.WriteText("new content", "utf-8")
		dst.WriteText("old content", "utf-8")

		// Should fail without overwrite
		err := src.MoveTo(dst, false)
		if err == nil {
			t.Error("MoveTo should fail when destination exists and overwrite=false")
		}

		// Should succeed with overwrite
		src.WriteText("new content", "utf-8") // Recreate source
		if err := src.MoveTo(dst, true); err != nil {
			t.Fatal(err)
		}

		content, _ := dst.ReadText("utf-8")
		if content != "new content" {
			t.Error("Destination was not overwritten")
		}
	})

	t.Run("Rename", func(t *testing.T) {
		src := New(filepath.Join(testDir, "rename_old.txt"))
		src.WriteText("test", "utf-8")

		if err := src.Rename("rename_new.txt", false); err != nil {
			t.Fatal(err)
		}

		oldPath := New(filepath.Join(testDir, "rename_old.txt"))
		newPath := New(filepath.Join(testDir, "rename_new.txt"))

		if oldPath.Exists() {
			t.Error("Old file still exists after rename")
		}
		if !newPath.Exists() {
			t.Error("New file doesn't exist after rename")
		}
	})
}

// ==================== SFTP Tests ====================

func TestPath_SFTP(t *testing.T) {
	if !isSFTPAvailable() {
		t.Skip("SFTP server not available")
	}

	t.Log("✓ SFTP server is available and responding")
	t.Logf("✓ Testing against: %s@%s:%s", sftpTestUser, sftpTestHost, sftpTestPort)

	testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	sftpDir := getSFTPTestPath(testID)
	defer cleanupSFTPTestDir(t, sftpDir)

	if err := sftpDir.MakeDir(true, true); err != nil {
		t.Fatalf("Failed to create SFTP test directory: %v", err)
	}

	t.Run("SFTP WriteText and ReadText", func(t *testing.T) {
		p := sftpDir.Join("test.txt")
		content := "Hello SFTP World!"

		if err := p.WriteText(content, "utf-8"); err != nil {
			t.Fatalf("WriteText() error = %v", err)
		}

		got, err := p.ReadText("utf-8")
		if err != nil {
			t.Fatalf("ReadText() error = %v", err)
		}

		if got != content {
			t.Errorf("ReadText() = %q, want %q", got, content)
		}
	})

	t.Run("SFTP MakeDir and List", func(t *testing.T) {
		subdir := sftpDir.Join("subdir")
		if err := subdir.MakeDir(true, true); err != nil {
			t.Fatal(err)
		}

		// Create some files
		subdir.Join("file1.txt").WriteText("test1", "utf-8")
		subdir.Join("file2.txt").WriteText("test2", "utf-8")

		files, err := subdir.List()
		if err != nil {
			t.Fatal(err)
		}

		if len(files) != 2 {
			t.Errorf("List() returned %d files, want 2", len(files))
		}
	})

	t.Run("SFTP to Local Copy", func(t *testing.T) {
		localDir := createTempDir(t)
		defer os.RemoveAll(localDir)

		sftpFile := sftpDir.Join("sftp_to_local.txt")
		localFile := New(filepath.Join(localDir, "local_file.txt"))

		content := "Transfer from SFTP to local"
		if err := sftpFile.WriteText(content, "utf-8"); err != nil {
			t.Fatal(err)
		}

		if err := sftpFile.CopyTo(localFile); err != nil {
			t.Fatal(err)
		}

		got, err := localFile.ReadText("utf-8")
		if err != nil {
			t.Fatal(err)
		}

		if got != content {
			t.Errorf("Content mismatch: got %q, want %q", got, content)
		}
	})

	t.Run("Local to SFTP Copy", func(t *testing.T) {
		localDir := createTempDir(t)
		defer os.RemoveAll(localDir)

		localFile := New(filepath.Join(localDir, "local_file.txt"))
		sftpFile := sftpDir.Join("local_to_sftp.txt")

		content := "Transfer from local to SFTP"
		if err := localFile.WriteText(content, "utf-8"); err != nil {
			t.Fatal(err)
		}

		if err := localFile.CopyTo(sftpFile); err != nil {
			t.Fatal(err)
		}

		got, err := sftpFile.ReadText("utf-8")
		if err != nil {
			t.Fatal(err)
		}

		if got != content {
			t.Errorf("Content mismatch: got %q, want %q", got, content)
		}
	})
}

// ==================== URL Tests ====================

func TestPath_URL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping URL tests in short mode")
	}

	t.Run("URL to Local Copy", func(t *testing.T) {
		testDir := createTempDir(t)
		defer os.RemoveAll(testDir)

		// Using a small, reliable test file
		urlPath := New("https://raw.githubusercontent.com/ImGajeed76/charmer/refs/heads/master/test_file.txt")
		localPath := New(filepath.Join(testDir, "downloaded.txt"))

		if err := urlPath.CopyTo(localPath); err != nil {
			t.Fatalf("CopyTo() error = %v", err)
		}

		if !localPath.Exists() {
			t.Error("Downloaded file doesn't exist")
		}

		// Verify content is not empty
		content, err := localPath.ReadText("utf-8")
		if err != nil {
			t.Fatal(err)
		}
		if len(content) == 0 {
			t.Error("Downloaded file is empty")
		}
	})

	t.Run("URL Stat", func(t *testing.T) {
		urlPath := New("https://raw.githubusercontent.com/ImGajeed76/charmer/refs/heads/master/test_file.txt")

		info, err := urlPath.Stat()
		if err != nil {
			t.Fatal(err)
		}

		if info.Name != "test_file.txt" {
			t.Errorf("Name = %q, want %q", info.Name, "test_file.txt")
		}
	})

	t.Run("URL IsFile", func(t *testing.T) {
		urlPath := New("https://example.com/file.txt")
		if !urlPath.IsFile() {
			t.Error("URL should be treated as file")
		}
		if urlPath.IsDir() {
			t.Error("URL should not be treated as directory")
		}
	})

	t.Run("URL operations should fail", func(t *testing.T) {
		urlPath := New("https://example.com/file.txt")

		// Cannot read URL directly
		_, err := urlPath.ReadText("utf-8")
		if err == nil {
			t.Error("ReadText should fail for URLs")
		}

		// Cannot write to URL
		err = urlPath.WriteText("test", "utf-8")
		if err == nil {
			t.Error("WriteText should fail for URLs")
		}

		// Cannot list URL
		_, err = urlPath.List()
		if err == nil {
			t.Error("List should fail for URLs")
		}
	})
}

// ==================== Error Handling Tests ====================

func TestPath_ErrorHandling(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	t.Run("Read non-existent file", func(t *testing.T) {
		p := New(filepath.Join(testDir, "nonexistent.txt"))
		_, err := p.ReadText("utf-8")
		if err == nil {
			t.Error("Should error when reading non-existent file")
		}
	})

	t.Run("Copy non-existent file", func(t *testing.T) {
		src := New(filepath.Join(testDir, "nonexistent.txt"))
		dst := New(filepath.Join(testDir, "dest.txt"))
		err := src.CopyTo(dst)
		if err == nil {
			t.Error("Should error when copying non-existent file")
		}

		// Verify it's the right error type
		var pathErr *pathmodels.PathError
		if err != nil && !errors.As(err, &pathErr) {
			t.Logf("Error type: %T", err)
		}
	})

	t.Run("List non-directory", func(t *testing.T) {
		p := New(filepath.Join(testDir, "file.txt"))
		p.WriteText("test", "utf-8")

		_, err := p.List()
		if err == nil {
			t.Error("Should error when listing a file")
		}
	})

	t.Run("Remove with missingOk", func(t *testing.T) {
		p := New(filepath.Join(testDir, "missing.txt"))

		// Should error without missingOk
		err := p.Remove(false, false)
		if err == nil {
			t.Error("Should error when removing non-existent file with missingOk=false")
		}

		// Should not error with missingOk
		err = p.Remove(true, false)
		if err != nil {
			t.Errorf("Should not error with missingOk=true: %v", err)
		}
	})
}

// ==================== Concurrency Tests ====================

func TestPath_Concurrency(t *testing.T) {
	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	t.Run("Concurrent writes to different files", func(t *testing.T) {
		dirPath := New(filepath.Join(testDir, "concurrent"))
		dirPath.MakeDir(true, true)

		var wg sync.WaitGroup
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				p := dirPath.Join(fmt.Sprintf("file-%d.txt", n))
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

		// Verify all files were created
		files, err := dirPath.List()
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 10 {
			t.Errorf("Expected 10 files, got %d", len(files))
		}
	})

	t.Run("Concurrent reads", func(t *testing.T) {
		p := New(filepath.Join(testDir, "concurrent-read.txt"))
		content := "test content"
		if err := p.WriteText(content, "utf-8"); err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				got, err := p.ReadText("utf-8")
				if err != nil {
					errors <- err
					return
				}
				if got != content {
					errors <- fmt.Errorf("content mismatch: got %q, want %q", got, content)
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent read error: %v", err)
		}
	})
}

// ==================== Edge Case Tests ====================

func TestPath_EdgeCases(t *testing.T) {
	t.Run("Unicode paths", func(t *testing.T) {
		testDir := createTempDir(t)
		defer os.RemoveAll(testDir)

		p := New(filepath.Join(testDir, "测试文件.txt"))
		content := "你好世界"

		if err := p.WriteText(content, "utf-8"); err != nil {
			t.Fatal(err)
		}

		got, err := p.ReadText("utf-8")
		if err != nil {
			t.Fatal(err)
		}

		if got != content {
			t.Errorf("Unicode content mismatch: got %q, want %q", got, content)
		}
	})

	t.Run("Paths with spaces", func(t *testing.T) {
		testDir := createTempDir(t)
		defer os.RemoveAll(testDir)

		p := New(filepath.Join(testDir, "file with spaces.txt"))
		if err := p.WriteText("test", "utf-8"); err != nil {
			t.Fatal(err)
		}

		if !p.Exists() {
			t.Error("File with spaces should exist")
		}
	})

	t.Run("Very long filename", func(t *testing.T) {
		testDir := createTempDir(t)
		defer os.RemoveAll(testDir)

		// Most file systems support up to 255 bytes for filename
		longName := strings.Repeat("a", 200) + ".txt"
		p := New(filepath.Join(testDir, longName))

		if err := p.WriteText("test", "utf-8"); err != nil {
			t.Fatal(err)
		}

		if !p.Exists() {
			t.Error("File with long name should exist")
		}
	})

	t.Run("Multiple dots in filename", func(t *testing.T) {
		testDir := createTempDir(t)
		defer os.RemoveAll(testDir)

		p := New(filepath.Join(testDir, "file.tar.gz.bak"))
		if err := p.WriteText("test", "utf-8"); err != nil {
			t.Fatal(err)
		}

		if p.Suffix() != "bak" {
			t.Errorf("Suffix() = %q, want %q", p.Suffix(), "bak")
		}
		if p.Stem() != "file.tar.gz" {
			t.Errorf("Stem() = %q, want %q", p.Stem(), "file.tar.gz")
		}
	})

	t.Run("Hidden files", func(t *testing.T) {
		testDir := createTempDir(t)
		defer os.RemoveAll(testDir)

		p := New(filepath.Join(testDir, ".hidden"))
		if err := p.WriteText("test", "utf-8"); err != nil {
			t.Fatal(err)
		}

		if !p.Exists() {
			t.Error("Hidden file should exist")
		}
	})
}

// ==================== Performance Tests ====================

func TestPath_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	testDir := createTempDir(t)
	defer os.RemoveAll(testDir)

	t.Run("Large file operations", func(t *testing.T) {
		p := New(filepath.Join(testDir, "large.bin"))

		// Create 10MB file
		largeData := make([]byte, 10*1024*1024)
		rand.Read(largeData)

		start := time.Now()
		if err := p.WriteBytes(largeData); err != nil {
			t.Fatal(err)
		}
		writeTime := time.Since(start)

		start = time.Now()
		got, err := p.ReadBytes()
		if err != nil {
			t.Fatal(err)
		}
		readTime := time.Since(start)

		if !bytes.Equal(got, largeData) {
			t.Error("Large file content mismatch")
		}

		t.Logf("10MB write: %v, read: %v", writeTime, readTime)
	})

	t.Run("Many small files", func(t *testing.T) {
		dirPath := New(filepath.Join(testDir, "many-files"))
		dirPath.MakeDir(true, true)

		start := time.Now()
		for i := 0; i < 1000; i++ {
			p := dirPath.Join(fmt.Sprintf("file-%d.txt", i))
			if err := p.WriteText("test", "utf-8"); err != nil {
				t.Fatal(err)
			}
		}
		createTime := time.Since(start)

		start = time.Now()
		files, err := dirPath.List()
		if err != nil {
			t.Fatal(err)
		}
		listTime := time.Since(start)

		if len(files) != 1000 {
			t.Errorf("Expected 1000 files, got %d", len(files))
		}

		t.Logf("Create 1000 files: %v, list: %v", createTime, listTime)
	})
}
