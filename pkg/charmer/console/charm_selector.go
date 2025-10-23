package console

import (
	"fmt"
	constants "github.com/ImGajeed76/charmer/internal"
	"github.com/charmbracelet/glamour"
	"golang.org/x/term"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/76creates/stickers/flexbox"
	"github.com/ImGajeed76/charmer/pkg/charmer/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// UI Constants
	defaultMaxEntries       = 5
	cardPadding             = 2
	cardHorizontalPadding   = 3
	topBarPadding           = 1
	maxEntriesOffset        = 7
	descriptionTotalPadding = 4
	descriptionBorderSpace  = 2
	descriptionTitleSpace   = 4
	descriptionWidthOffset  = 10
	maxCacheSize            = 100
)

// Panel focus constants
const (
	PanelLeft  = "left"
	PanelRight = "right"
)

// UI Styles configuration
var styles = struct {
	base             lipgloss.Style
	card             lipgloss.Style
	cardFocused      lipgloss.Style
	rightCard        lipgloss.Style
	rightCardFocused lipgloss.Style
	topBar           lipgloss.Style
	selectedItem     lipgloss.Style
	path             lipgloss.Style
	search           lipgloss.Style
	searchMatch      lipgloss.Style
	section          lipgloss.Style
	cursor           lipgloss.Style
	title            lipgloss.Style
	cwd              lipgloss.Style
	hover            lipgloss.Style
}{
	base: lipgloss.NewStyle().Padding(1),
	card: lipgloss.NewStyle().
		Padding(cardPadding, cardHorizontalPadding).
		Width(0).
		Height(0).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")),
	cardFocused: lipgloss.NewStyle().
		Padding(cardPadding, cardHorizontalPadding).
		Width(0).
		Height(0).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(constants.Theme.PrimaryColor)),
	rightCard: lipgloss.NewStyle().
		Padding(cardPadding, cardHorizontalPadding).
		Width(0).
		Height(0).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")),
	rightCardFocused: lipgloss.NewStyle().
		Padding(cardPadding, cardHorizontalPadding).
		Width(0).
		Height(0).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(constants.Theme.PrimaryColor)),
	topBar: lipgloss.NewStyle().
		Padding(topBarPadding).
		Foreground(lipgloss.Color(constants.Theme.SecondaryColor)).
		Align(lipgloss.Center),
	selectedItem: lipgloss.NewStyle().
		Foreground(lipgloss.Color(constants.Theme.PrimaryColor)).
		Bold(true).
		Background(lipgloss.Color("236")),
	path: lipgloss.NewStyle().
		Foreground(lipgloss.Color(constants.Theme.SecondaryColor)).
		Italic(true).
		Padding(0, 0, 1, 0),
	search: lipgloss.NewStyle().
		Foreground(lipgloss.Color(constants.Theme.PrimaryColor)).
		Bold(true).
		Padding(0, 0, 0, 0),
	searchMatch: lipgloss.NewStyle().
		Underline(true).
		Background(lipgloss.Color("237")),
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
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("236")).
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
	helpBar   *flexbox.Cell

	// Focus state
	focusedPanel string

	// Mouse state
	mouseX         int
	mouseY         int
	lastMousePress bool

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

	// Initialization state
	initialized bool
}

