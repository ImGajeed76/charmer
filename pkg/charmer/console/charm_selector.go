package console

import (
	"fmt"
	constants "github.com/ImGajeed76/charmer/internal"
	"github.com/charmbracelet/glamour"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/76creates/stickers/flexbox"
	"github.com/ImGajeed76/charmer/pkg/charmer/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UI Styles configuration
var styles = struct {
	base         lipgloss.Style
	card         lipgloss.Style
	rightCard    lipgloss.Style
	topBar       lipgloss.Style
	selectedItem lipgloss.Style
	path         lipgloss.Style
	searchMatch  lipgloss.Style
	section      lipgloss.Style
	cursor       lipgloss.Style
	title        lipgloss.Style
	cwd          lipgloss.Style
	hover        lipgloss.Style
}{
	base: lipgloss.NewStyle().Padding(1),
	card: lipgloss.NewStyle().
		Padding(2, 3). // Increased padding
		Width(0).
		Height(0).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")), // Subtle border
	rightCard: lipgloss.NewStyle().
		Padding(2, 3).
		Width(0).
		Height(0).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(constants.Theme.PrimaryColor)),
	topBar: lipgloss.NewStyle().
		Padding(1).
		Foreground(lipgloss.Color(constants.Theme.SecondaryColor)).
		Align(lipgloss.Center),
	selectedItem: lipgloss.NewStyle().
		Foreground(lipgloss.Color(constants.Theme.PrimaryColor)).
		Bold(true).
		Background(lipgloss.Color("236")), // Subtle highlight background
	path: lipgloss.NewStyle().
		Foreground(lipgloss.Color(constants.Theme.SecondaryColor)).
		Italic(true).
		Padding(0, 0, 1, 0), // Added bottom padding
	searchMatch: lipgloss.NewStyle().
		Underline(true).
		Background(lipgloss.Color("237")), // Subtle highlight for search matches
	section: lipgloss.NewStyle().
		PaddingBottom(1),
	cursor: lipgloss.NewStyle().
		Foreground(lipgloss.Color(constants.Theme.PrimaryColor)).
		Bold(true),
	title: lipgloss.NewStyle().
		Foreground(lipgloss.Color(constants.Theme.SecondaryColor)).
		Bold(true),
	cwd: lipgloss.NewStyle().
		Foreground(lipgloss.Color("202")),
	hover: lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).  // Changed to a softer cyan color
		Background(lipgloss.Color("236")). // Added subtle background
		Bold(true),
}

// CharmSelectorItem represents a selectable item in the charm interface
type CharmSelectorItem struct {
	Title       string
	Path        string
	Description string
}

// CharmSelectorModel represents the application state
type CharmSelectorModel struct {
	charms      map[string]models.CharmFunc
	currentPath *string
	options     []string
	cursor      int
	offset      int
	maxEntries  int
	searchTerm  string

	// UI components
	flexbox   *flexbox.FlexBox
	topBar    *flexbox.Cell
	leftCard  *flexbox.Cell
	rightCard *flexbox.Cell

	// Mouse state
	mouseX    int
	mouseY    int
	mouseDown bool

	// Description box
	descriptionOffset    int
	descriptionMaxHeight int
	descriptionMaxWidth  int
	currentDescription   string
	descriptionLines     []string
	markdownRenderer     *glamour.TermRenderer
	descriptionCache     map[string]string
	descriptionLineCache map[string][]string
	lastSelectedOption   string

	// Hover
	hoverIndex int
	isHovering bool
}

// NewCharmSelectorModel creates and initializes a new CharmSelectorModel
func NewCharmSelectorModel(charms map[string]models.CharmFunc, currentPath *string) *CharmSelectorModel {
	// Initialize UI components
	topBar := flexbox.NewCell(1, 1).
		SetContent(styles.topBar.Render("Charmer")).
		SetStyle(styles.topBar)

	leftCard := flexbox.NewCell(1, 7).
		SetContent("Navigation").
		SetStyle(styles.card)

	rightCard := flexbox.NewCell(1, 7).
		SetContent("Description").
		SetStyle(styles.rightCard)

	// Create flexbox layout
	fb := flexbox.New(0, 0)
	rows := []*flexbox.Row{
		fb.NewRow().AddCells(topBar),
		fb.NewRow().AddCells(leftCard, rightCard),
	}
	fb.AddRows(rows)

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)

	return &CharmSelectorModel{
		charms:               charms,
		currentPath:          currentPath,
		options:              []string{},
		maxEntries:           5,
		searchTerm:           "",
		flexbox:              fb,
		topBar:               topBar,
		leftCard:             leftCard,
		rightCard:            rightCard,
		markdownRenderer:     renderer,
		descriptionCache:     make(map[string]string),
		descriptionLineCache: make(map[string][]string),
	}
}

