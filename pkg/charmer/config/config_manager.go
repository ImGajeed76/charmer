package config

import (
	"fmt"
	"github.com/ImGajeed76/charmer/pkg/charmer/console"
	"github.com/zalando/go-keyring"
)

// Config represents a configuration instance that uses the system keyring
// to securely store values.
type Config struct {
	service string
}

// New creates a new Config instance with the given service name.
// The service name is used to namespace the stored values in the keyring.
func New(service string) (*Config, error) {
	if service == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}
	return &Config{
		service: service,
	}, nil
}

// Set stores a value in the keyring under the given key.
func (c *Config) Set(key, value string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	return keyring.Set(c.service, key, value)
}

// SetDefault stores a value in the keyring if it doesn't already exist.
// If the key already exists, it returns nil without modifying the existing value.
func (c *Config) SetDefault(key, value string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Check if the key already exists
	existing, err := keyring.Get(c.service, key)
	if err == nil && existing != "" {
		// Key exists, return without modification
		return nil
	}

	// Key doesn't exist or there was an error reading it,
	// set the default value
	return keyring.Set(c.service, key, value)
}

// Get retrieves a value from the keyring by its key.
// Returns an empty string if the key doesn't exist.
func (c *Config) Get(key string) string {
	if key == "" {
		return ""
	}

	value, err := keyring.Get(c.service, key)
	if err != nil {
		return ""
	}
	return value
}

// Exists checks if a key exists in the keyring.
func (c *Config) Exists(key string) bool {
	if key == "" {
		return false
	}

	_, err := keyring.Get(c.service, key)
	return err == nil
}

// Delete removes a value from the keyring by its key.
func (c *Config) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	return keyring.Delete(c.service, key)
}

// DeleteAll removes all values stored under the service name.
func (c *Config) DeleteAll() error {
	return keyring.DeleteAll(c.service)
}

// SetFromInput prompts the user for input and stores the value in the keyring.
func (c *Config) SetFromInput(key string, options console.InputOptions) (string, error) {
	value, err := console.Input(options)
	if err != nil {
		return "", err
	}

	err = c.Set(key, value)
	if err != nil {
		return "", err
	}

	return value, nil
}
