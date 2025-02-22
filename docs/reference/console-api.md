# Charmer Console API Documentation

Charmer provides a set of utilities for interacting with the terminal in the `console` package. These utilities
include functions for text input, list selection, progress bars, and binary yes/no confirmations.

## Table of Contents

1. [Overview](#overview)
2. [YesNo Component](#yesno-component)
3. [Input Component](#input-component)
4. [ProgressBar Component](#progressbar-component)
5. [ListSelect Component](#listselect-component)

## Overview

This API provides a set of tools that make it easier to use charmer. It includes functions for common terminal

- Binary yes/no confirmations
- Text input with validation
- Progress bars with customizable appearance
- List selection interfaces

The API is built using the [Charm libraries](https://github.com/charmbracelet) for terminal UI components.

You are more than welcome to contribute to this API by adding more functions or improving the existing ones.

## YesNo Component

The `YesNo` component provides a simple binary confirmation prompt.

### Types

```go
type YesNoOptions struct {
    Prompt     string
    DefaultYes bool   // If true, "Yes" is pre-selected
    YesText    string // Custom text for "Yes" option
    NoText     string // Custom text for "No" option
}
```

### Functions

```go
func DefaultYesNoOptions() YesNoOptions
```

Returns default options for the YesNo component:

- Prompt: "Confirm?"
- DefaultYes: true
- YesText: "Yes"
- NoText: "No"

```go
func YesNo(options ...YesNoOptions) (bool, error)
```

Displays a yes/no prompt and returns the user's selection as a boolean value.

### Example Usage

```go
package charms

import (
	"github.com/ImGajeed76/charmer/pkg/charmer/console"
)

// myFunc godoc
// @Charm
// @Title myFunc
// @Description This is a sample charm
func myFunc() {
	// Simple usage with defaults
	result, _ := console.YesNo()

	// Custom options
	result, _ := console.YesNo(console.YesNoOptions{
		Prompt:     "Do you want to continue?",
		DefaultYes: false,
		YesText:    "Continue",
		NoText:     "Cancel",
	})
}
```

## Input Component

The `Input` component provides a text input field with optional validation.

### Types

```go
type InputOptions struct {
    Prompt      string
    Regex       string
    RegexError  string // Custom error message for regex validation
    Default     string
    Placeholder string
    CharLimit   int
    Width       int
    Required    bool // If true, empty input is not allowed
}
```

### Functions

```go
func DefaultInputOptions() InputOptions
```

Returns default options for the Input component:

- Prompt: "Enter value:"
- CharLimit: 156
- Width: 20
- Required: false
- RegexError: "Input format is invalid"

```go
func Input(options ...InputOptions) (string, error)
```

Displays a text input prompt and returns the user's input as a string.

### Example Usage

```go
package charms

import (
	"github.com/ImGajeed76/charmer/pkg/charmer/console"
)

// myFunc godoc
// @Charm
// @Title myFunc
// @Description This is a sample charm
func myFunc() {
	// Simple usage with defaults
	value1, _ := console.Input()

	// Custom options with validation
	value2, _ := console.Input(console.InputOptions{
		Prompt:      "Enter your name:",
		Regex:       "^[a-zA-Z ]+$",
		RegexError:  "Please enter letters and spaces only",
		Default:     "John",
		Placeholder: "Enter name here",
		Required:    true,
	})
}
```

## ProgressBar Component

The `ProgressBar` component provides a visual progress indicator with customizable appearance.

### Types

```go
type ProgressOptions struct {
    GradientColors []string
    Width          int
    Padding        int
}

type ProgressBar struct {
    Update func (total, count int64)
    Close  func ()
    Finish func ()
}
```

### Functions

```go
func DefaultProgressOptions() ProgressOptions
```

Returns default options for the ProgressBar component:

- GradientColors: []string{"#5956e0", "#e86ef6"}
- Width: 80
- Padding: 2

```go
func NewProgressBar(opts ...ProgressOptions) *ProgressBar
```

Creates a new progress bar with the specified options.

### Methods

```go
func (p *ProgressBar) Update(total, count int64)
```

Updates the progress bar's current state. The `total` parameter represents the total units of work, and `count`
represents the completed units.

```go
func (p *ProgressBar) Close()
```

Closes the progress bar and cleans up resources.

```go
func (p *ProgressBar) Finish()
```

Sets the progress to 100% and then closes the progress bar.

### Example Usage

```go
package charms

import (
	"github.com/ImGajeed76/charmer/pkg/charmer/console"
	"time"
)

// myFunc godoc
// @Charm
// @Title myFunc
// @Description This is a sample charm
func myFunc() {
	// Create a progress bar with default options
	progressBar := console.NewProgressBar()

	// Update progress (e.g., in a loop)
	for i := int64(0); i <= 100; i++ {
		progressBar.Update(100, i)
		time.Sleep(50 * time.Millisecond)
	}

	// Finish the progress bar
	progressBar.Finish()

	// Or close it manually
	// progressBar.Close()
}
```

!!! tip "Usage in file operations"

    The `Update` methode can also be provided to a **CopyTo** or **MoveTo** operation to show the progress of the operation.

## ListSelect Component

The `ListSelect` component provides a selectable list of options.

### Types

```go
type ListSelectOptions struct {
    Title string
}
```

### Functions

```go
func DefaultListSelectOptions() ListSelectOptions
```

Returns default options for the ListSelect component:

- Title: "Select an option:"

```go
func ListSelect(items []string, options ...ListSelectOptions) (int, error)
```

Displays a list of options and returns the selected index.

### Example Usage

```go
package charms

import (
	"fmt"
	"github.com/ImGajeed76/charmer/pkg/charmer/console"
)

// myFunc godoc
// @Charm
// @Title myFunc
// @Description This is a sample charm
func myFunc() {
	options := []string{"Option 1", "Option 2", "Option 3"}

	// Simple usage
	selectedIndex, err := console.ListSelect(options)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Selected: %s\n", options[selectedIndex])

	// With custom title
	selectedIndex, err = console.ListSelect(options, console.ListSelectOptions{
		Title: "Choose your favorite:",
	})
}
```

!!! warning "Documentation Errors"

    If you find any errors or inconsistencies in this documentation, please report them by 
    [creating an issue](https://github.com/ImGajeed76/charmer/issues/new) on the Charmer GitHub repository. 
    Your feedback helps improve the project for everyone!