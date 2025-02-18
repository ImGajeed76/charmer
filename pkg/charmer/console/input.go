package console

import (
	"fmt"
	constants "github.com/ImGajeed76/charmer/internal"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"regexp"
	"strings"
)

var (
	// Style definitions
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(constants.Theme.PrimaryColor)).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(constants.Theme.PrimaryColor))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(constants.Theme.ErrorColor)).
			Italic(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(constants.Theme.TertiaryColor)).
			Italic(true)

	placeholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(constants.Theme.TertiaryColor))
)

// InputOptions allows customization of the input behavior
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

// DefaultInputOptions returns the default options
func DefaultInputOptions() InputOptions {
	return InputOptions{
		Prompt:     "Enter value:",
		CharLimit:  156,
		Width:      20,
		Required:   false,
		RegexError: "Input format is invalid",
	}
}

// Input takes a prompt and optional options, returns the validated user input
func Input(opts ...InputOptions) (string, error) {
	fmt.Print("\033[H\033[2J")
	options := DefaultInputOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	p := tea.NewProgram(initialModel(options))
	m, err := p.Run()
	if err != nil {
		return "", err
	}
	fmt.Print("\033[H\033[2J")

	finalModel := m.(inputModel)
	if finalModel.quitted {
		return "", fmt.Errorf("input cancelled")
	}
	return finalModel.textInput.Value(), nil
}

type inputModel struct {
	textInput textinput.Model
	options   InputOptions
	regex     *regexp.Regexp
	quitted   bool
	err       error
}

func initialModel(options InputOptions) inputModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = options.CharLimit
	ti.Width = options.Width

	// Style the text input
	ti.Prompt = "" // We'll handle the prompt separately
	ti.TextStyle = inputStyle
	ti.PlaceholderStyle = placeholderStyle

	if options.Default != "" {
		ti.SetValue(options.Default)
	}
	if options.Placeholder != "" {
		ti.Placeholder = options.Placeholder
	}

	var letterRegex *regexp.Regexp
	if options.Regex != "" {
		letterRegex = regexp.MustCompile(options.Regex)
	}

	return inputModel{
		textInput: ti,
		options:   options,
		regex:     letterRegex,
	}
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) validateInput(input string) (bool, string) {
	if m.options.Required && strings.TrimSpace(input) == "" {
		return false, "Input is required"
	}
	if m.regex != nil && input != "" && !m.regex.MatchString(input) {
		return false, m.options.RegexError
	}
	return true, ""
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if valid, _ := m.validateInput(m.textInput.Value()); valid {
				return m, tea.Quit
			}
			return m, nil
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitted = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	var builder strings.Builder

	// Add the prompt with styling
	builder.WriteString(promptStyle.Render(m.options.Prompt))
	builder.WriteString("\n\n")

	// Add the input field
	builder.WriteString(m.textInput.View())
	builder.WriteString("\n\n")

	// Add error message if validation fails
	if valid, errMsg := m.validateInput(m.textInput.Value()); !valid && m.textInput.Value() != "" {
		builder.WriteString(errorStyle.Render(errMsg))
		builder.WriteString("\n")
	}

	// Add hint text
	builder.WriteString(hintStyle.Render("(esc to cancel)"))
	builder.WriteString("\n")

	return builder.String()
}

// Example usage:
/*
func main() {
    // Simple usage
    value1, _ := Input()

    // Custom options with colors
    value2, _ := Input(InputOptions{
        Prompt:      "Enter your name:",
        Regex:       "^[a-zA-Z ]+$",
        RegexError:  "Please enter letters and spaces only",
        Default:     "John",
        Placeholder: "Enter name here",
        Required:    true,
    })
}
*/
