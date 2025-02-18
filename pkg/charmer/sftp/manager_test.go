package sftpmanager

import (
	"context"
	"testing"
	"time"
)

const (
	sftpUser = "sftptest"
	sftpPass = "testpass123"
	sftpHost = "localhost"
	sftpPort = 22
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name   string
		config ManagerConfig
		want   ManagerConfig
	}{
		{
			name:   "default configuration",
			config: ManagerConfig{},
			want: ManagerConfig{
				MaxIdleTime:     DefaultMaxIdleTime,
				MaxConnections:  DefaultMaxConnections,
				CleanupInterval: DefaultCleanupInterval,
			},
		},
		{
			name: "custom configuration",
			config: ManagerConfig{
				MaxIdleTime:     10 * time.Minute,
				MaxConnections:  5,
				CleanupInterval: 1 * time.Minute,
			},
			want: ManagerConfig{
				MaxIdleTime:     10 * time.Minute,
				MaxConnections:  5,
				CleanupInterval: 1 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.config)
			if manager.config.MaxIdleTime != tt.want.MaxIdleTime {
				t.Errorf("MaxIdleTime = %v, want %v", manager.config.MaxIdleTime, tt.want.MaxIdleTime)
			}
			if manager.config.MaxConnections != tt.want.MaxConnections {
				t.Errorf("MaxConnections = %v, want %v", manager.config.MaxConnections, tt.want.MaxConnections)
			}
			if manager.config.CleanupInterval != tt.want.CleanupInterval {
				t.Errorf("CleanupInterval = %v, want %v", manager.config.CleanupInterval, tt.want.CleanupInterval)
			}
		})
	}
}

func TestGetClient(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	defer manager.Close()

	tests := []struct {
		name    string
		details ConnectionDetails
		wantErr bool
	}{
		{
			name: "valid connection",
			details: ConnectionDetails{
				Hostname:          sftpHost,
				Port:              sftpPort,
				Username:          sftpUser,
				Password:          sftpPass,
				ConnectTimeout:    5 * time.Second,
				MaxRetries:        2,
				RetryDelay:        time.Second,
				EnableCompression: true,
			},
			wantErr: false,
		},
		{
			name: "invalid credentials",
			details: ConnectionDetails{
				Hostname:       sftpHost,
				Port:           sftpPort,
				Username:       "invalid",
				Password:       "invalid",
				ConnectTimeout: 2 * time.Second,
				MaxRetries:     1,
				RetryDelay:     time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := manager.GetClient(ctx, tt.details)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Test if client is functional
				pwd, err := client.Getwd()
				if err != nil {
					t.Errorf("Failed to get working directory: %v", err)
				}
				if pwd == "" {
					t.Error("Working directory is empty")
				}
			}
		})
	}
}

func TestConnectionPool(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxConnections: 2,
	})
	defer manager.Close()

	details := ConnectionDetails{
		Hostname:          sftpHost,
		Port:              sftpPort,
		Username:          sftpUser,
		Password:          sftpPass,
		ConnectTimeout:    5 * time.Second,
		KeepAliveInterval: 30 * time.Second,
	}

	ctx := context.Background()

	// Test connection pooling
	client1, err := manager.GetClient(ctx, details)
	if err != nil {
		t.Fatalf("Failed to get first client: %v", err)
	}

	// Get the same client from pool
	client2, err := manager.GetClient(ctx, details)
	if err != nil {
		t.Fatalf("Failed to get second client: %v", err)
	}

	if client1 != client2 {
		t.Error("Expected to get the same client from pool")
	}

	// Test connection limit
	details.Username = "another_user" // Force new connection
	_, err = manager.GetClient(ctx, details)
	if err == nil {
		t.Error("Expected error when exceeding connection limit")
	}
}

func TestConnectionCleanup(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxIdleTime:     2 * time.Second,
		CleanupInterval: time.Second,
	})
	defer manager.Close()

	details := ConnectionDetails{
		Hostname:       sftpHost,
		Port:           sftpPort,
		Username:       sftpUser,
		Password:       sftpPass,
		ConnectTimeout: 5 * time.Second,
	}

	ctx := context.Background()

	// Create a client
	_, err := manager.GetClient(ctx, details)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Wait for cleanup
	time.Sleep(3 * time.Second)

	// Check if connection was cleaned up
	stats := manager.Stats()
	if len(stats) > 0 {
		t.Error("Expected all connections to be cleaned up")
	}
}

func TestGlobalManager(t *testing.T) {
	manager1 := GetGlobalManager()
	manager2 := GetGlobalManager()

	if manager1 != manager2 {
		t.Error("Expected to get the same global manager instance")
	}
}
