package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// tickMsg is sent on every tick (100ms) for status expiry and async polling.
type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// Init initializes the bubbletea program.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.WindowSize())
}

// Update handles messages and returns updated state + next command.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return nil

	case tickMsg:
		// Expire status message
		if m.Status != nil && m.Status.IsExpired() {
			m.Status = nil
		}
		// Poll async git operation
		if m.Git != nil && m.Git.AsyncCmd != nil {
			m.checkAsyncDone()
			if m.Git != nil {
				m.Git.SpinnerTick++
			}
		}
		return tickCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		m.handleMouse(tea.MouseEvent(msg))
		return nil
	}
	return nil
}

// handleKey dispatches key events based on current mode.
func (m *Model) handleKey(key tea.KeyMsg) tea.Cmd {
	// Global quit
	if key.Type == tea.KeyCtrlC {
		m.ShouldQuit = true
		return tea.Quit
	}

	switch m.Mode {
	case ModeFileList:
		// Bookmarks panel gets its own key handler when focused
		if m.FocusedPanel == PanelBookmarks {
			return m.handleKeyBookmarks(key)
		}
		return m.handleKeyFileList(key)
	case ModeViewer:
		return m.handleKeyViewer(key)
	case ModePathClipboard:
		return m.handleKeyClipboardPanel(key)
	case ModeOpenWith:
		return m.handleKeyOpenWith(key)
	case ModeSettings:
		return m.handleKeySettings(key)
	case ModeCommandPalette:
		return m.handleKeyPalette(key)
	case ModeHelp:
		return m.handleKeyHelp(key)
	case ModeGit:
		return m.handleKeyGit(key)
	case ModeFileManager:
		return m.handleKeyFileManager(key)
	}
	return nil
}

// handleMouse handles mouse events.
func (m *Model) handleMouse(mouse tea.MouseEvent) {
	switch mouse.Action {
	case tea.MouseActionPress:
		switch mouse.Button {
		case tea.MouseButtonWheelUp:
			if m.Mode == ModeViewer {
				if m.PreviewScroll > 0 {
					m.PreviewScroll -= 3
					if m.PreviewScroll < 0 {
						m.PreviewScroll = 0
					}
				}
			} else if m.Mode == ModeFileList {
				m.moveUp()
				m.loadPreviewForSelected()
			}
		case tea.MouseButtonWheelDown:
			if m.Mode == ModeViewer {
				if m.PreviewScroll < len(m.ViewerLines)-1 {
					m.PreviewScroll += 3
				}
			} else if m.Mode == ModeFileList {
				m.moveDown()
				m.loadPreviewForSelected()
			}
		}
	}
}
