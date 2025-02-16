package console

import (
	"fmt"
	constants "github.com/ImGajeed76/charmer/pkg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(constants.Theme.PrimaryColor)).
			Bold(true)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(constants.Theme.SecondaryColor))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(constants.Theme.PrimaryColor)).
				Bold(true)
)

// ListSelectOptions allows customization of the list select behavior
type ListSelectOptions struct {
	Title string
}

// DefaultListSelectOptions returns the default options
func DefaultListSelectOptions() ListSelectOptions {
	return ListSelectOptions{
		Title: "Select an option:",
	}
}

// ListSelect takes a slice of strings and returns the selected index
func ListSelect(items []string, opts ...ListSelectOptions) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("no items provided")
	}

	fmt.Print("\033[H\033[2J") // Clear screen
	options := DefaultListSelectOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	p := tea.NewProgram(initialListModel(items, options))
	m, err := p.Run()
	if err != nil {
		return -1, err
	}
	fmt.Print("\033[H\033[2J") // Clear screen

	finalModel := m.(listModel)
	if finalModel.quitted {
		return -1, fmt.Errorf("selection cancelled")
	}
	return finalModel.cursor + finalModel.offset, nil
}

type listModel struct {
	items    []string
	cursor   int
	options  ListSelectOptions
	quitted  bool
	height   int
	maxItems int
	offset   int
}

func initialListModel(items []string, options ListSelectOptions) listModel {
	return listModel{
		items:   items,
		cursor:  0,
		options: options,
	}
}

func (m listModel) Init() tea.Cmd {
	return nil
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.maxItems = m.height - 4
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitted = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.offset > 0 {
				m.offset--
			}
		case "down", "j":
			if m.cursor < m.maxItems-1 {
				m.cursor++
			} else if m.offset+m.maxItems < len(m.items) {
				m.offset++
			}
		}
	}

	return m, nil
}

func (m listModel) View() string {
	var builder strings.Builder

	// Title
	builder.WriteString(titleStyle.Render(m.options.Title))
	builder.WriteString("\n\n")

	// Items
	cutItems := m.items
	if len(m.items) > m.maxItems {
		cutItems = m.items[m.offset : m.offset+m.maxItems]
	}

	for i, item := range cutItems {
		if i == m.cursor {
			builder.WriteString(selectedItemStyle.Render("▸ " + item))
		} else {
			builder.WriteString(itemStyle.Render("  " + item))
		}
		builder.WriteString("\n")
	}

	// Help text
	builder.WriteString("\n")
	builder.WriteString(hintStyle.Render("↑/↓ to move • enter to select • esc to cancel"))

	return builder.String()
}

// Example usage:
/*
func main() {
    options := []string{"Option 1", "Option 2", "Option 3"}

    // Simple usage
    selectedIndex, err := ListSelect(options)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    fmt.Printf("Selected: %s\n", options[selectedIndex])

    // With custom title
    selectedIndex, err = ListSelect(options, ListSelectOptions{
        Title: "Choose your favorite:",
    })
}
*/
