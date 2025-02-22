# Installation

Getting started with Charmer is straightforward. Follow these steps to set up Charmer in your Go project.

## Prerequisites

- Go 1.24 or later
- A Go project where you want to implement a TUI

## Installing Charmer

Add Charmer to your Go project using the `go get` command:

```bash
go get github.com/ImGajeed76/charmer@latest
```

> **Note:** You maybe have to run `go mod tidy` to update your `go.mod` file after installing Charmer.

This command fetches the latest version of Charmer and adds it to your project's dependencies.

## Setting Up Your Project

### 1. Update Your `main.go`

Create or update your `main.go` file to include the Charmer generator and runner:

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

The `//go:generate` directive tells Go to run the Charmer code generator when you execute `go generate`.

### 2. Create Your First Charm

Create a directory called `charms` in your project root (if it doesn't exist already). This is where you'll place your
charm functions:

```bash
mkdir -p charms
```

If you don't create this directory manually, the generator will create it for you with a sample charm.

### 3. Run the Generator

Execute the following command to generate the necessary code for your Charmer TUI:

```bash
go generate
```

This will:

- Create a registry for your charm functions
- Set up the navigation structure
- Generate all the necessary UI code

### 4. Run Your Application

Now you can run your application:

```bash
go run main.go
```

Or build an executable:

```bash
go build
./your-project
```

## Verifying the Installation

After running your application, you should see a beautiful terminal interface with navigation options. If you're seeing
this, congratulations! Charmer is successfully installed and running.

If you encounter any issues, check the [troubleshooting section](../guides/troubleshooting.md)
or [open an issue](https://github.com/ImGajeed76/charmer/issues/new) on the GitHub repository.

## Next Steps

Now that you have Charmer installed, head over to the [Quick Start](quick-start.md) guide to learn how to create your
own custom charms.