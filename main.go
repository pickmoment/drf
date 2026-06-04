package main

import (
	"fmt"
	"os"

	"github.com/pickmoment/drf/app"

	tea "github.com/charmbracelet/bubbletea"
)

// program wraps *app.Model to satisfy the tea.Model interface.
type program struct {
	m *app.Model
}

func (p program) Init() tea.Cmd {
	return p.m.Init()
}

func (p program) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := p.m.Update(msg)
	if p.m.ShouldQuit {
		return p, tea.Quit
	}
	return p, cmd
}

func (p program) View() string {
	return p.m.View()
}

func main() {
	m, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "초기화 실패: %v\n", err)
		os.Exit(1)
	}

	// Pre-load preview for the first selected file
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24}) // default size until resize msg

	prog := tea.NewProgram(
		program{m: m},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "실행 오류: %v\n", err)
		os.Exit(1)
	}
}
