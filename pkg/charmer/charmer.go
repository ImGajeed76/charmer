package charmer

import (
	"github.com/ImGajeed76/charmer/pkg/charmer/console"
	"github.com/ImGajeed76/charmer/pkg/charmer/models"
	tea "github.com/charmbracelet/bubbletea"
	"log"
	"strings"
)

func Run(charms map[string]models.CharmFunc) {
	selectedPath := ""

	m := console.NewCharmSelectorModel(charms, &selectedPath)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	// get the selected charm and execute it
	if selectedPath != "" {
		selectedPath = strings.TrimSuffix(selectedPath, "/")
		charm := charms[selectedPath]
		charm.Execute.(func())()
	}
}