// NewCharmSelectorModel creates and initializes a new CharmSelectorModel
func NewCharmSelectorModel(charms map[string]models.CharmFunc, currentPath *string) *CharmSelectorModel {
	if currentPath == nil {
		empty := ""
		currentPath = &empty
	}

	// Initialize UI components
	topBar := flexbox.NewCell(1, 1).
		SetContent(styles.topBar.Render("Charmer")).
		SetStyle(styles.topBar)

	leftCard := flexbox.NewCell(1, 7).
		SetContent("Navigation").
		SetStyle(styles.cardFocused)

	rightCard := flexbox.NewCell(1, 7).
		SetContent("Description").
		SetStyle(styles.rightCard)

	helpBar := flexbox.NewCell(1, 1).
		SetStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Padding(0, 1))

	// Create flexbox layout
	fb := flexbox.New(0, 0)
	rows := []*flexbox.Row{
		fb.NewRow().AddCells(topBar),
		fb.NewRow().AddCells(leftCard, rightCard),
		fb.NewRow().AddCells(helpBar),
	}
	fb.AddRows(rows)

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		// Fallback to nil renderer if initialization fails
		renderer = nil
	}

	// Normalize the initial path
	*currentPath = normalizePath(*currentPath)

	return &CharmSelectorModel{
		charms:               charms,
		currentPath:          currentPath,
		options:              []string{},
		maxEntries:           defaultMaxEntries,
		searchTerm:           "",
		flexbox:              fb,
		topBar:               topBar,
		leftCard:             leftCard,
		rightCard:            rightCard,
		helpBar:              helpBar,
		focusedPanel:         PanelLeft,
		markdownRenderer:     renderer,
		descriptionCache:     make(map[string]string),
		descriptionLineCache: make(map[string][]string),
		mouseX:               1,
		mouseY:               1,
	}
}

// normalizePath ensures path uses forward slashes and has trailing slash if not empty
func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	// Convert backslashes to forward slashes
	path = filepath.ToSlash(path)
	// Ensure trailing slash
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

// getCurrentPath safely gets the current path value
func (m *CharmSelectorModel) getCurrentPath() string {
	if m.currentPath == nil {
		return ""
	}
	return *m.currentPath
}

// setCurrentPath safely sets the current path value
func (m *CharmSelectorModel) setCurrentPath(path string) {
	if m.currentPath != nil {
		*m.currentPath = path
	}
}

func (m *CharmSelectorModel) getCacheKey(option string) string {
	return m.getCurrentPath() + "|" + option
}

func (m *CharmSelectorModel) Init() tea.Cmd {
	m.updateOptions()

	// Update the TopBar to include the current working directory
	cwd, _ := os.Getwd()
	title := styles.title.Render(fmt.Sprintf("Charmer - v%s", constants.Version))
	cwdText := styles.cwd.Render(cwd)
	m.topBar.SetContent(title + "\n" + cwdText)

	// Get terminal size and initialize dimensions immediately
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w, h = 80, 24 // fallback
	}

	m.updateDimensions(w, h)

	m.prerenderDescription()
	m.updateDescriptionView()
	m.updateCardStyles()
	m.initialized = true

	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: w, Height: h}
	}
}

// updateDimensions updates all dimension-dependent values
func (m *CharmSelectorModel) updateDimensions(width, height int) {
	m.flexbox.SetWidth(width)
	m.flexbox.SetHeight(height)
	m.flexbox.ForceRecalculate()

	m.maxEntries = m.leftCard.GetHeight() - maxEntriesOffset
	if m.maxEntries < 1 {
		m.maxEntries = 1
	}

	m.descriptionMaxHeight = m.rightCard.GetHeight() -
		(descriptionTotalPadding + descriptionBorderSpace + descriptionTitleSpace)
	if m.descriptionMaxHeight < 1 {
		m.descriptionMaxHeight = 1
	}

	m.descriptionMaxWidth = m.rightCard.GetWidth() - descriptionWidthOffset
	if m.descriptionMaxWidth < 20 {
		m.descriptionMaxWidth = 20
	}

	// Recreate renderer with new width
	if renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.descriptionMaxWidth),
	); err == nil {
		m.markdownRenderer = renderer
	}
}

// updateCardStyles updates the card styles based on focus
func (m *CharmSelectorModel) updateCardStyles() {
	if m.focusedPanel == PanelLeft {
		m.leftCard.SetStyle(styles.cardFocused)
		m.rightCard.SetStyle(styles.rightCard)
	} else {
		m.leftCard.SetStyle(styles.card)
		m.rightCard.SetStyle(styles.rightCardFocused)
	}
}

