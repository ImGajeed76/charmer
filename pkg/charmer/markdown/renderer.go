package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Style definitions with improved colors and spacing
var (
	baseStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	headingStyles = map[int]lipgloss.Style{
		1: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF75B5")).
			MarginTop(2).
			MarginBottom(1),
		2: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF75B5")).
			MarginTop(2).
			MarginBottom(1),
		3: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF75B5")).
			MarginTop(1).
			MarginBottom(1),
		4: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF75B5")),
		5: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF75B5")),
		6: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF75B5")),
	}

	boldStyle = lipgloss.NewStyle().
			Bold(true)

	italicStyle = lipgloss.NewStyle().
			Italic(true)

	codeBlockStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2A2A2A")).
			Foreground(lipgloss.Color("#A9B1D6")).
			PaddingLeft(1).
			PaddingRight(1).
			MarginTop(1).
			MarginBottom(1)

	inlineCodeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2A2A2A")).
			Foreground(lipgloss.Color("#A9B1D6")).
			PaddingLeft(1).
			PaddingRight(1)

	blockquoteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C7C7C")).
			PaddingLeft(1).
			BorderLeft(true).
			BorderStyle(lipgloss.Border{
			Left: "│",
		}).
		BorderForeground(lipgloss.Color("#7C7C7C"))

	linkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7AA2F7")).
			Underline(true)
)

type lineType int

const (
	normalLine lineType = iota
	headingLine
	listItemLine
	codeBlockLine
	blockquoteLine
)

type lineInfo struct {
	content   string
	typ       lineType
	level     int
	indent    string
	listStyle string
}

// Improved word wrap that better handles indentation and preserves formatting
func wordWrap(text string, width int, indent string, preserveIndent bool) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var result strings.Builder
	currentLine := indent + words[0]
	effectiveWidth := width
	if preserveIndent {
		effectiveWidth -= len(indent)
	}

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) > effectiveWidth {
			result.WriteString(currentLine + "\n")
			if preserveIndent {
				currentLine = indent
			} else {
				currentLine = strings.Repeat(" ", len(indent))
			}
			currentLine += word
		} else {
			currentLine += " " + word
		}
	}
	result.WriteString(currentLine)

	return result.String()
}

// New function to parse line information
func parseLine(line string) lineInfo {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return lineInfo{typ: normalLine}
	}

	// Check for headings
	if match := regexp.MustCompile(`^(#{1,6})\s(.+)`).FindStringSubmatch(trimmed); match != nil {
		return lineInfo{
			typ:     headingLine,
			level:   len(match[1]),
			content: match[2],
		}
	}

	// Check for list items
	if match := regexp.MustCompile(`^(\s*)([-*+]|\d+\.)\s(.+)`).FindStringSubmatch(line); match != nil {
		indent := match[1]
		listStyle := match[2]
		content := match[3]
		return lineInfo{
			typ:       listItemLine,
			level:     (len(indent) / 2) + 1,
			indent:    indent,
			listStyle: listStyle,
			content:   content,
		}
	}

	// Check for blockquotes
	if strings.HasPrefix(trimmed, ">") {
		return lineInfo{
			typ:     blockquoteLine,
			content: strings.TrimPrefix(trimmed, ">"),
		}
	}

	return lineInfo{
		typ:     normalLine,
		content: line,
	}
}

// Improved inline formatting that preserves formatting across line breaks
func formatInline(text string) string {
	// Store formatting positions to preserve them during wrapping
	type format struct {
		start, end int
		style      lipgloss.Style
	}
	var formats []format

	// Handle inline code (protected from other formatting)
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllStringFunc(text, func(match string) string {
		code := match[1 : len(match)-1]
		return inlineCodeStyle.Render(code)
	})

	// Find all bold sections
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	for _, match := range boldRegex.FindAllStringSubmatchIndex(text, -1) {
		formats = append(formats, format{
			start: match[2],
			end:   match[3],
			style: boldStyle,
		})
	}

	// Find all italic sections
	italicRegex := regexp.MustCompile(`\*([^*]+)\*`)
	for _, match := range italicRegex.FindAllStringSubmatchIndex(text, -1) {
		formats = append(formats, format{
			start: match[2],
			end:   match[3],
			style: italicStyle,
		})
	}

	// Apply formatting in reverse order to handle nested formats
	for i := len(formats) - 1; i >= 0; i-- {
		f := formats[i]
		text = text[:f.start] + f.style.Render(text[f.start:f.end]) + text[f.end:]
	}

	// Handle links last
	text = regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`).ReplaceAllStringFunc(text, func(match string) string {
		parts := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`).FindStringSubmatch(match)
		text, url := parts[1], parts[2]
		return linkStyle.Render(text) + " (" + url + ")"
	})

	return text
}

// RenderMarkdown converts markdown text to formatted console output
func RenderMarkdown(markdown string, maxWidth int) string {
	lines := strings.Split(markdown, "\n")
	var output bytes.Buffer
	var inCodeBlock bool
	var codeBlockBuffer bytes.Buffer
	var prevLineEmpty bool

	effectiveWidth := maxWidth - baseStyle.GetPaddingLeft()

	for i, line := range lines {
		// Handle empty lines
		if strings.TrimSpace(line) == "" {
			if !inCodeBlock {
				if !prevLineEmpty {
					output.WriteString("\n")
				}
				prevLineEmpty = true
			} else {
				codeBlockBuffer.WriteString("\n")
			}
			continue
		}
		prevLineEmpty = false

		// Handle code blocks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if !inCodeBlock {
				inCodeBlock = true
				continue
			} else {
				inCodeBlock = false
				codeContent := codeBlockBuffer.String()
				// Preserve indentation in code blocks
				lines := strings.Split(codeContent, "\n")
				var processedLines []string
				for _, l := range lines {
					if strings.TrimSpace(l) != "" {
						processedLines = append(processedLines, l)
					}
				}
				output.WriteString(codeBlockStyle.Render(strings.Join(processedLines, "\n")))
				codeBlockBuffer.Reset()
				output.WriteString("\n")
				continue
			}
		}
		if inCodeBlock {
			codeBlockBuffer.WriteString(line + "\n")
			continue
		}

		// Parse line information
		info := parseLine(line)

		// Process the line based on its type
		switch info.typ {
		case headingLine:
			wrappedText := wordWrap(formatInline(info.content), effectiveWidth, "", false)
			output.WriteString(headingStyles[info.level].Render(wrappedText) + "\n")

		case listItemLine:
			bullet := info.listStyle
			if strings.Contains("-*+", bullet) {
				bullet = "•"
			}
			indent := strings.Repeat("  ", info.level-1) + bullet + " "
			wrappedText := wordWrap(formatInline(info.content), effectiveWidth-len(indent), indent, true)
			output.WriteString(baseStyle.Render(wrappedText) + "\n")

		case blockquoteLine:
			wrappedText := wordWrap(formatInline(info.content), effectiveWidth-4, "", false)
			output.WriteString(blockquoteStyle.Render(wrappedText) + "\n")

		default:
			wrappedText := wordWrap(formatInline(line), effectiveWidth, "", false)
			output.WriteString(baseStyle.Render(wrappedText) + "\n")
		}

		// Handle paragraph spacing
		if i < len(lines)-1 && strings.TrimSpace(lines[i+1]) == "" {
			output.WriteString("\n")
		}
	}

	return output.String()
}
