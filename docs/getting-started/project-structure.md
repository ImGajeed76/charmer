# Project Structure

Understanding the recommended project structure for a Charmer application will help you organize your code efficiently.
This guide explains how to structure your Go project to make the most of Charmer's capabilities.

## Basic Structure

A typical Charmer project follows this structure:

```
your-project/
├── main.go                       # Entry point with go:generate directive
├── go.mod                        # Go module definition
├── go.sum                        
├── charms/                       # Directory containing your charm functions
│   ├── greeting.go               
│   └── Utilities                 # Optional: Subdirectories for organizing charms
│       └── advanced.go               
└── internal/                     
    └── registry/                 
        └── generated.go          # Auto-generated registry of all your charms (do not edit)
```

## Main Components

### Entry Point (`main.go`)

The `main.go` file is the entry point of your application. It should include:

1. The `go:generate` directive to run the Charmer code generator
2. Import statements for your registry and Charmer package
3. The `main()` function that runs your Charmer application

```go
//go:generate go run github.com/ImGajeed76/charmer/tools/generate

package main

import (
	"your-project/internal/registry"
	"github.com/ImGajeed76/charmer/pkg/charmer"
)

func main() {
	charmer.Run(registry.RegisteredCharms)
}
```

### Charms Directory

The `charms/` directory contains all your charm function files. Each file can contain multiple charm functions, but it's
often helpful to organize related functions in the same file.

```go
// charms/greeting.go
package charms

import "fmt"

// SayHello godoc
// @Charm
// @Title Say Hello
// @Description Displays a simple greeting
func SayHello() {
	fmt.Println("Hello, world!")
}

// GreetUser godoc
// @Charm
// @Title Greet User
// @Description Displays a personalized greeting with the user's name
func GreetUser(name string) {
	fmt.Printf("Hello, %s! Welcome to Charmer!\n", name)
}
```

### Generated Registry

When you run `go generate`, Charmer creates the registry directory and files automatically:

```
internal/registry/registry.go
```

This file contains the code that registers all your charm functions with the TUI system. You should not edit this file
manually as it's regenerated each time you run `go generate`.

## Organization Strategies

### Functional Organization

Organize your charm functions by feature or functionality:

```
charms/
├── user_management.go     # Functions related to user operations
├── data_processing.go     # Functions for data manipulation
└── system_utilities.go    # System-level utility operations
```

### Directory-Based Organization

A powerful way to organize your charms is by using nested folders within the `charms/` directory. This creates a clean
hierarchical structure in the TUI where the folder name becomes the category title:

```
charms/
├── User/                  # "User" category in the TUI
│   ├── create.go          # Functions for user creation
│   └── manage.go          # Functions for user management
├── Network/               # "Network" category in the TUI
│   ├── diagnostics.go     # Network diagnostic functions
│   └── config.go          # Network configuration functions
└── System/                # "System" category in the TUI
    ├── monitor.go         # System monitoring functions
    └── maintenance.go     # System maintenance functions
```

With this approach, the TUI will first present the user with category selections (User, Network, System) and then show
the specific functions within the selected category. This prevents the interface from becoming bloated with too many
options at once, creating a cleaner, more navigable experience.

## Build Artifacts

After building your Charmer application, you'll have a single executable that contains your entire TUI:

```
your-project  # Executable binary
```

This makes distribution simple - users only need the single binary file to run your application.

## Example Project Layout

Here's a more complete example of a Charmer project structure:

```
my-cli-tool/
├── main.go
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── charms/
│   ├── System/                    # System category in the TUI
│   │   ├── info.go                # System information commands
│   │   └── maintenance.go         # System maintenance commands
│   ├── Network/                   # Network category in the TUI
│   │   ├── diagnostics.go         # Network diagnostic commands
│   │   └── configuration.go       # Network configuration commands
│   └── Files/                     # Files category in the TUI
│       ├── operations.go          # File operation commands
│       └── search.go              # File search commands
├── internal/
│   ├── registry/
│   │   └── registry.go
│   └── helpers/
│       └── common.go
└── docs/
    └── screenshots/
        └── demo.png
```

## Best Practices

1. **Use descriptive annotations**: Clear titles and descriptions make your TUI more user-friendly.

2. **Use folder organization for large applications**: For applications with many commands, use the directory-based
   organization to create logical categories and prevent UI clutter.

3. **Organize related functions**: Group related functionality in the same file for better maintainability.

4. **Update generated code**: Run `go generate` whenever you add, modify, or remove charm functions.

5. **Document your charms**: Include detailed descriptions to help users understand what each function does.

6. **Balance menu depth**: Avoid creating too many nested levels in directory structure nesting.

7. **Follow Go standards**: Use standard Go naming conventions and code organization practices.
