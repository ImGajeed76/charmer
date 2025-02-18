package console

import (
	"fmt"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"strings"
	"sync"
)

const (
	padding  = 2
	maxWidth = 80
)

type ProgressOptions struct {
	GradientColors []string
	Width          int
	Padding        int
}

func DefaultProgressOptions() ProgressOptions {
	return ProgressOptions{
		GradientColors: []string{"#5956e0", "#e86ef6"},
		Width:          maxWidth,
		Padding:        padding,
	}
}

type ProgressBar struct {
	Update func(total, count int64)
	Close  func()
	Finish func()
}

type progressMsg struct {
	total int64
	count int64
}

type progressModel struct {
	progress  progress.Model
	options   ProgressOptions
	percent   float64
	quitting  bool
	updateCh  chan progressMsg
	closeCh   chan struct{}
	closeOnce sync.Once
}

func (m *progressModel) Init() tea.Cmd {
	return func() tea.Msg {
		return <-m.updateCh
	}
}

func (m *progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - m.options.Padding*2 - 4
		if m.progress.Width > m.options.Width {
			m.progress.Width = m.options.Width
		}
		return m, nil

	case progressMsg:
		if msg.total == 0 {
			m.percent = 0
		} else {
			m.percent = float64(msg.count) / float64(msg.total)
		}
		cmd := m.progress.SetPercent(m.percent)

		// Check if we should quit
		select {
		case <-m.closeCh:
			return m, tea.Quit
		default:
		}

		// Return a command to read the next update
		return m, tea.Batch(
			cmd,
			func() tea.Msg {
				select {
				case msg := <-m.updateCh:
					return msg
				case <-m.closeCh:
					return tea.Quit
				}
			},
		)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m *progressModel) View() string {
	pad := strings.Repeat(" ", m.options.Padding)
	return "\n" + pad + m.progress.View() + "\n\n"
}

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

func NewProgressBar(opts ...ProgressOptions) *ProgressBar {
	options := DefaultProgressOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	updateCh := make(chan progressMsg)
	closeCh := make(chan struct{})

	p := progress.New(
		progress.WithGradient(options.GradientColors[0], options.GradientColors[1]),
		progress.WithWidth(options.Width),
	)

	m := &progressModel{
		progress: p,
		options:  options,
		updateCh: updateCh,
		closeCh:  closeCh,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if _, err := tea.NewProgram(m).Run(); err != nil {
			fmt.Println("Error running progress bar:", err)
			os.Exit(1)
		}
	}()

	return &ProgressBar{
		Update: func(total, count int64) {
			select {
			case <-closeCh:
				return
			default:
				updateCh <- progressMsg{total: total, count: count}
			}
		},
		Close: func() {
			m.closeOnce.Do(func() {
				close(closeCh)
			})
			wg.Wait()
		},
		Finish: func() {
			// Send 100% progress and then close
			select {
			case <-closeCh:
				return
			default:
				updateCh <- progressMsg{total: 1, count: 1}
				m.closeOnce.Do(func() {
					close(closeCh)
				})
				wg.Wait()
			}
		},
	}
}
