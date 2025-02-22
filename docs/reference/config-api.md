# Charmer Configuration Manager

The Charmer Configuration Manager provides a secure way to store and manage configuration values using the system's native keyring/keychain. This documentation covers the complete API for the `config` package.

## Overview

The Configuration Manager uses the system's secure keyring (via [zalando/go-keyring](https://github.com/zalando/go-keyring)) to safely store sensitive configuration values. Each configuration instance is namespaced using a service name, allowing multiple applications or components to maintain separate configurations.

## Installation

The config package is included as part of Charmer:

```go
import "github.com/ImGajeed76/charmer/pkg/charmer/config"
```

## Core Concepts

### Service Names
Each Config instance is associated with a service name that acts as a namespace for stored values. This allows multiple applications to use the keyring without conflicts.

### Keys and Values
- All values are stored as strings
- Empty keys are not allowed
- Keys must be unique within a service namespace
- Values can be empty strings

## API Reference

### Creating a New Config Instance

```go
func New(service string) (*Config, error)
```

Creates a new configuration manager instance.

**Parameters:**

- `service` (string): The service name to namespace stored values

**Returns:**

- `*Config`: Configuration manager instance
- `error`: Error if service name is empty

**Example:**
```go
cfg, err := config.New("myapp")
if err != nil {
    log.Fatal(err)
}
```

### Setting Values

#### Set
```go
func (c *Config) Set(key, value string) error
```

Stores a value in the keyring.

**Parameters:**

- `key` (string): The key to store the value under
- `value` (string): The value to store

**Returns:**

- `error`: Error if key is empty or storage fails

**Example:**
```go
err := cfg.Set("api_key", "secret123")
```

#### SetDefault
```go
func (c *Config) SetDefault(key, value string) error
```

Stores a value only if the key doesn't already exist.

**Parameters:**

- `key` (string): The key to store the value under
- `value` (string): The default value to store

**Returns:**

- `error`: Error if key is empty or storage fails

**Example:**
```go
err := cfg.SetDefault("timeout", "30")
```

#### SetFromInput
```go
func (c *Config) SetFromInput(key string, options console.InputOptions) (string, error)
```

Prompts the user for input and stores the provided value.

**Parameters:**

- `key` (string): The key to store the value under
- `options` (console.InputOptions): Input prompt configuration

**Returns:**

- `string`: The value entered by the user
- `error`: Error if input collection or storage fails

**Example:**
```go
value, err := cfg.SetFromInput("password", console.InputOptions{
    Prompt: "Enter password: "
})
```

### Retrieving Values

#### Get
```go
func (c *Config) Get(key string) string
```

Retrieves a value from the keyring.

**Parameters:**

- `key` (string): The key to retrieve

**Returns:**

- `string`: The stored value, or empty string if not found

**Example:**
```go
apiKey := cfg.Get("api_key")
```

#### Exists
```go
func (c *Config) Exists(key string) bool
```

Checks if a key exists in the keyring.

**Parameters:**

- `key` (string): The key to check

**Returns:**

- `bool`: True if the key exists, false otherwise

**Example:**
```go
if cfg.Exists("api_key") {
    // Key exists
}
```

### Deleting Values

#### Delete
```go
func (c *Config) Delete(key string) error
```

Removes a single value from the keyring.

**Parameters:**

- `key` (string): The key to remove

**Returns:**

- `error`: Error if key is empty or deletion fails

**Example:**
```go
err := cfg.Delete("old_key")
```

#### DeleteAll
```go
func (c *Config) DeleteAll() error
```

Removes all values stored under the service name.

**Returns:**

- `error`: Error if deletion fails

**Example:**
```go
err := cfg.DeleteAll()
```

## Best Practices

1. **Service Names**
    - Use descriptive, unique service names
    - Consider including version or environment information
    - Example: `myapp-prod` or `myapp-v1`

2. **Key Names**
    - Use consistent naming conventions
    - Consider namespacing keys for different components
    - Example: `db.password`, `api.key`

3. **Error Handling**
    - Always check errors when setting values
    - Consider providing fallback values when getting keys that might not exist

## Example Usage

Here's a real-world example showing how to implement a global configuration manager for SFTP settings:

```go
package config

import (
	"github.com/ImGajeed76/charmer/pkg/charmer/config"
	"github.com/ImGajeed76/charmer/pkg/charmer/console"
	"github.com/ImGajeed76/charmer/pkg/charmer/path"
	"sync"
)

var (
	globalCfg *config.Config
	once      sync.Once
)

func InitConfig() {
	once.Do(func() {
		cfg, err := config.New("charmer-testing")
		if err != nil {
			panic(err)
		}
		globalCfg = cfg
	})

	err := globalCfg.SetDefault("sftp-hostname", "myserver.com")
	if err != nil {
		panic(err)
	}

	err = globalCfg.SetDefault("sftp-port", "22")
	if err != nil {
		panic(err)
	}

	changeSettings, changeErr := console.YesNo(console.YesNoOptions{
		Prompt:     "Do you want to change the SFTP settings?",
		DefaultYes: false,
		YesText:    "Yes, change settings",
		NoText:     "No, keep existing settings",
	})

	if changeErr != nil {
		panic(changeErr)
	}

	if changeSettings {
		err = globalCfg.Delete("sftp-username")
		if err != nil {
			return
		}
		err = globalCfg.Delete("sftp-password")
		if err != nil {
			return
		}
	}

	if !globalCfg.Exists("sftp-username") {
		_, usernameErr := globalCfg.SetFromInput("sftp-username", console.InputOptions{
			Prompt:   "Enter the SFTP username",
			Required: true,
		})
		if usernameErr != nil {
			panic(usernameErr)
		}
	}

	if !globalCfg.Exists("sftp-password") {
		_, passwordErr := globalCfg.SetFromInput("sftp-password", console.InputOptions{
			Prompt: "Enter the SFTP password (will be stored securely)",
		})
		if passwordErr != nil {
			panic(passwordErr)
		}
	}
}

func Cfg() *config.Config {
	return globalCfg
}

func GetSFTPConfig() *path.SFTPConfig {
	return &path.SFTPConfig{
		Host:     globalCfg.Get("sftp-hostname"),
		Port:     globalCfg.Get("sftp-port"),
		Username: globalCfg.Get("sftp-username"),
		Password: globalCfg.Get("sftp-password"),
	}
}

```

## Error Handling

The configuration manager returns errors in the following cases:

1. Empty service name when creating a new instance
2. Empty key when setting or deleting values
3. System keyring access errors
4. User input collection errors

Always check returned errors and handle them appropriately in your application.

## Limitations

1. Only string values are supported
2. No built-in encryption (relies on system keyring security)
3. No support for structured data (must be serialized/deserialized)
4. Service names and keys must be non-empty strings

## Related Documentation

- [Charmer Main Documentation](/)
- [Console Package Documentation](console-api.md)
- [Go Keyring Documentation](https://github.com/zalando/go-keyring)