func (m *CharmSelectorModel) getCacheKey(option string) string {
	return *m.currentPath + option
}

func (m *CharmSelectorModel) Init() tea.Cmd {
	m.updateOptions()

	// Update the TopBar to include the current working directory in a new line
	cwd, _ := os.Getwd()
	title := styles.title.Render(fmt.Sprintf("Charmer - v%s", constants.Version))
	cwd = styles.cwd.Render(cwd)
	m.topBar.SetContent(title + "\n" + cwd)

	return nil
}

// updateOptions filters and updates available options based on the current path and search term
func (m *CharmSelectorModel) updateOptions() {
	if m.searchTerm != "" {
		m.updateSearchOptions()
	} else {
		m.options = GetAvailablePathOptions(m.charms, *m.currentPath)
	}
}

// updateSearchOptions updates options based on the current search term
func (m *CharmSelectorModel) updateSearchOptions() {
	filtered := make([]string, 0)
	searchLower := strings.ToLower(m.searchTerm)

	for path, charm := range m.charms {
		if m.matchesSearch(path, charm, searchLower) {
			filtered = append(filtered, path)
		}
	}

	sort.Strings(filtered)
	m.options = filtered
}

// matchesSearch checks if a charm matches the search criteria
func (m *CharmSelectorModel) matchesSearch(path string, charm models.CharmFunc, searchTerm string) bool {
	return strings.Contains(strings.ToLower(path), searchTerm) ||
		strings.Contains(strings.ToLower(charm.Title), searchTerm) ||
		strings.Contains(strings.ToLower(charm.Description), searchTerm)
}

// Update handles UI state updates based on user input
func (m *CharmSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check if we've reached a terminal charm
	if _, isCharm := m.charms[strings.TrimSuffix(*m.currentPath, "/")]; isCharm {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.MouseMsg:
		return m.handleMouseMsg(msg)
	}

	return m, nil
}

// handleWindowSize updates the UI layout based on window size
func (m *CharmSelectorModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.flexbox.SetWidth(msg.Width)
	m.flexbox.SetHeight(msg.Height)
	m.flexbox.ForceRecalculate()
	m.maxEntries = m.leftCard.GetHeight() - 7

	totalPadding := 4 // top + bottom padding
	borderSpace := 2  // top + bottom borders
	titleSpace := 4   // space for title
	m.descriptionMaxHeight = m.rightCard.GetHeight() - (totalPadding + borderSpace + titleSpace)
	m.descriptionMaxWidth = m.rightCard.GetWidth() - 10

	m.markdownRenderer, _ = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.descriptionMaxWidth),
	)

	m.cleanup()

	m.prerenderDescription()
	m.updateDescriptionView()
	return m, nil
}

// handleKeyPress processes keyboard input
func (m *CharmSelectorModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		*m.currentPath = "" // reset so no function gets called
		return m, tea.Quit
	case "up":
		if m.mouseX < m.leftCard.GetWidth() {
			m.navigateUp()
		} else {
			m.scrollDescriptionUp()
		}
	case "down":
		if m.mouseX < m.leftCard.GetWidth() {
			m.navigateDown()
		} else {
			m.scrollDescriptionDown()
		}
	case "enter":
		return m.handleEnter()
	case "backspace":
		return m.handleBackspace()
	case "esc":
		return m.handleEscape()
	default:
		return m.handleSearchInput(msg)
	}
	return m, nil
}

