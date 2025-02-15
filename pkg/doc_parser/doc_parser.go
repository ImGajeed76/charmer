package doc_parser

import (
	"bufio"
	"strings"
)

type Docs struct {
	Title       string
	Description string
}

func ParseAnnotations(docstring string) Docs {
	docs := Docs{}
	scanner := bufio.NewScanner(strings.NewReader(docstring))

	var currentKey string
	var description strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimSpace(line)

		// If we find a new annotation
		if strings.HasPrefix(line, "@") {
			// If we were building a description, save it
			if currentKey == "Description" {
				docs.Description = strings.TrimSpace(description.String())
				description.Reset()
			}

			parts := strings.SplitN(line[1:], " ", 2)
			if len(parts) != 2 {
				continue
			}

			currentKey = parts[0]
			value := strings.TrimSpace(parts[1])

			switch currentKey {
			case "Title":
				docs.Title = value
			case "Description":
				description.WriteString(value)
			}
			continue
		}

		// If we're in a description and find a non-empty continuation line
		if currentKey == "Description" && line != "" {
			description.WriteString("\n") // Add nl between lines
			description.WriteString(line)
		}
	}

	// Save any remaining description
	if currentKey == "Description" {
		docs.Description = strings.TrimSpace(description.String())
	}

	return docs
}
