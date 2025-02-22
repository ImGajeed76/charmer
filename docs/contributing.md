# Contributing to Charmer

Thank you for your interest in contributing to Charmer! This document provides guidelines and information for
contributors who want to help improve this Go-based TUI generator.

## 💭 A Note from the Creator

Hey there! I'm ImGajeed76, the creator of Charmer. I want to be upfront with you: I'm not perfect, and neither is this
code. Like any developer, I make mistakes, and there's always room for improvement in the codebase. I believe in
continuous learning and getting better with each iteration.

If you find issues, have suggestions for better implementations, or see ways to improve the code structure - please
don't hesitate to speak up! I'm here to learn and grow alongside this project, and I value every contribution and piece
of feedback.

Remember: perfect code doesn't exist, but better code does. Let's work together to make Charmer better, one commit at a
time.

## 🌟 Ways to Contribute

There are many ways you can contribute to Charmer:

1. **Code Contributions**: Implement new features or fix bugs
2. **Documentation**: Improve existing docs or write new guides
3. **Bug Reports**: Submit detailed bug reports
4. **Feature Requests**: Suggest new features or improvements
5. **Examples**: Create example implementations
6. **Testing**: Write tests and identify edge cases

## 🚀 Getting Started

### Prerequisites

- Go 1.24 or higher
- Basic understanding of Go programming
- Basic familiarity with [Charm](https://github.com/charmbracelet) libraries (If you want to improve TUI)
- Understanding of TUI (Terminal User Interface) concepts (If you want to improve TUI)

### Setting Up Your Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/charmer.git
   cd charmer
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/ImGajeed76/charmer.git
   ```
4. Install dependencies:
   ```bash
   go mod download
   ```

## 📝 Development Workflow

1. **Create a Branch**:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-fix-name
   ```

2. **Make Your Changes**:
    - Follow the Go coding standards
    - Keep your changes focused and atomic
    - Add tests for new functionality
    - Update documentation as needed

3. **Test Your Changes**:
   ```bash
   go test ./...
   go generate
   ```

4. **Commit Your Changes**:
   ```bash
   git commit -m "type: brief description of changes"
   ```
   Commit message types:
    - `feat`: New feature
    - `fix`: Bug fix
    - `docs`: Documentation changes
    - `test`: Adding or modifying tests
    - `refactor`: Code refactoring
    - `style`: Formatting changes
    - `chore`: Maintenance tasks

5. **Push and Create a Pull Request**:
   ```bash
   git push origin your-branch-name
   ```

## 🎨 Code Style Guidelines

### Go Code

- Follow the standard Go formatting guidelines
- Use `gofmt` to format your code
- Follow idiomatic Go practices
- Use meaningful variable and function names
- Add comments for non-obvious code sections

### Annotations

When adding new annotations:

- Use PascalCase for annotation names
- Document the annotation's purpose and usage
- Add examples in the documentation
- Follow the existing annotation pattern:
  ```go
  // @Charm
  // @Title "Your Feature Title"
  // @Description "A clear description of what your feature does"
  func YourFeature() {
      // Implementation
  }
  ```

## 📚 Documentation Guidelines

### Writing Documentation

- Use clear, concise language
- Include code examples where appropriate
- Follow Markdown best practices
- Update the mkdocs configuration if adding new pages
- Test documentation locally using mkdocs:
  ```bash
  mkdocs serve
  ```

### Example Documentation Format

```markdown
# Feature Name

## Overview

Brief description of the feature.

## Usage

    ```go
    // Code example
    ```

## Parameters

- `param1`: Description of first parameter
- `param2`: Description of second parameter

## Examples

Practical examples of feature usage.

```

## 🐛 Reporting Issues

When reporting issues, please include:

1. Charmer version
2. Go version
3. Operating system
4. Steps to reproduce
5. Expected vs actual behavior
6. Any relevant code snippets
7. Error messages (if any)

## 🎯 Pull Request Guidelines

**Before Submitting**:

   - Ensure all tests pass
   - Update documentation if needed
   - Add tests for new features
   - Follow code style guidelines
   - Rebase on latest main branch

**PR Description**:

   - Clearly describe the changes
   - Link to related issues
   - Include screenshots for UI changes
   - List breaking changes (if any)

**Review Process**:

   - Address review comments promptly
   - Keep discussions focused and professional
   - Update your PR as needed

## 🎉 Recognition

Contributors will be:
- Listed in the project's CONTRIBUTORS.md file
- Mentioned in release notes for significant contributions
- Credited in documentation where appropriate

## ❓ Getting Help

- Check existing issues and discussions
- Reach out to maintainers
- Read through our [Wiki](../../wiki) (coming soon)

## 📜 License Agreements

- All contributions must be licensed under GPL-3.0
- You must have the right to license your contribution
- Include copyright notices where appropriate

Thank you for contributing to Charmer! Together, we can make terminal applications more beautiful and easier to create. 🌟

---
For any questions not covered here, please open an issue or reach out to the maintainers.