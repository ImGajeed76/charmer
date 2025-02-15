# 🎭 Charmer

Charmer is a Go package that automatically generates Terminal User Interfaces (TUIs) from your Go functions. Powered by [Charm](https://github.com/charmbracelet) libraries, it transforms annotated functions into beautiful, navigable command-line interfaces without the hassle of UI implementation.

## ✨ Features

- 🎯 **Simple Integration** - Just annotate your functions with `@Charm`
- 🌳 **Automatic TUI Generation** - Create hierarchical menus with zero UI code
- ⚡ **Charm-Powered** - Built on the robust Bubbles and BubbleTea libraries
- 📝 **Documentation-Driven** - Use annotations to define your CLI structure
- 🚀 **Focus on Logic** - Write your functions, let Charmer handle the UI

## 🚀 Quick Start

### Installation

```bash
go get github.com/imgajeed76/charmer/pkg/charmer@v0.0.0
```

### Setup Your Project

1. Create your `main.go`:
```go
//go:generate go run github.com/imgajeed76/charmer/tools/generate/main.go

package main

import (
    "your-project/internal/registry"
    "github.com/imgajeed76/charmer/pkg/charmer"
)

func main() {
    charmer.Run(registry.RegisteredCharms)
}
```

2. Create a `charms` directory in your project
3. Add your charm functions:

```go
package charms

import "fmt"

// @Charm
// @Title PrintHello
// @Description This function prints hello
// This description is multiline
func PrintHello() {
    fmt.Println("Hello")
}
```

### Build & Run

1. Generate the TUI code:
```bash
go generate
```

2. Run your application:
```bash
go run main.go
```

Or build an executable:
```bash
go build
```

## 🎨 How It Works

Charmer uses a annotation-based approach to create TUIs:

1. Add `@Charm` annotations to your functions
2. Define titles and descriptions using `@Title` and `@Description`
3. Run `go generate` to create the TUI structure
4. Charmer handles the rest - navigation, UI rendering, and execution

## 📁 Project Structure

```
your-project/
├── main.go
└── charms/
    ├── hello.go
    └── other_commands.go
```

## 🛠️ Development Status

⚠️ **Early Alpha Stage**

This project is currently in early development. Features and APIs may change significantly. The current version might not be fully functional as package publishing is still being configured.

## 📝 License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## 💖 Acknowledgments

- Built with love in Switzerland by ImGajeed76
- Powered by the amazing [Charm](https://github.com/charmbracelet) libraries

---

🌟 **Purpose**: Simplifying the creation of beautiful terminal utility applications, one function at a time.

⚠️ **Note**: This is an alpha release. Expect changes and improvements as the project evolves.