package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	versionFile = "internal/version.go"
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	version := os.Args[1]

	// Ensure version starts with 'v'
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	// Validate semantic versioning
	if !isValidVersion(version) {
		printError("Invalid version format. Use format: v1.0.0 or 1.0.0")
		os.Exit(1)
	}

	// Show what will happen
	printInfo(fmt.Sprintf("This will update the version to %s", version))
	printInfo("Steps:")
	fmt.Println("  1. Update internal/version.go")
	fmt.Println("  2. Commit the change")
	fmt.Println("  3. Create git tag " + version)
	fmt.Println("  4. Push to remote")
	fmt.Println()

	// Confirm
	if !confirm("Continue?") {
		printWarning("Aborted")
		os.Exit(0)
	}

	// Check git status
	if hasUncommittedChanges() {
		printError("You have uncommitted changes. Please commit or stash them first.")
		os.Exit(1)
	}

	// Update version.go
	printStep("Updating version.go...")
	if err := updateVersionFile(version); err != nil {
		printError(fmt.Sprintf("Failed to update version file: %v", err))
		os.Exit(1)
	}
	printSuccess("✓ Updated version.go")

	// Commit
	printStep("Committing changes...")
	if err := gitCommit(version); err != nil {
		printError(fmt.Sprintf("Failed to commit: %v", err))
		os.Exit(1)
	}
	printSuccess("✓ Committed changes")

	// Create tag
	printStep("Creating git tag...")
	if err := gitTag(version); err != nil {
		printError(fmt.Sprintf("Failed to create tag: %v", err))
		os.Exit(1)
	}
	printSuccess(fmt.Sprintf("✓ Created tag %s", version))

	// Ask before pushing
	fmt.Println()
	if !confirm("Push to remote?") {
		printWarning("Skipped push. Don't forget to push manually:")
		fmt.Printf("  git push origin main\n")
		fmt.Printf("  git push origin %s\n", version)
		os.Exit(0)
	}

	// Push commit
	printStep("Pushing commit...")
	if err := gitPush(); err != nil {
		printError(fmt.Sprintf("Failed to push commit: %v", err))
		os.Exit(1)
	}
	printSuccess("✓ Pushed commit")

	// Push tag
	printStep("Pushing tag...")
	if err := gitPushTag(version); err != nil {
		printError(fmt.Sprintf("Failed to push tag: %v", err))
		os.Exit(1)
	}
	printSuccess("✓ Pushed tag")

	fmt.Println()
	printSuccess(fmt.Sprintf("🎉 Successfully released %s!", version))
	printInfo(fmt.Sprintf("Users can now use: go get github.com/ImGajeed76/charmer@%s", version))
}

func printUsage() {
	fmt.Println("Usage: go run scripts/version/main.go <version>")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run scripts/version/main.go 1.0.0")
	fmt.Println("  go run scripts/version/main.go v1.0.0")
	fmt.Println("  go run scripts/version/main.go 2.1.3")
}

func isValidVersion(version string) bool {
	// Match semantic versioning: v1.2.3, v1.2.3-beta, v1.2.3-alpha.1, etc.
	pattern := `^v\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$`
	matched, _ := regexp.MatchString(pattern, version)
	return matched
}

func updateVersionFile(version string) error {
	content := fmt.Sprintf(`package internal

var Version = "%s"
`, version)
	return os.WriteFile(versionFile, []byte(content), 0644)
}

func hasUncommittedChanges() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

func gitCommit(version string) error {
	// Add version.go
	if err := runCommand("git", "add", versionFile); err != nil {
		return err
	}

	// Commit
	message := fmt.Sprintf("chore: bump version to %s", version)
	return runCommand("git", "commit", "-m", message)
}

func gitTag(version string) error {
	message := fmt.Sprintf("Release %s", version)
	return runCommand("git", "tag", "-a", version, "-m", message)
}

func gitPush() error {
	return runCommand("git", "push", "origin", "HEAD")
}

func gitPushTag(version string) error {
	return runCommand("git", "push", "origin", version)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func confirm(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s%s (y/N): %s", colorYellow, question, colorReset)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func printSuccess(msg string) {
	fmt.Printf("%s%s%s\n", colorGreen, msg, colorReset)
}

func printError(msg string) {
	fmt.Printf("%s%s%s\n", colorRed, msg, colorReset)
}

func printWarning(msg string) {
	fmt.Printf("%s%s%s\n", colorYellow, msg, colorReset)
}

func printInfo(msg string) {
	fmt.Printf("%s%s%s\n", colorCyan, msg, colorReset)
}

func printStep(msg string) {
	fmt.Printf("%s%s%s\n", colorCyan, msg, colorReset)
}