// updateOptions filters and updates available options based on the current path and search term
func (m *CharmSelectorModel) updateOptions() {
	if m.searchTerm != "" {
		m.updateSearchOptions()
	} else {
		m.options = GetAvailablePathOptions(m.charms, m.getCurrentPath())
	}

	// Ensure cursor and offset are within valid bounds
	m.ensureValidCursorPosition()
}

// ensureValidCursorPosition ensures cursor and offset are within bounds
func (m *CharmSelectorModel) ensureValidCursorPosition() {
	if len(m.options) == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}

	// Ensure cursor + offset is within options
	if m.cursor+m.offset >= len(m.options) {
		if len(m.options) <= m.maxEntries {
			m.cursor = len(m.options) - 1
			m.offset = 0
		} else {
			m.offset = len(m.options) - m.maxEntries
			m.cursor = m.maxEntries - 1
		}
	}

	// Ensure cursor is within maxEntries
	if m.cursor >= m.maxEntries {
		m.cursor = m.maxEntries - 1
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
	currentPath := m.getCurrentPath()
	if _, isCharm := m.charms[strings.TrimSuffix(currentPath, "/")]; isCharm {
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
	if m.initialized {
		m.cleanup()
	}

	m.updateDimensions(msg.Width, msg.Height)
	m.prerenderDescription()
	m.updateDescriptionView()
	m.updateCardStyles()
	m.initialized = true

	return m, nil
}

// handleKeyPress processes keyboard input
func (m *CharmSelectorModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left":
		m.focusedPanel = PanelLeft
		m.updateCardStyles()
	case "right":
		m.focusedPanel = PanelRight
		m.updateCardStyles()
	case "up":
		if m.focusedPanel == PanelLeft {
			m.navigateUp()
		} else {
			m.scrollDescriptionUp()
		}
	case "down":
		if m.focusedPanel == PanelLeft {
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

	currentMousePress := msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress
	wasPressed := !m.lastMousePress && currentMousePress
	m.lastMousePress = currentMousePress

	// Determine which panel the mouse is over
	leftCardWidth := m.leftCard.GetWidth()
	mousePanel := PanelLeft
	if m.mouseX >= leftCardWidth {
		mousePanel = PanelRight
	}

	// Handle scroll based on panel
	if msg.Button == tea.MouseButtonWheelUp {
		if mousePanel == PanelLeft {
			m.navigateUp()
		} else {
			m.scrollDescriptionUp()
		}
	} else if msg.Button == tea.MouseButtonWheelDown {
		if mousePanel == PanelLeft {
			m.navigateDown()
		} else {
			m.scrollDescriptionDown()
		}
	}

	// Handle hover (only on left panel)
	if mousePanel == PanelLeft {
		m.updateHoverState()
	} else {
		m.isHovering = false
	}

	// Handle click
	if wasPressed {
		// Focus the clicked panel
		if m.focusedPanel != mousePanel {
			m.focusedPanel = mousePanel
			m.updateCardStyles()
		}

		// If clicking on left panel with hover
		if mousePanel == PanelLeft && m.isHovering && m.isValidIndex(m.hoverIndex) {
			if m.cursor+m.offset != m.hoverIndex {
				// First click: select the charm
				m.cursor = m.hoverIndex - m.offset
				if m.cursor < 0 {
					m.cursor = 0
				}
				m.descriptionOffset = 0
				m.prerenderDescription()
				m.updateDescriptionView()
			} else {
				// Second click on same charm: execute
				return m.handleEnter()
			}
		}
	}

	return m, nil
}

// updateHoverState updates hover state based on mouse position
func (m *CharmSelectorModel) updateHoverState() {
	if m.mouseX >= m.leftCard.GetWidth() {
		m.isHovering = false
		return
	}

	relativeY := m.mouseY - m.topBar.GetHeight() - 1
	if m.searchTerm != "" {
		relativeY--
	}
	relativeY -= 6

	if relativeY >= 0 && relativeY < m.maxEntries {
		m.hoverIndex = relativeY + m.offset
		m.isHovering = m.isValidIndex(m.hoverIndex)
	} else {
		m.isHovering = false
	}
}

// isValidIndex checks if an index is valid for the current options
func (m *CharmSelectorModel) isValidIndex(index int) bool {
	return index >= 0 && index < len(m.options)
}

func (m *CharmSelectorModel) navigateUp() {
	if m.cursor > 0 {
		m.cursor--
	} else if m.offset > 0 {
		m.offset--
	}
	m.descriptionOffset = 0
	m.prerenderDescription()
	m.updateDescriptionView()
}

func (m *CharmSelectorModel) navigateDown() {
	if !m.isValidIndex(m.cursor + m.offset) {
		return
	}

	if m.cursor < len(m.options)-1 && m.cursor < m.maxEntries-1 {
		m.cursor++
	} else if (m.cursor + m.offset) < len(m.options)-1 {
		m.offset++
	}
	m.descriptionOffset = 0
	m.prerenderDescription()
	m.updateDescriptionView()
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

	index := m.cursor + m.offset
	if !m.isValidIndex(index) {
		return m, nil
	}

	selectedOption := m.options[index]
	oldPath := m.getCurrentPath()

	if m.searchTerm != "" {
		m.setCurrentPath(selectedOption)
		m.searchTerm = ""
	} else if selectedOption == ".." {
		return m.handleBackspace()
	} else {
		newPath := normalizePath(filepath.Join(m.getCurrentPath(), selectedOption))
		m.setCurrentPath(newPath)
	}

	if oldPath != m.getCurrentPath() {
		m.cleanup()
	}

	m.updateOptions()
	m.descriptionOffset = 0
	m.resetNavigationState()
	m.prerenderDescription()
	m.updateDescriptionView()

	if len(m.options) == 1 && m.options[0] == ".." {
		return m, tea.Quit
	}

	return m, nil
}

func (m *CharmSelectorModel) handleBackspace() (tea.Model, tea.Cmd) {
	if m.searchTerm != "" {
		m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
		m.updateOptions()
		m.resetNavigationState()
	} else {
		m.navigateBack()
	}
	m.descriptionOffset = 0
	m.prerenderDescription()
	m.updateDescriptionView()
	return m, nil
}

func (m *CharmSelectorModel) handleEscape() (tea.Model, tea.Cmd) {
	if m.searchTerm != "" {
		m.searchTerm = ""
		m.updateOptions()
		m.resetNavigationState()
	} else {
		// Close Selector
		m.setCurrentPath("") // reset so no function gets called
		return m, tea.Quit
	}
	m.descriptionOffset = 0
	m.prerenderDescription()
	m.updateDescriptionView()
	return m, nil
}

func (m *CharmSelectorModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(msg.String()) == 1 && msg.Type == tea.KeyRunes {
		m.searchTerm += msg.String()
		m.updateOptions()
		m.resetNavigationState()
		m.prerenderDescription()
		m.updateDescriptionView()
	}
	return m, nil
}

func (m *CharmSelectorModel) resetNavigationState() {
	m.cursor = 0
	m.offset = 0
}

func (m *CharmSelectorModel) navigateBack() {
	currentPath := m.getCurrentPath()
	if currentPath == "" {
		return
	}

	currentPath = strings.TrimSuffix(currentPath, "/")
	lastSlash := strings.LastIndex(currentPath, "/")
	if lastSlash == -1 {
		currentPath = ""
	} else {
		currentPath = currentPath[:lastSlash+1]
	}

	m.setCurrentPath(currentPath)
	m.updateOptions()
	m.resetNavigationState()
	m.prerenderDescription()
	m.updateDescriptionView()
}

// View renders the UI
func (m *CharmSelectorModel) View() string {
	if !m.initialized {
		return "(move your mouse or press any key to fix the UI...)"
	}

	var leftCardContent strings.Builder

	// Show current path and search with section spacing
	pathSection := styles.section.Render(
		styles.path.Render("Charm Folder: /" + m.getCurrentPath()))
	leftCardContent.WriteString(pathSection)

	if m.searchTerm != "" {
		searchSection := styles.section.Render(
			styles.search.Render("Search: " + m.searchTerm))
		leftCardContent.WriteString(searchSection)
	}
	leftCardContent.WriteString("\n")

	m.renderNavigationOptions(&leftCardContent)
	m.leftCard.SetContent(leftCardContent.String())

	// Set help text based on focused panel
	helpText := m.getHelpText()
	m.helpBar.SetContent(helpText)

	return m.flexbox.Render()
}

// getHelpText returns appropriate help text based on state
func (m *CharmSelectorModel) getHelpText() string {
	var generalHelp, panelHelp string

	if m.searchTerm != "" {
		generalHelp = "Enter: Select | Type: Search | Backspace: Clear Search | Esc: Stop Search"
	} else {
		generalHelp = "Enter: Select | Type: Search | Backspace: Back | Esc: Quit"
	}

	if m.focusedPanel == PanelLeft {
		panelHelp = "←→: Switch Panel | ↑↓: Navigate Charms"
	} else {
		panelHelp = "←→: Switch Panel | ↑↓: Scroll Description"
	}

	return generalHelp + "  •  " + panelHelp
}

func (m *CharmSelectorModel) prerenderDescription() {
	index := m.cursor + m.offset
	if len(m.options) == 0 || !m.isValidIndex(index) {
		m.currentDescription = ""
		m.descriptionLines = nil
		m.lastSelectedOption = ""
		return
	}

	selectedOption := m.options[index]
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
	var fullPath string
	if m.searchTerm != "" {
		fullPath = selectedOption
	} else {
		fullPath = m.getCurrentPath() + selectedOption
	}
	if charm, ok := m.charms[fullPath]; ok {
		rendered := charm.Description

		// Use markdown renderer if available
		if m.markdownRenderer != nil {
			if md, err := m.markdownRenderer.Render(charm.Description); err == nil {
				rendered = md
			}
		}

		// Enforce cache size limit
		if len(m.descriptionCache) > maxCacheSize {
			m.cleanup()
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

	// Ensure offset is within bounds
	if m.descriptionOffset >= len(m.descriptionLines) {
		m.descriptionOffset = len(m.descriptionLines) - 1
	}
	if m.descriptionOffset < 0 {
		m.descriptionOffset = 0
	}

	// Calculate visible range
	startIdx := m.descriptionOffset
	endIdx := m.descriptionOffset + m.descriptionMaxHeight
	if endIdx > len(m.descriptionLines) {
		endIdx = len(m.descriptionLines)
	}

	var content strings.Builder
	content.Grow(m.descriptionMaxWidth * m.descriptionMaxHeight)

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
	}

	var fullPath string
	if m.searchTerm != "" {
		fullPath = option
	} else {
		fullPath = m.getCurrentPath() + option
	}
	if charm, ok := m.charms[fullPath]; ok {
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
			title = m.highlightSearchMatch(title, m.searchTerm)
		}

		// Highlight search matches in path
		if strings.Contains(strings.ToLower(path), strings.ToLower(m.searchTerm)) {
			path = m.highlightSearchMatch(path, m.searchTerm)
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

// highlightSearchMatch highlights matching text in the original string
func (m *CharmSelectorModel) highlightSearchMatch(text, searchTerm string) string {
	lowerText := strings.ToLower(text)
	lowerSearch := strings.ToLower(searchTerm)
	idx := strings.Index(lowerText, lowerSearch)

	if idx == -1 {
		return text
	}

	matchLen := len(searchTerm)
	return text[:idx] +
		styles.searchMatch.Render(text[idx:idx+matchLen]) +
		text[idx+matchLen:]
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
	currentPath := m.getCurrentPath()
	segment := strings.TrimPrefix(path, currentPath)
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
	// Clear caches to prevent memory leaks
	m.descriptionCache = make(map[string]string)
	m.descriptionLineCache = make(map[string][]string)
	m.lastSelectedOption = ""
}
