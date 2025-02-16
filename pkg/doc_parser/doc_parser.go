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

			currentKey = parts[0]

			switch currentKey {
			case "Title":
				if len(parts) != 2 {
					continue
				}
				value := strings.TrimSpace(parts[1])
				docs.Title = value
			case "Description":
				if len(parts) != 2 {
					description.WriteString("")
					continue
				}
				value := strings.TrimSpace(parts[1])
				description.WriteString(value)
			}
			continue
		}

		// If we're in a description and find a non-empty continuation line
		if currentKey == "Description" {
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