func (m *CharmSelectorModel) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	m.mouseX = msg.X
	m.mouseY = msg.Y

	lastMouseDown := m.mouseDown
	m.mouseDown = false

	if msg.Button == tea.MouseButtonWheelUp {
		if m.mouseX < m.leftCard.GetWidth() {
			m.navigateUp()
		} else {
			m.scrollDescriptionUp()
		}
	} else if msg.Button == tea.MouseButtonWheelDown {
		if m.mouseX < m.leftCard.GetWidth() {
			m.navigateDown()
		} else {
			m.scrollDescriptionDown()
		}
	} else if msg.Button == tea.MouseButtonLeft {
		m.mouseDown = true
	}

	if !lastMouseDown && m.mouseDown {
		if m.isHovering && m.cursor != m.hoverIndex {
			m.cursor = m.hoverIndex
			m.descriptionOffset = 0
			m.prerenderDescription()
		} else if m.isHovering && m.cursor == m.hoverIndex {
			m.handleEnter()
		}
	}

	if m.mouseX < m.leftCard.GetWidth() {
		relativeY := msg.Y - m.topBar.GetHeight() - 1
		if m.searchTerm != "" {
			relativeY--
		}
		relativeY -= 6

		if relativeY >= 0 && relativeY < m.maxEntries {
			m.hoverIndex = relativeY + m.offset
			m.isHovering = m.hoverIndex < len(m.options)
		} else {
			m.isHovering = false
		}
	} else {
		m.isHovering = false
	}

	return m, nil
}

func (m *CharmSelectorModel) navigateUp() {
	if m.cursor > 0 {
		m.cursor--
	} else if m.offset > 0 {
		m.offset--
	}
	m.descriptionOffset = 0
	m.prerenderDescription()
}

func (m *CharmSelectorModel) navigateDown() {
	if m.cursor < len(m.options)-1 && m.cursor < m.maxEntries-1 {
		m.cursor++
	} else if (m.cursor + m.offset) < len(m.options)-1 {
		m.offset++
	}
	m.descriptionOffset = 0
	m.prerenderDescription()
}

func (m *CharmSelectorModel) scrollDescriptionUp() {
	if m.descriptionOffset > 0 {
		m.descriptionOffset--
		m.updateDescriptionView()
	}
}

func (m *CharmSelectorModel) scrollDescriptionDown() {
	if len(m.descriptionLines) > m.descriptionMaxHeight &&
		m.descriptionOffset < len(m.descriptionLines)-m.descriptionMaxHeight {
		m.descriptionOffset++
		m.updateDescriptionView()
	}
}

func (m *CharmSelectorModel) handleEnter() (tea.Model, tea.Cmd) {
	if len(m.options) == 0 {
		return m, nil
	}

	selectedOption := m.options[m.cursor+m.offset]
	oldPath := *m.currentPath

	if m.searchTerm != "" {
		*m.currentPath = selectedOption
	} else if selectedOption == ".." {
		return m.handleBackspace()
	} else {
		*m.currentPath = filepath.Join(*m.currentPath, selectedOption) + "/"
		// Windows Problem: filepath.Join() uses backslashes on Windows, change to forward slashes
		*m.currentPath = strings.ReplaceAll(*m.currentPath, "\\", "/")
		m.updateOptions()
	}

	if oldPath != *m.currentPath {
		m.cleanup()
	}

	m.descriptionOffset = 0
	m.resetNavigationState()
	m.prerenderDescription()

	if len(m.options) == 0 {
		return m, tea.Quit
	}

	return m, nil
}

func (m *CharmSelectorModel) handleBackspace() (tea.Model, tea.Cmd) {
	if m.searchTerm != "" {
		m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
		m.updateOptions()
	} else {
		m.navigateBack()
	}
	m.descriptionOffset = 0
	m.prerenderDescription()
	return m, nil
}

func (m *CharmSelectorModel) handleEscape() (tea.Model, tea.Cmd) {
	if m.searchTerm != "" {
		m.searchTerm = ""
		m.updateOptions()
	} else {
		m.navigateBack()
	}
	m.descriptionOffset = 0
	m.prerenderDescription()
	return m, nil
}

func (m *CharmSelectorModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(msg.String()) == 1 && msg.Type == tea.KeyRunes {
		m.searchTerm += msg.String()
		m.updateOptions()
		m.resetNavigationState()
		m.prerenderDescription()
	}
	return m, nil
}

func (m *CharmSelectorModel) resetNavigationState() {
	m.cursor = 0
	m.offset = 0
}

