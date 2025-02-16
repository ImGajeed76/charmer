package console

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var (
	// Style definitions (reusing from original input)
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ADD8")).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
)

// YesNoOptions allows customization of the yes/no input behavior
type YesNoOptions struct {
	Prompt     string
	DefaultYes bool   // If true, "Yes" is pre-selected
	YesText    string // Custom text for "Yes" option
	NoText     string // Custom text for "No" option
}

// DefaultYesNoOptions returns the default options
func DefaultYesNoOptions() YesNoOptions {
	return YesNoOptions{
		Prompt:     "Confirm?",
		DefaultYes: true,
		YesText:    "Yes",
		NoText:     "No",
	}
}

// YesNo displays a yes/no prompt and returns the user's choice
func YesNo(opts ...YesNoOptions) (bool, error) {
	fmt.Print("\033[H\033[2J") // Clear screen
	options := DefaultYesNoOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	p := tea.NewProgram(initialYesNoModel(options))
	m, err := p.Run()
	if err != nil {
		return false, err
	}
	fmt.Print("\033[H\033[2J") // Clear screen

	finalModel := m.(yesNoModel)
	if finalModel.quitted {
		return false, fmt.Errorf("input cancelled")
	}
	return finalModel.yes, nil
}

type yesNoModel struct {
	options YesNoOptions
	yes     bool // Current selection (true = Yes, false = No)
	quitted bool
}

func initialYesNoModel(options YesNoOptions) yesNoModel {
	return yesNoModel{
		options: options,
		yes:     options.DefaultYes,
	}
}

func (m yesNoModel) Init() tea.Cmd {
	return nil
}

func (m yesNoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "right", "h", "l":
			m.yes = !m.yes
			return m, nil
		case "enter":
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.quitted = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m yesNoModel) View() string {
	var builder strings.Builder

	// Add the prompt
	builder.WriteString(promptStyle.Render(m.options.Prompt))
	builder.WriteString("\n\n")

	// Render Yes/No options
	yesStyle := unselectedStyle
	noStyle := unselectedStyle
	if m.yes {
		yesStyle = selectedStyle
	} else {
		noStyle = selectedStyle
	}

	builder.WriteString(yesStyle.Render(m.options.YesText))
	builder.WriteString("  ")
	builder.WriteString(noStyle.Render(m.options.NoText))
	builder.WriteString("\n\n")

	// Add hint text
	builder.WriteString(hintStyle.Render("(←/→ to move, enter to select, esc to cancel)"))
	builder.WriteString("\n")

	return builder.String()
}

// Example usage:
/*
func main() {
    // Simple usage
    result, _ := YesNo()

    // Custom options
    result, _ := YesNo(YesNoOptions{
        Prompt:     "Do you want to continue?",
        DefaultYes: false,
        YesText:    "Continue",
        NoText:     "Cancel",
    })
}
*/
