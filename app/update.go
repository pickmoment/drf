package app

import (
	"time"

	"github.com/pickmoment/drf/git"

	tea "github.com/charmbracelet/bubbletea"
)

// tickMsg is sent on every tick (100ms) for status expiry and async polling.
type tickMsg struct{}

// editorFinishedMsg is sent when a terminal editor process exits.
type editorFinishedMsg struct{ err error }

// gitStatusMsg is sent when the async git status refresh completes.
type gitStatusMsg struct {
	status *git.GitStatus
	dir    string
}

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

	case editorFinishedMsg:
		if msg.err != nil {
			m.SetStatusError("편집기 오류: " + msg.err.Error())
		}
		return tickCmd()

	case gitStatusMsg:
		if m.Git != nil && msg.dir == m.CurrentDir {
			m.Git.Status = msg.status
			if msg.status != nil {
				if m.Git.StagedIdx >= len(msg.status.Staged) {
					m.Git.StagedIdx = max(0, len(msg.status.Staged)-1)
				}
				if m.Git.UnstagedIdx >= len(msg.status.Unstaged) {
					m.Git.UnstagedIdx = max(0, len(msg.status.Unstaged)-1)
				}
			}
		}
		return nil

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
	case ModeOpenChoice:
		return m.handleKeyOpenChoice(key)
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
