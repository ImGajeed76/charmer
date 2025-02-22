# Quick Start Guide

This guide will walk you through creating your first Charmer-powered application with beautiful terminal UIs.

## Prerequisites

Before you begin, make sure you have:

- Installed Charmer as described in the [Installation Guide](installation.md)
- A basic understanding of Go programming

## Creating Your First Charm

Let's create a simple greeting charm that asks for a user's name and displays a personalized greeting.

### 1. Create a Charm File

In your project's `charms` directory, create a new file called `greeting.go`:

```go
package charms

import (
	"fmt"
	"github.com/ImGajeed76/charmer/pkg/charmer/console"
)

// Greeting godoc
// @Charm
// @Title Greeting
// @Description
// # Greeting
// ## Description
// This is a simple greeting function that asks for a name and greets the user.
func Greeting() {
	name, _ := console.Input(console.InputOptions{
		Prompt: "What is your name?",
	})

	fmt.Printf("Hello, %s!\n", name)
}
```

### 2. Generate Your TUI

Run the generator to create the TUI code:

```bash
go generate
```

### 3. Run Your Application

```bash
go run main.go
```

You should now see a beautiful terminal interface with your Greeting charm. When selected, it will prompt for a name and
display a greeting.

## Understanding Charm Annotations

Charmer uses annotations to transform regular Go functions into interactive TUI elements:

- `@Charm`: Marks a function to be included in the TUI
- `@Title`: Sets the display title in the menu
- `@Description`: Provides detailed information (supports Markdown)

### Multiline Descriptions

As shown in the example, descriptions can span multiple lines and use Markdown:

```go
// @Description
// # Greeting
// ## Description
// This is a simple greeting function that asks for a name and greets the user.
```

This will be rendered beautifully in the TUI with proper formatting.

## Using Console Utilities

Charmer provides several utilities for terminal interaction in the `console` package:

### Text Input

The `Input` function provides a simple way to get text input from users:

```go
// Simple usage
value, err := console.Input()

// With custom options
name, err := console.Input(console.InputOptions{
Prompt:      "Enter your name:",
Regex:       "^[a-zA-Z ]+$",
RegexError:  "Please enter letters and spaces only",
Default:     "John",
Placeholder: "Enter name here",
Required:    true,
})
```

### Selection Lists

For selecting from a list of options:

```go
options := []string{"Option 1", "Option 2", "Option 3"}

// Simple usage
selectedIndex, err := console.ListSelect(options)

// With custom title
selectedIndex, err = console.ListSelect(options, console.ListSelectOptions{
Title: "Choose your favorite:",
})
```

### Yes/No Confirmations

Get simple yes/no confirmations:

```go
// Simple usage
result, err := console.YesNo()

// Custom options
result, err = console.YesNo(console.YesNoOptions{
Prompt:     "Do you want to continue?",
DefaultYes: false,
YesText:    "Continue",
NoText:     "Cancel",
})
```

## Creating a Hierarchical Menu

You can organize your charms into hierarchical menus using the folder structure:

### Create a Submenu

1. Create a directory in your `charms` folder:

```bash
mkdir -p charms/Utils
```

2. Add charm functions to this directory:

```go
// charms/Utils/calculator.go
package utils

import (
	"fmt"
	"github.com/ImGajeed76/charmer/pkg/charmer/console"
	"strconv"
)

// Add godoc
// @Charm
// @Title Add Numbers
// @Description Adds two numbers together
func Add() {
	num1Str, _ := console.Input(console.InputOptions{
		Prompt: "Enter first number:",
	})

	num2Str, _ := console.Input(console.InputOptions{
		Prompt: "Enter second number:",
	})

	num1, _ := strconv.ParseFloat(num1Str, 64)
	num2, _ := strconv.ParseFloat(num2Str, 64)

	fmt.Printf("Result: %.2f\n", num1+num2)
}
```

3. Generate the TUI:

```bash
go generate
```

This will create a "Utils" submenu containing your "Add Numbers" charm.

!!! tip "Folder Naming"

    The folder name will be used as the submenu title. If your folder is named `Utils`, the submenu will be titled "Utils".

!!! note "Nested Submenus"

    You can nest submenus as deep as you like by creating additional directories.

Happy charming!