func (m *CharmSelectorModel) navigateBack() {
	if *m.currentPath == "" {
		return
	}

	*m.currentPath = strings.TrimSuffix(*m.currentPath, "/")
	lastSlash := strings.LastIndex(*m.currentPath, "/")
	if lastSlash == -1 {
		*m.currentPath = ""
	} else {
		*m.currentPath = (*m.currentPath)[:lastSlash+1]
	}

	m.updateOptions()
	m.resetNavigationState()
	m.prerenderDescription()
}

// View renders the UI
func (m *CharmSelectorModel) View() string {
	var leftCardContent strings.Builder

	// Show current path and search with section spacing
	pathSection := styles.section.Render(
		styles.path.Render("Current Path: " + *m.currentPath))
	leftCardContent.WriteString(pathSection)

	if m.searchTerm != "" {
		searchSection := styles.section.Render(
			fmt.Sprintf("Search: %s", m.searchTerm))
		leftCardContent.WriteString(searchSection)
	}
	leftCardContent.WriteString("\n")

	m.renderNavigationOptions(&leftCardContent)
	m.leftCard.SetContent(leftCardContent.String())
	return m.flexbox.Render()
}

func (m *CharmSelectorModel) prerenderDescription() {
	if len(m.options) == 0 || m.cursor+m.offset >= len(m.options) {
		m.currentDescription = ""
		m.descriptionLines = nil
		return
	}

	selectedOption := m.options[m.cursor+m.offset]
	cacheKey := m.getCacheKey(selectedOption)

	// If the selected option hasn't changed, no need to update
	if m.lastSelectedOption == cacheKey {
		return
	}
	m.lastSelectedOption = cacheKey

	// Check if we have a cached version
	if cachedLines, ok := m.descriptionLineCache[cacheKey]; ok {
		m.descriptionLines = cachedLines
		m.currentDescription = m.descriptionCache[cacheKey]
		return
	}

	// Only render if it's actually a charm
	if charm, ok := m.charms[*m.currentPath+selectedOption]; ok {
		rendered, err := m.markdownRenderer.Render(charm.Description)
		if err != nil {
			m.currentDescription = "Error rendering description"
			m.descriptionLines = []string{"Error rendering description"}
			return
		}

		// Cache the results
		m.descriptionCache[cacheKey] = rendered
		m.descriptionLineCache[cacheKey] = strings.Split(rendered, "\n")

		m.currentDescription = rendered
		m.descriptionLines = m.descriptionLineCache[cacheKey]
	} else {
		m.currentDescription = ""
		m.descriptionLines = nil
	}
}

func (m *CharmSelectorModel) updateDescriptionView() {
	if len(m.descriptionLines) == 0 {
		m.rightCard.SetContent("")
		return
	}

	// Calculate visible range
	startIdx := m.descriptionOffset
	endIdx := m.descriptionOffset + m.descriptionMaxHeight
	if endIdx > len(m.descriptionLines) {
		endIdx = len(m.descriptionLines)
	}

	var content strings.Builder
	content.Grow(m.descriptionMaxWidth * m.descriptionMaxHeight) // Preallocate buffer

	// Add scroll indicators and content
	if m.descriptionOffset > 0 {
		content.WriteString("↑ More above\n\n")
	}

	// Join only the visible lines
	visibleLines := m.descriptionLines[startIdx:endIdx]
	visibleContent := strings.Join(visibleLines, "\n")

	// Calculate required padding
	visibleLineCount := len(visibleLines)
	if visibleLineCount < m.descriptionMaxHeight {
		padding := strings.Repeat("\n", m.descriptionMaxHeight-visibleLineCount)
		visibleContent += padding
	}

	content.WriteString(visibleContent)

	if endIdx < len(m.descriptionLines) {
		content.WriteString("\n\n↓ More below")
	}

	m.rightCard.SetContent(content.String())
}

// renderNavigationOptions renders the navigation options in the left card
func (m *CharmSelectorModel) renderNavigationOptions(content *strings.Builder) {
	if m.offset > 0 {
		content.WriteString("  ...\n")
	} else {
		content.WriteString("\n")
	}

	// Render visible options
	for i, option := range m.options {
		if i < m.offset || i >= m.offset+m.maxEntries {
			continue
		}

		m.renderOption(content, i, option)
	}

	if m.offset+m.maxEntries < len(m.options) {
		content.WriteString("  ...")
	}
}

