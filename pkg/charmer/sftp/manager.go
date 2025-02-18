package sftpmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var (
	globalManager *Manager
	once          sync.Once
)

// Default configuration values
const (
	DefaultMaxIdleTime       = 5 * time.Minute
	DefaultConnectTimeout    = 10 * time.Second
	DefaultMaxRetries        = 3
	DefaultRetryDelay        = 1 * time.Second
	DefaultKeepAliveInterval = 30 * time.Second
	DefaultMaxConnections    = 10
	DefaultCleanupInterval   = 2 * time.Minute
)

// ConnectionDetails holds the information needed to establish an SFTP connection
type ConnectionDetails struct {
	Hostname          string
	Port              int
	Username          string
	Password          string
	ConnectTimeout    time.Duration
	MaxRetries        int
	RetryDelay        time.Duration
	KeepAliveInterval time.Duration
	EnableCompression bool
}

// String returns a unique string representation of the connection details
func (cd ConnectionDetails) String() string {
	return fmt.Sprintf("%s@%s:%d", cd.Username, cd.Hostname, cd.Port)
}

// applyDefaults sets default values for unspecified fields
func (cd *ConnectionDetails) applyDefaults() {
	if cd.ConnectTimeout == 0 {
		cd.ConnectTimeout = DefaultConnectTimeout
	}
	if cd.MaxRetries == 0 {
		cd.MaxRetries = DefaultMaxRetries
	}
	if cd.RetryDelay == 0 {
		cd.RetryDelay = DefaultRetryDelay
	}
	if cd.KeepAliveInterval == 0 {
		cd.KeepAliveInterval = DefaultKeepAliveInterval
	}
}

// clientInfo holds the SFTP client and its last used timestamp
type clientInfo struct {
	client    *sftp.Client
	sshClient *ssh.Client
	lastUsed  time.Time
}

// ManagerConfig holds the configuration for the SFTP manager
type ManagerConfig struct {
	MaxIdleTime     time.Duration
	MaxConnections  int
	CleanupInterval time.Duration
}

// Manager handles SFTP client pooling and lifecycle
type Manager struct {
	clients map[string]*clientInfo
	mu      sync.RWMutex
	config  ManagerConfig
	done    chan struct{}
}

// NewManager creates a new Manager with the given configuration
func NewManager(config ManagerConfig) *Manager {
	if config.MaxIdleTime == 0 {
		config.MaxIdleTime = DefaultMaxIdleTime
	}
	if config.MaxConnections == 0 {
		config.MaxConnections = DefaultMaxConnections
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = DefaultCleanupInterval
	}

	m := &Manager{
		clients: make(map[string]*clientInfo),
		config:  config,
		done:    make(chan struct{}),
	}
	go m.cleanup()
	return m
}

// GetGlobalManager returns the global SFTP manager instance, creating it if needed
func GetGlobalManager() *Manager {
	once.Do(func() {
		globalManager = NewManager(ManagerConfig{})
	})
	return globalManager
}

// GetClient is a convenience function that uses the global manager
func GetClient(ctx context.Context, details ConnectionDetails) (*sftp.Client, error) {
	return GetGlobalManager().GetClient(ctx, details)
}

// GetClient returns an SFTP client for the given connection details
func (m *Manager) GetClient(ctx context.Context, details ConnectionDetails) (*sftp.Client, error) {
	details.applyDefaults()
	key := details.String()

	// Check connection pool limit
	m.mu.RLock()
	if len(m.clients) >= m.config.MaxConnections {
		m.mu.RUnlock()
		return nil, fmt.Errorf("connection pool limit reached (%d)", m.config.MaxConnections)
	}
	m.mu.RUnlock()

	// Try to get existing client
	if client, ok := m.getExistingClient(key); ok {
		return client, nil
	}

	// Create new client with retries
	var client *sftp.Client
	var err error
	for attempt := 0; attempt <= details.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if client, err = m.createNewClient(details); err == nil {
				return client, nil
			}
			if attempt < details.MaxRetries {
				time.Sleep(details.RetryDelay)
			}
		}
	}
	return nil, fmt.Errorf("failed to create client after %d attempts: %v", details.MaxRetries+1, err)
}

func (m *Manager) getExistingClient(key string) (*sftp.Client, bool) {
	m.mu.RLock()
	info, exists := m.clients[key]
	if exists {
		info.lastUsed = time.Now()
	}
	m.mu.RUnlock()

	if exists {
		// Test if connection is still alive
		_, err := info.client.Getwd()
		if err == nil {
			return info.client, true
		}

		// Connection is dead, remove it
		m.mu.Lock()
		delete(m.clients, key)
		m.mu.Unlock()
	}
	return nil, false
}

func (m *Manager) createNewClient(details ConnectionDetails) (*sftp.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: details.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(details.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
		Timeout:         details.ConnectTimeout,
	}

	if details.EnableCompression {
		sshConfig.SetDefaults()
		sshConfig.Ciphers = append(sshConfig.Ciphers, "zlib@openssh.com")
	}

	// Connect to SSH server
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", details.Hostname, details.Port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %v", err)
	}

	// Setup keep-alive if configured
	if details.KeepAliveInterval > 0 {
		go m.keepAlive(sshClient, details.KeepAliveInterval)
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %v", err)
	}

	// Store new client
	info := &clientInfo{
		client:    sftpClient,
		sshClient: sshClient,
		lastUsed:  time.Now(),
	}

	key := details.String()
	m.mu.Lock()
	m.clients[key] = info
	m.mu.Unlock()

	return sftpClient, nil
}

func (m *Manager) keepAlive(client *ssh.Client, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				return
			}
		case <-m.done:
			return
		}
	}
}

// cleanup periodically checks for and removes idle connections
func (m *Manager) cleanup() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for key, info := range m.clients {
				if now.Sub(info.lastUsed) > m.config.MaxIdleTime {
					info.client.Close()
					info.sshClient.Close()
					delete(m.clients, key)
				}
			}
			m.mu.Unlock()
		case <-m.done:
			return
		}
	}
}

// Close closes all connections and stops the cleanup goroutine
func (m *Manager) Close() {
	close(m.done) // Signal cleanup goroutine to stop

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, info := range m.clients {
		info.client.Close()
		info.sshClient.Close()
	}

	m.clients = make(map[string]*clientInfo)
}

// Stats returns current connection statistics
func (m *Manager) Stats() map[string]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]time.Time, len(m.clients))
	for key, info := range m.clients {
		stats[key] = info.lastUsed
	}
	return stats
}