// renderOption renders a single navigation option
func (m *CharmSelectorModel) renderOption(content *strings.Builder, index int, option string) {
	cursor := " "
	if index == m.cursor+m.offset {
		cursor = ">"
		if _, ok := m.charms[*m.currentPath+option]; ok {
			m.updateDescriptionView()
		} else {
			m.rightCard.SetContent("")
		}
	}

	if charm, ok := m.charms[*m.currentPath+option]; ok {
		m.renderCharmOption(content, index, option, cursor, charm)
	} else {
		m.renderPathOption(content, index, option, cursor)
	}
}

// renderCharmOption renders a charm option with title and path
func (m *CharmSelectorModel) renderCharmOption(content *strings.Builder, index int, option, cursor string, charm models.CharmFunc) {
	var optionText string
	if m.searchTerm != "" {
		title := charm.Title
		path := option

		// Highlight search matches in title
		if strings.Contains(strings.ToLower(title), strings.ToLower(m.searchTerm)) {
			idx := strings.Index(strings.ToLower(title), strings.ToLower(m.searchTerm))
			matchLen := len(m.searchTerm)
			title = title[:idx] +
				styles.searchMatch.Render(title[idx:idx+matchLen]) +
				title[idx+matchLen:]
		}

		// Highlight search matches in path
		if strings.Contains(strings.ToLower(path), strings.ToLower(m.searchTerm)) {
			idx := strings.Index(strings.ToLower(path), strings.ToLower(m.searchTerm))
			matchLen := len(m.searchTerm)
			path = path[:idx] +
				styles.searchMatch.Render(path[idx:idx+matchLen]) +
				path[idx+matchLen:]
		}

		optionText = fmt.Sprintf("%s %s (%s)",
			styles.cursor.Render(cursor),
			title,
			path)
	} else {
		segment := m.getPathSegment(option)
		optionText = fmt.Sprintf("%s %s (%s)",
			styles.cursor.Render(cursor),
			charm.Title,
			segment)
	}

	switch {
	case m.isHovering && index == m.hoverIndex:
		optionText = styles.hover.Render(optionText)
	case index == m.cursor+m.offset:
		optionText = styles.selectedItem.Render(optionText)
	}

	content.WriteString(optionText + "\n")
}

// renderPathOption renders a path option
func (m *CharmSelectorModel) renderPathOption(content *strings.Builder, index int, option, cursor string) {
	optionText := cursor + " " + option

	switch {
	case m.isHovering && index == m.hoverIndex:
		optionText = styles.hover.Render(optionText)
	case index == m.cursor+m.offset:
		optionText = styles.selectedItem.Render(optionText)
	}

	content.WriteString(optionText + "\n")
}

// getPathSegment extracts the relevant path segment for display
func (m *CharmSelectorModel) getPathSegment(path string) string {
	segment := strings.TrimPrefix(path, *m.currentPath)
	segment = strings.TrimPrefix(segment, "/")
	if strings.Contains(segment, "/") {
		segment = strings.Split(segment, "/")[0]
	}
	return segment
}

// GetAvailablePathOptions returns a sorted list of available path options
func GetAvailablePathOptions(charms map[string]models.CharmFunc, currentPath string) []string {
	uniqueOptions := make(map[string]bool)

	for path := range charms {
		if strings.HasPrefix(path, currentPath) {
			remaining := strings.TrimPrefix(path, currentPath)
			remaining = strings.TrimPrefix(remaining, "/")

			if remaining != "" {
				firstSegment := strings.Split(remaining, "/")[0]
				uniqueOptions[firstSegment] = true
			}
		}
	}

	options := make([]string, 0, len(uniqueOptions))
	for option := range uniqueOptions {
		options = append(options, option)
	}
	sort.Strings(options)

	if currentPath != "" {
		options = append([]string{".."}, options...)
	}

	return options
}

func (m *CharmSelectorModel) cleanup() {
	// Clear caches when path changes or on exit
	m.descriptionCache = make(map[string]string)
	m.descriptionLineCache = make(map[string][]string)
}
