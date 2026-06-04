package app

import (
	"os"
	"strconv"

	"github.com/pickmoment/drf/fs"
	"github.com/pickmoment/drf/git"
	"github.com/pickmoment/drf/state"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyFileList handles key events in file list mode.
func (m *Model) handleKeyFileList(key tea.KeyMsg) tea.Cmd {
	if m.IsSearching {
		return m.handleKeyFileListSearch(key)
	}

	switch key.Type {
	case tea.KeyUp:
		if m.FocusedPanel == PanelBookmarks {
			if m.BookmarkIndex > 0 {
				m.BookmarkIndex--
			}
		} else if m.FocusedPanel == PanelPathClipboard {
			if m.PathClipboardIdx > 0 {
				m.PathClipboardIdx--
			}
		} else {
			m.moveUp()
			m.loadPreviewForSelected()
		}
	case tea.KeyDown:
		if m.FocusedPanel == PanelBookmarks {
			if m.BookmarkIndex < len(m.Config.Bookmarks)-1 {
				m.BookmarkIndex++
			}
		} else if m.FocusedPanel == PanelPathClipboard {
			if m.PathClipboardIdx < len(m.PathClipboard)-1 {
				m.PathClipboardIdx++
			}
		} else {
			m.moveDown()
			m.loadPreviewForSelected()
		}
	case tea.KeyPgUp:
		m.jumpUp()
		m.loadPreviewForSelected()
	case tea.KeyPgDown:
		m.jumpDown()
		m.loadPreviewForSelected()
	case tea.KeyEnter:
		if m.FocusedPanel == PanelBookmarks {
			cmd := m.navigateToBookmark()
			m.loadPreviewForSelected()
			return cmd
		}
		if m.FocusedPanel == PanelPathClipboard {
			if m.PathClipboardIdx < len(m.PathClipboard) {
				cmd := m.navigateTo(m.PathClipboard[m.PathClipboardIdx])
				m.loadPreviewForSelected()
				return cmd
			}
			return nil
		}
		return m.enterOrOpen()
	case tea.KeyLeft:
		cmd := m.goParent()
		m.loadPreviewForSelected()
		return cmd
	case tea.KeyRight:
		if m.FocusedPanel == PanelFileList {
			entry := m.SelectedEntry()
			if entry != nil {
				if entry.IsDir {
					cmd := m.navigateTo(entry.Path)
					m.loadPreviewForSelected()
					return cmd
				}
				return m.openInViewer(entry.Path)
			}
		}
	case tea.KeyBackspace:
		cmd := m.goParent()
		m.loadPreviewForSelected()
		return cmd
	case tea.KeyTab:
		// Cycle focus
		switch m.FocusedPanel {
		case PanelFileList:
			if m.Config.Ui.ShowBookmarksPanel {
				m.FocusedPanel = PanelBookmarks
			} else if len(m.PathClipboard) > 0 {
				m.FocusedPanel = PanelPathClipboard
			}
		case PanelBookmarks:
			if len(m.PathClipboard) > 0 {
				m.FocusedPanel = PanelPathClipboard
			} else {
				m.FocusedPanel = PanelFileList
			}
		case PanelPathClipboard:
			m.FocusedPanel = PanelFileList
		}
	case tea.KeyHome:
		m.SelectedIndex = 0
		m.loadPreviewForSelected()
	case tea.KeyEnd:
		if len(m.FilteredIndices) > 0 {
			m.SelectedIndex = len(m.FilteredIndices) - 1
		}
		m.loadPreviewForSelected()
	case tea.KeyRunes:
		return m.handleKeyFileListRunes(key)
	case tea.KeySpace:
		// Open fullscreen viewer
		entry := m.SelectedEntry()
		if entry != nil && !entry.IsDir {
			return m.openInViewer(entry.Path)
		}
	case tea.KeyEsc:
		if m.FocusedPanel != PanelFileList {
			m.FocusedPanel = PanelFileList
		}
	}
	return nil
}

func (m *Model) handleKeyFileListSearch(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.IsSearching = false
		m.SearchQuery = ""
		m.updateFilter()
		m.loadPreviewForSelected()
	case tea.KeyEnter:
		m.IsSearching = false
		m.loadPreviewForSelected()
	case tea.KeyBackspace:
		if len(m.SearchQuery) > 0 {
			runes := []rune(m.SearchQuery)
			m.SearchQuery = string(runes[:len(runes)-1])
			m.updateFilter()
			m.loadPreviewForSelected()
		}
	case tea.KeyRunes:
		m.SearchQuery += string(key.Runes)
		m.updateFilter()
		m.loadPreviewForSelected()
	}
	return nil
}

func (m *Model) handleKeyFileListRunes(key tea.KeyMsg) tea.Cmd {
	if len(key.Runes) == 0 {
		return nil
	}
	ch := key.Runes[0]

	// Vim-style navigation
	if m.Config.Keymap.VimKeys {
		switch ch {
		case 'j':
			if m.FocusedPanel == PanelBookmarks {
				if m.BookmarkIndex < len(m.Config.Bookmarks)-1 {
					m.BookmarkIndex++
				}
			} else if m.FocusedPanel == PanelPathClipboard {
				if m.PathClipboardIdx < len(m.PathClipboard)-1 {
					m.PathClipboardIdx++
				}
			} else {
				m.moveDown()
				m.loadPreviewForSelected()
			}
			return nil
		case 'k':
			if m.FocusedPanel == PanelBookmarks {
				if m.BookmarkIndex > 0 {
					m.BookmarkIndex--
				}
			} else if m.FocusedPanel == PanelPathClipboard {
				if m.PathClipboardIdx > 0 {
					m.PathClipboardIdx--
				}
			} else {
				m.moveUp()
				m.loadPreviewForSelected()
			}
			return nil
		case 'h':
			cmd := m.goParent()
			m.loadPreviewForSelected()
			return cmd
		case 'l':
			entry := m.SelectedEntry()
			if entry != nil {
				if entry.IsDir {
					cmd := m.navigateTo(entry.Path)
					m.loadPreviewForSelected()
					return cmd
				}
				return m.openInViewer(entry.Path)
			}
			return nil
		case 'g':
			// g → Git 모드 진입 (파일 목록에서는 gg 감지 없음)
			m.Mode = ModeGit
			if m.Git == nil {
				m.Git = state.NewGitState(m.CurrentDir)
			}
			m.gitRefresh()
			return nil
		case 'G':
			if len(m.FilteredIndices) > 0 {
				m.SelectedIndex = len(m.FilteredIndices) - 1
			}
			m.loadPreviewForSelected()
			return nil
		}
	}

	switch ch {
	case '/':
		m.IsSearching = true
		m.SearchQuery = ""
	case 'b':
		m.toggleBookmark()
	case 'B':
		m.FocusedPanel = PanelBookmarks
	case 'y':
		if path := m.SelectedPath(); path != "" {
			m.writeToClipboard(path)
			m.SetStatusSuccess("경로 복사됨: " + path)
		}
	case 'p':
		m.togglePathClipboard()
	case 'P':
		m.Mode = ModePathClipboard
		m.PathClipboardIdx = 0
	case 'o':
		m.Mode = ModeOpenWith
		m.OpenWithIndex = 0
	case 'e':
		// Open in editor
		path := m.SelectedPath()
		if path != "" {
			m.PendingEditorOpen = true
		}
	case 'n':
		m.fmStart()
		op := FmOpNewDir
		m.FmOperation = &op
		m.FmInput = ""
		m.FmCursor = 0
	case 'r':
		m.fmStart()
		op := FmOpRename
		m.FmOperation = &op
		if entry := m.SelectedEntry(); entry != nil {
			m.FmInput = entry.Name
			m.FmCursor = len([]rune(entry.Name))
		}
	case 'd':
		if m.FocusedPanel == PanelBookmarks {
			m.bookmarkDelete()
		} else {
			m.fmStart()
			op := FmOpDelete
			m.FmOperation = &op
		}
	case 'c':
		m.fmStart()
		op := FmOpCopy
		m.FmOperation = &op
		m.FmInput = m.CurrentDir
		m.FmCursor = len([]rune(m.FmInput))
	case 'm':
		m.fmStart()
		op := FmOpMove
		m.FmOperation = &op
		m.FmInput = m.CurrentDir
		m.FmCursor = len([]rune(m.FmInput))
	case 'W', 'w':
		m.PreviewWrap = !m.PreviewWrap
		m.loadPreviewForSelected()
	case 'L':
		m.PreviewLineNumbers = !m.PreviewLineNumbers
	case '?':
		m.Mode = ModeHelp
	case 'q', 'Q':
		m.ShouldQuit = true
	case '.':
		m.Config.General.ShowHidden = !m.Config.General.ShowHidden
		m.updateFilter()
		m.loadPreviewForSelected()
	}
	return nil
}

// handleKeyViewer handles key events in fullscreen viewer mode.
func (m *Model) handleKeyViewer(key tea.KeyMsg) tea.Cmd {
	if m.ViewerIsGoto {
		return m.handleKeyViewerGoto(key)
	}
	if m.ViewerIsSearching {
		return m.handleKeyViewerSearch(key)
	}

	pageSize := m.ViewerHeight
	if pageSize < 1 {
		pageSize = 20
	}

	switch key.Type {
	case tea.KeyUp:
		if m.PreviewScroll > 0 {
			m.PreviewScroll--
		}
	case tea.KeyDown:
		if m.PreviewScroll < len(m.ViewerLines)-1 {
			m.PreviewScroll++
		}
	case tea.KeyLeft:
		if m.PreviewHScroll > 0 {
			m.PreviewHScroll -= 4
			if m.PreviewHScroll < 0 {
				m.PreviewHScroll = 0
			}
		}
	case tea.KeyRight:
		m.PreviewHScroll += 4
	case tea.KeyPgUp:
		m.PreviewScroll -= pageSize
		if m.PreviewScroll < 0 {
			m.PreviewScroll = 0
		}
	case tea.KeyPgDown, tea.KeySpace:
		m.PreviewScroll += pageSize
		max := len(m.ViewerLines) - 1
		if m.PreviewScroll > max {
			m.PreviewScroll = max
		}
	case tea.KeyEsc:
		m.Mode = ModeFileList
		m.clearViewerSearch()
	case tea.KeyRunes:
		return m.handleKeyViewerRunes(key)
	}
	return nil
}

func (m *Model) handleKeyViewerSearch(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.ViewerIsSearching = false
		m.ViewerSearchQuery = ""
	case tea.KeyEnter:
		m.ViewerIsSearching = false
		m.viewerDoSearch(m.ViewerSearchQuery)
	case tea.KeyBackspace:
		if len(m.ViewerSearchQuery) > 0 {
			runes := []rune(m.ViewerSearchQuery)
			m.ViewerSearchQuery = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		m.ViewerSearchQuery += string(key.Runes)
	}
	return nil
}

func (m *Model) handleKeyViewerGoto(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.ViewerIsGoto = false
		m.ViewerGotoInput = ""
	case tea.KeyEnter:
		m.ViewerIsGoto = false
		if n, err := strconv.Atoi(m.ViewerGotoInput); err == nil {
			m.PreviewScroll = n - 1
			if m.PreviewScroll < 0 {
				m.PreviewScroll = 0
			}
			if m.PreviewScroll >= len(m.ViewerLines) {
				m.PreviewScroll = max(0, len(m.ViewerLines)-1)
			}
		}
		m.ViewerGotoInput = ""
	case tea.KeyBackspace:
		if len(m.ViewerGotoInput) > 0 {
			m.ViewerGotoInput = m.ViewerGotoInput[:len(m.ViewerGotoInput)-1]
		}
	case tea.KeyRunes:
		m.ViewerGotoInput += string(key.Runes)
	}
	return nil
}

func (m *Model) handleKeyViewerRunes(key tea.KeyMsg) tea.Cmd {
	if len(key.Runes) == 0 {
		return nil
	}
	ch := key.Runes[0]
	pageSize := m.ViewerHeight
	if pageSize < 1 {
		pageSize = 20
	}

	switch ch {
	case 'q', 'Q':
		m.Mode = ModeFileList
		m.clearViewerSearch()
	case 'j':
		if m.PreviewScroll < len(m.ViewerLines)-1 {
			m.PreviewScroll++
		}
	case 'k':
		if m.PreviewScroll > 0 {
			m.PreviewScroll--
		}
	case 'h':
		if m.PreviewHScroll > 0 {
			m.PreviewHScroll -= 4
			if m.PreviewHScroll < 0 {
				m.PreviewHScroll = 0
			}
		}
	case 'l':
		m.PreviewHScroll += 4
	case 'g':
		if m.ViewerPrevKeyG {
			m.PreviewScroll = 0
			m.ViewerPrevKeyG = false
		} else {
			m.ViewerPrevKeyG = true
		}
		return nil
	case 'G':
		m.PreviewScroll = max(0, len(m.ViewerLines)-1)
	case 'f':
		m.PreviewScroll += pageSize
		if m.PreviewScroll >= len(m.ViewerLines) {
			m.PreviewScroll = max(0, len(m.ViewerLines)-1)
		}
	case 'b':
		m.PreviewScroll -= pageSize
		if m.PreviewScroll < 0 {
			m.PreviewScroll = 0
		}
	case 'd':
		m.PreviewScroll += pageSize / 2
		if m.PreviewScroll >= len(m.ViewerLines) {
			m.PreviewScroll = max(0, len(m.ViewerLines)-1)
		}
	case 'u':
		m.PreviewScroll -= pageSize / 2
		if m.PreviewScroll < 0 {
			m.PreviewScroll = 0
		}
	case '/':
		m.ViewerIsSearching = true
		m.ViewerSearchQuery = ""
	case 'n':
		m.viewerNextMatch()
	case 'N':
		m.viewerPrevMatch()
	case ':':
		m.ViewerIsGoto = true
		m.ViewerGotoInput = ""
	case 'W', 'w':
		m.PreviewWrap = !m.PreviewWrap
		m.reloadViewerLines()
	case 'L':
		m.PreviewLineNumbers = !m.PreviewLineNumbers
	case 'y':
		if len(m.ViewerLines) > 0 && m.PreviewScroll < len(m.ViewerLines) {
			m.writeToClipboard(m.ViewerLines[m.PreviewScroll])
		}
	}
	m.ViewerPrevKeyG = false
	return nil
}

// handleKeyBookmarks handles key events in bookmarks panel mode.
func (m *Model) handleKeyBookmarks(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyUp:
		if m.BookmarkIndex > 0 {
			m.BookmarkIndex--
		}
	case tea.KeyDown:
		if m.BookmarkIndex < len(m.Config.Bookmarks)-1 {
			m.BookmarkIndex++
		}
	case tea.KeyEnter:
		cmd := m.navigateToBookmark()
		m.FocusedPanel = PanelFileList
		return cmd
	case tea.KeyEsc, tea.KeyTab:
		m.FocusedPanel = PanelFileList
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			switch key.Runes[0] {
			case 'd':
				m.bookmarkDelete()
			case 'q':
				m.FocusedPanel = PanelFileList
			}
		}
	}
	return nil
}

// handleKeyClipboardPanel handles key events in path clipboard panel.
func (m *Model) handleKeyClipboardPanel(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyUp:
		if m.PathClipboardIdx > 0 {
			m.PathClipboardIdx--
		}
	case tea.KeyDown:
		if m.PathClipboardIdx < len(m.PathClipboard)-1 {
			m.PathClipboardIdx++
		}
	case tea.KeyEnter:
		if m.PathClipboardIdx < len(m.PathClipboard) {
			cmd := m.navigateTo(m.PathClipboard[m.PathClipboardIdx])
			m.loadPreviewForSelected()
			m.Mode = ModeFileList
			return cmd
		}
		m.Mode = ModeFileList
	case tea.KeyEsc:
		m.Mode = ModeFileList
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			ch := key.Runes[0]
			switch ch {
			case 'j':
				if m.PathClipboardIdx < len(m.PathClipboard)-1 {
					m.PathClipboardIdx++
				}
			case 'k':
				if m.PathClipboardIdx > 0 {
					m.PathClipboardIdx--
				}
			case 'y':
				if m.PathClipboardIdx < len(m.PathClipboard) {
					m.writeToClipboard(m.PathClipboard[m.PathClipboardIdx])
				}
			case 'd':
				if m.PathClipboardIdx < len(m.PathClipboard) {
					m.PathClipboard = append(m.PathClipboard[:m.PathClipboardIdx], m.PathClipboard[m.PathClipboardIdx+1:]...)
					if m.PathClipboardIdx >= len(m.PathClipboard) && m.PathClipboardIdx > 0 {
						m.PathClipboardIdx--
					}
				}
			case 'q':
				m.Mode = ModeFileList
			}
		}
	}
	return nil
}

// handleKeyOpenChoice handles key events in the open-choice modal.
func (m *Model) handleKeyOpenChoice(key tea.KeyMsg) tea.Cmd {
	choiceCount := 2

	switch key.Type {
	case tea.KeyUp:
		if m.OpenChoiceIndex > 0 {
			m.OpenChoiceIndex--
		}
	case tea.KeyDown:
		if m.OpenChoiceIndex < choiceCount-1 {
			m.OpenChoiceIndex++
		}
	case tea.KeyEnter:
		return m.executeOpenChoice()
	case tea.KeyEsc:
		m.Mode = ModeFileList
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			switch key.Runes[0] {
			case 'j':
				if m.OpenChoiceIndex < choiceCount-1 {
					m.OpenChoiceIndex++
				}
			case 'k':
				if m.OpenChoiceIndex > 0 {
					m.OpenChoiceIndex--
				}
			case 'q':
				m.Mode = ModeFileList
			}
		}
	}
	return nil
}

// executeOpenChoice performs the selected action from the open-choice modal.
func (m *Model) executeOpenChoice() tea.Cmd {
	entry := m.SelectedEntry()
	if entry == nil {
		m.Mode = ModeFileList
		return nil
	}
	path := entry.Path
	m.Mode = ModeFileList

	switch m.OpenChoiceIndex {
	case 0:
		return m.openWithDefaultApp(path)
	case 1:
		return m.openInVSCode(path)
	}
	return nil
}

// handleKeyOpenWith handles key events in open-with mode.
func (m *Model) handleKeyOpenWith(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyUp:
		if m.OpenWithIndex > 0 {
			m.OpenWithIndex--
		}
	case tea.KeyDown:
		if m.OpenWithIndex < len(m.Config.Openers)-1 {
			m.OpenWithIndex++
		}
	case tea.KeyEnter:
		return m.executeOpenWith()
	case tea.KeyEsc:
		m.Mode = ModeFileList
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			switch key.Runes[0] {
			case 'j':
				if m.OpenWithIndex < len(m.Config.Openers)-1 {
					m.OpenWithIndex++
				}
			case 'k':
				if m.OpenWithIndex > 0 {
					m.OpenWithIndex--
				}
			case 'q':
				m.Mode = ModeFileList
			}
		}
	}
	return nil
}

// handleKeySettings handles key events in settings mode.
func (m *Model) handleKeySettings(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.Mode = ModeFileList
	case tea.KeyRunes:
		if len(key.Runes) > 0 && key.Runes[0] == 'q' {
			m.Mode = ModeFileList
		}
	}
	return nil
}

// handleKeyPalette handles key events in command palette mode.
func (m *Model) handleKeyPalette(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.Mode = ModeFileList
	case tea.KeyEnter:
		m.Mode = ModeFileList
	case tea.KeyRunes:
		if len(key.Runes) > 0 && key.Runes[0] == 'q' {
			m.Mode = ModeFileList
		}
	}
	return nil
}

// handleKeyHelp handles key events in help mode.
func (m *Model) handleKeyHelp(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.Mode = ModeFileList
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			ch := key.Runes[0]
			if ch == 'q' || ch == '?' {
				m.Mode = ModeFileList
			}
		}
	}
	return nil
}

// handleKeyGit dispatches to the appropriate git key handler.
func (m *Model) handleKeyGit(key tea.KeyMsg) tea.Cmd {
	if m.Git == nil {
		m.Mode = ModeFileList
		return nil
	}
	g := m.Git

	// Async progress - only allow cancel
	if g.AsyncCmd != nil {
		return m.handleKeyGitAsyncProgress(key)
	}

	// Confirm modal
	if g.Confirm != nil {
		return m.handleKeyGitConfirm(key)
	}

	// Commit input
	if g.IsCommitting {
		return m.handleKeyGitCommitInput(key)
	}

	// Branch input
	if g.BranchInputActive {
		return m.handleKeyGitBranchInput(key)
	}

	// Branch panel
	if g.BranchPanelOpen {
		return m.handleKeyGitBranchPanel(key)
	}

	// Diff fullscreen
	if g.DiffFullscreen {
		return m.handleKeyGitDiffFullscreen(key)
	}

	// Log mode
	if g.ShowLog {
		return m.handleKeyGitLog(key)
	}

	// Normal git file panels
	return m.handleKeyGitFiles(key)
}

func (m *Model) handleKeyGitAsyncProgress(key tea.KeyMsg) tea.Cmd {
	// ESC to cancel not easily supported; just allow 'q'
	if key.Type == tea.KeyRunes && len(key.Runes) > 0 && key.Runes[0] == 'q' {
		if m.Git.AsyncCmd != nil && m.Git.AsyncCmd.Process != nil {
			_ = m.Git.AsyncCmd.Process.Kill()
		}
		m.finishAsync()
	}
	return nil
}

func (m *Model) handleKeyGitConfirm(key tea.KeyMsg) tea.Cmd {
	confirm := m.Git.Confirm
	m.Git.Confirm = nil

	switch key.Type {
	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return nil
		}
		ch := key.Runes[0]
		if ch == 'y' || ch == 'Y' {
			m.executeGitConfirm(confirm)
		}
	}
	return nil
}

func (m *Model) executeGitConfirm(confirm *state.ConfirmData) {
	if m.Git == nil || m.Git.Status == nil {
		return
	}
	root := m.Git.Status.Root
	switch confirm.Kind {
	case state.ConfirmDeleteBranchSoft:
		if err := git.DeleteBranch(root, confirm.Name, false); err != nil {
			m.SetStatusError("브랜치 삭제 실패: " + err.Error())
		} else {
			m.SetStatusSuccess("브랜치 삭제됨: " + confirm.Name)
			m.Git.Branches = git.ListBranches(root)
		}
	case state.ConfirmDeleteBranchForce:
		if err := git.DeleteBranch(root, confirm.Name, true); err != nil {
			m.SetStatusError("브랜치 강제 삭제 실패: " + err.Error())
		} else {
			m.SetStatusSuccess("브랜치 강제 삭제됨: " + confirm.Name)
			m.Git.Branches = git.ListBranches(root)
		}
	case state.ConfirmCheckoutFile:
		if err := git.RestoreFile(root, confirm.Name); err != nil {
			m.SetStatusError("파일 되돌리기 실패: " + err.Error())
		} else {
			m.SetStatusSuccess("파일 되돌림: " + confirm.Name)
			m.gitRefresh()
		}
	case state.ConfirmForcePush:
		m.startGitAsync(state.AsyncData{
			Kind:   state.AsyncPush,
			Force:  true,
			Branch: m.Git.Status.Branch,
		})
	}
}

func (m *Model) handleKeyGitCommitInput(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.Git.IsCommitting = false
		m.Git.CommitInput = ""
	case tea.KeyEnter:
		if m.Git.CommitInput == "" {
			m.SetStatusError("커밋 메시지를 입력하세요")
			return nil
		}
		if m.Git.Status == nil {
			return nil
		}
		if err := git.CommitChanges(m.Git.Status.Root, m.Git.CommitInput); err != nil {
			m.SetStatusError("커밋 실패: " + err.Error())
		} else {
			m.SetStatusSuccess("커밋 완료")
			m.gitRefresh()
		}
		m.Git.IsCommitting = false
		m.Git.CommitInput = ""
	case tea.KeyBackspace:
		if len(m.Git.CommitInput) > 0 {
			runes := []rune(m.Git.CommitInput)
			m.Git.CommitInput = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		m.Git.CommitInput += string(key.Runes)
	}
	return nil
}

func (m *Model) handleKeyGitBranchInput(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.Git.BranchInputActive = false
		m.Git.BranchInput = ""
	case tea.KeyEnter:
		if m.Git.BranchInput == "" {
			m.Git.BranchInputActive = false
			return nil
		}
		if m.Git.Status == nil {
			return nil
		}
		if err := git.CreateBranch(m.Git.Status.Root, m.Git.BranchInput); err != nil {
			m.SetStatusError("브랜치 생성 실패: " + err.Error())
		} else {
			m.SetStatusSuccess("브랜치 생성 및 전환됨: " + m.Git.BranchInput)
			m.Git.Branches = git.ListBranches(m.Git.Status.Root)
			m.gitRefresh()
		}
		m.Git.BranchInputActive = false
		m.Git.BranchInput = ""
	case tea.KeyBackspace:
		if len(m.Git.BranchInput) > 0 {
			runes := []rune(m.Git.BranchInput)
			m.Git.BranchInput = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		m.Git.BranchInput += string(key.Runes)
	}
	return nil
}

func (m *Model) handleKeyGitBranchPanel(key tea.KeyMsg) tea.Cmd {
	g := m.Git
	switch key.Type {
	case tea.KeyUp:
		if g.BranchIdx > 0 {
			g.BranchIdx--
		}
	case tea.KeyDown:
		if g.BranchIdx < len(g.Branches)-1 {
			g.BranchIdx++
		}
	case tea.KeyEnter:
		if g.BranchIdx < len(g.Branches) && g.Status != nil {
			branch := g.Branches[g.BranchIdx]
			if err := git.SwitchBranch(g.Status.Root, branch.Name); err != nil {
				m.SetStatusError("브랜치 전환 실패: " + err.Error())
			} else {
				m.SetStatusSuccess("브랜치 전환됨: " + branch.Name)
				m.gitRefresh()
			}
			g.BranchPanelOpen = false
		}
	case tea.KeyEsc:
		g.BranchPanelOpen = false
	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return nil
		}
		ch := key.Runes[0]
		switch ch {
		case 'j':
			if g.BranchIdx < len(g.Branches)-1 {
				g.BranchIdx++
			}
		case 'k':
			if g.BranchIdx > 0 {
				g.BranchIdx--
			}
		case 'n':
			g.BranchInputActive = true
			g.BranchInput = ""
		case 'd':
			if g.BranchIdx < len(g.Branches) {
				branch := g.Branches[g.BranchIdx]
				g.Confirm = &state.ConfirmData{
					Kind: state.ConfirmDeleteBranchSoft,
					Name: branch.Name,
				}
			}
		case 'D':
			if g.BranchIdx < len(g.Branches) {
				branch := g.Branches[g.BranchIdx]
				g.Confirm = &state.ConfirmData{
					Kind: state.ConfirmDeleteBranchForce,
					Name: branch.Name,
				}
			}
		case 'q':
			g.BranchPanelOpen = false
		}
	}
	return nil
}

func (m *Model) handleKeyGitDiffFullscreen(key tea.KeyMsg) tea.Cmd {
	g := m.Git
	pageSize := m.Height - 2
	if pageSize < 1 {
		pageSize = 20
	}

	switch key.Type {
	case tea.KeyUp:
		if g.DiffScroll > 0 {
			g.DiffScroll--
		}
	case tea.KeyDown:
		g.DiffScroll++
	case tea.KeyPgUp:
		g.DiffScroll -= pageSize
		if g.DiffScroll < 0 {
			g.DiffScroll = 0
		}
	case tea.KeyPgDown:
		g.DiffScroll += pageSize
	case tea.KeyEsc:
		g.DiffFullscreen = false
	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return nil
		}
		ch := key.Runes[0]
		switch ch {
		case 'j':
			g.DiffScroll++
		case 'k':
			if g.DiffScroll > 0 {
				g.DiffScroll--
			}
		case 'q':
			g.DiffFullscreen = false
		case 'W', 'w':
			g.DiffWrap = !g.DiffWrap
		}
	}
	return nil
}

func (m *Model) handleKeyGitLog(key tea.KeyMsg) tea.Cmd {
	g := m.Git
	switch key.Type {
	case tea.KeyUp:
		if g.LogFocused {
			if g.LogIdx > 0 {
				g.LogIdx--
				g.LoadCommitShow()
			}
		} else if g.LogFileFocused {
			if g.CommitFileIdx > 0 {
				g.CommitFileIdx--
				g.LoadCommitFileDiff()
			}
		} else {
			if g.CommitShowScroll > 0 {
				g.CommitShowScroll--
			}
		}
	case tea.KeyDown:
		if g.LogFocused {
			if g.LogIdx < len(g.Log)-1 {
				g.LogIdx++
				g.LoadCommitShow()
			}
		} else if g.LogFileFocused {
			if g.CommitFileIdx < len(g.CommitFiles)-1 {
				g.CommitFileIdx++
				g.LoadCommitFileDiff()
			}
		} else {
			g.CommitShowScroll++
		}
	case tea.KeyEsc:
		g.ShowLog = false
		g.LogFocused = false
		g.LogFileFocused = false
	case tea.KeyTab:
		if g.LogFocused {
			g.LogFocused = false
			g.LogFileFocused = true
		} else if g.LogFileFocused {
			g.LogFileFocused = false
		} else {
			g.LogFocused = true
		}
	case tea.KeyRunes:
		m.handleKeyGitLogRunes(key)
	}
	return nil
}

func (m *Model) handleKeyGitLogRunes(key tea.KeyMsg) {
	if len(key.Runes) == 0 {
		return
	}
	g := m.Git
	ch := key.Runes[0]
	switch ch {
	case 'j':
		if g.LogFocused {
			if g.LogIdx < len(g.Log)-1 {
				g.LogIdx++
				g.LoadCommitShow()
			}
		} else if g.LogFileFocused {
			if g.CommitFileIdx < len(g.CommitFiles)-1 {
				g.CommitFileIdx++
				g.LoadCommitFileDiff()
			}
		} else {
			g.CommitShowScroll++
		}
	case 'k':
		if g.LogFocused {
			if g.LogIdx > 0 {
				g.LogIdx--
				g.LoadCommitShow()
			}
		} else if g.LogFileFocused {
			if g.CommitFileIdx > 0 {
				g.CommitFileIdx--
				g.LoadCommitFileDiff()
			}
		} else {
			if g.CommitShowScroll > 0 {
				g.CommitShowScroll--
			}
		}
	case 'l':
		if g.LogFocused {
			g.LogFocused = false
			g.LogFileFocused = true
		} else if g.LogFileFocused {
			g.LogFileFocused = false
		}
	case 'h':
		if g.LogFileFocused {
			g.LogFileFocused = false
			g.LogFocused = true
		} else if !g.LogFocused {
			g.LogFocused = true
		}
	case 'q':
		g.ShowLog = false
		g.LogFocused = false
		g.LogFileFocused = false
	}
}

func (m *Model) handleKeyGitFiles(key tea.KeyMsg) tea.Cmd {
	g := m.Git
	// Always allow exiting, even when not in a git repository
	if key.Type == tea.KeyEsc {
		m.Mode = ModeFileList
		return nil
	}
	if key.Type == tea.KeyRunes && len(key.Runes) > 0 {
		if key.Runes[0] == 'q' || key.Runes[0] == 'Q' {
			m.Mode = ModeFileList
			return nil
		}
	}
	if g.Status == nil {
		return nil
	}
	root := g.Status.Root

	switch key.Type {
	case tea.KeyUp:
		m.gitMoveUp()
	case tea.KeyDown:
		m.gitMoveDown()
	case tea.KeyTab:
		if g.Section == state.GitSectionStaged {
			g.Section = state.GitSectionUnstaged
		} else {
			g.Section = state.GitSectionStaged
		}
		g.LoadDiff()
	case tea.KeyEnter:
		g.DiffFullscreen = true
	case tea.KeyEsc:
		m.Mode = ModeFileList
	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return nil
		}
		ch := key.Runes[0]
		switch ch {
		case 'j':
			m.gitMoveDown()
		case 'k':
			m.gitMoveUp()
		case 's':
			// Stage selected file (unstaged section) or stage all unstaged (staged section).
			if g.Section == state.GitSectionUnstaged && g.UnstagedIdx < len(g.Status.Unstaged) {
				f := g.Status.Unstaged[g.UnstagedIdx]
				if err := git.StageFile(root, f.Path); err != nil {
					m.SetStatusError("스테이지 실패: " + err.Error())
				} else {
					m.gitRefresh()
				}
			} else if g.Section == state.GitSectionStaged {
				// In staged section, 's' stages all remaining unstaged files.
				if err := git.StageAll(root); err != nil {
					m.SetStatusError("전체 스테이지 실패: " + err.Error())
				} else {
					m.SetStatusSuccess("전체 스테이지 완료")
					m.gitRefresh()
				}
			}
		case 'u':
			// Unstage selected staged file.
			if g.Section == state.GitSectionStaged && g.StagedIdx < len(g.Status.Staged) {
				f := g.Status.Staged[g.StagedIdx]
				if err := git.UnstageFile(root, f.Path); err != nil {
					m.SetStatusError("언스테이지 실패: " + err.Error())
				} else {
					m.gitRefresh()
				}
			}
		case 'a':
			// Stage all unstaged files.
			if err := git.StageAll(root); err != nil {
				m.SetStatusError("전체 스테이지 실패: " + err.Error())
			} else {
				m.SetStatusSuccess("전체 스테이지 완료")
				m.gitRefresh()
			}
		case 'A':
			// Unstage all staged files.
			if err := git.UnstageAll(root); err != nil {
				m.SetStatusError("전체 언스테이지 실패: " + err.Error())
			} else {
				m.SetStatusSuccess("전체 언스테이지 완료")
				m.gitRefresh()
			}
		case 'c':
			if len(g.Status.Staged) == 0 {
				m.SetStatusError("스테이지된 변경이 없습니다")
			} else {
				g.IsCommitting = true
				g.CommitInput = ""
			}
		case 'p':
			// Regular push.
			m.startGitAsync(state.AsyncData{
				Kind:   state.AsyncPush,
				Force:  false,
				Branch: g.Status.Branch,
			})
		case 'F':
			// Force push — requires confirmation.
			g.Confirm = &state.ConfirmData{
				Kind: state.ConfirmForcePush,
				Name: g.Status.Branch,
			}
		case 'P':
			m.startGitAsync(state.AsyncData{Kind: state.AsyncPull})
		case 'f':
			m.startGitAsync(state.AsyncData{Kind: state.AsyncFetch})
		case 'L':
			g.ShowLog = true
			g.LogFocused = true
			if g.Status != nil {
				g.Log = git.GetLog(g.Status.Root)
			}
			if len(g.Log) > 0 {
				g.LoadCommitShow()
			}
		case 'b':
			g.BranchPanelOpen = true
			g.BranchIdx = 0
			if g.Status != nil {
				g.Branches = git.ListBranches(g.Status.Root)
			}
		case 'R':
			m.gitRefresh()
		case 'r':
			// Restore (checkout) file
			var currentFile *git.GitFile
			if g.Section == state.GitSectionUnstaged && g.UnstagedIdx < len(g.Status.Unstaged) {
				f := g.Status.Unstaged[g.UnstagedIdx]
				currentFile = &f
			}
			if currentFile != nil {
				g.Confirm = &state.ConfirmData{
					Kind: state.ConfirmCheckoutFile,
					Name: currentFile.Path,
				}
			}
		case 'W', 'w':
			g.DiffWrap = !g.DiffWrap
		case 'q', 'Q':
			m.Mode = ModeFileList
		}
	}
	return nil
}

func (m *Model) gitMoveUp() {
	g := m.Git
	if g.Status == nil {
		return
	}
	switch g.Section {
	case state.GitSectionStaged:
		if g.StagedIdx > 0 {
			g.StagedIdx--
			g.LoadDiff()
		}
	case state.GitSectionUnstaged:
		if g.UnstagedIdx > 0 {
			g.UnstagedIdx--
			g.LoadDiff()
		}
	}
}

func (m *Model) gitMoveDown() {
	g := m.Git
	if g.Status == nil {
		return
	}
	switch g.Section {
	case state.GitSectionStaged:
		if g.StagedIdx < len(g.Status.Staged)-1 {
			g.StagedIdx++
			g.LoadDiff()
		}
	case state.GitSectionUnstaged:
		if g.UnstagedIdx < len(g.Status.Unstaged)-1 {
			g.UnstagedIdx++
			g.LoadDiff()
		}
	}
}

// handleKeyFileManager handles key events in file manager mode.
func (m *Model) handleKeyFileManager(key tea.KeyMsg) tea.Cmd {
	// Overwrite confirm
	if m.FmOverwriteTarget != "" {
		return m.handleKeyFmOverwriteConfirm(key)
	}
	// Error state
	if m.FmError != "" {
		switch key.Type {
		case tea.KeyEsc, tea.KeyEnter:
			m.FmError = ""
			m.Mode = ModeFileList
		}
		return nil
	}
	// Menu
	if m.FmOperation == nil {
		return m.handleKeyFmMenu(key)
	}
	// Delete confirm
	if *m.FmOperation == FmOpDelete {
		return m.handleKeyFmDeleteConfirm(key)
	}
	// Input
	return m.handleKeyFmInput(key)
}

func (m *Model) handleKeyFmOverwriteConfirm(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			ch := key.Runes[0]
			if ch == 'y' || ch == 'Y' {
				m.executeFmOperationForce()
				return nil
			}
		}
		m.FmOverwriteTarget = ""
		m.Mode = ModeFileList
	case tea.KeyEsc:
		m.FmOverwriteTarget = ""
		m.Mode = ModeFileList
	}
	return nil
}

func (m *Model) handleKeyFmMenu(key tea.KeyMsg) tea.Cmd {
	menuItems := []FmOp{FmOpCopy, FmOpMove, FmOpRename, FmOpDelete, FmOpNewDir}
	switch key.Type {
	case tea.KeyUp:
		if m.FmMenuIdx > 0 {
			m.FmMenuIdx--
		}
	case tea.KeyDown:
		if m.FmMenuIdx < len(menuItems)-1 {
			m.FmMenuIdx++
		}
	case tea.KeyEnter:
		op := menuItems[m.FmMenuIdx]
		m.FmOperation = &op
		switch op {
		case FmOpCopy, FmOpMove:
			m.FmInput = m.CurrentDir
			m.FmCursor = len([]rune(m.FmInput))
		case FmOpRename:
			if entry := m.SelectedEntry(); entry != nil {
				m.FmInput = entry.Name
				m.FmCursor = len([]rune(entry.Name))
			}
		case FmOpDelete:
			m.FmInput = ""
		case FmOpNewDir:
			m.FmInput = ""
			m.FmCursor = 0
		}
	case tea.KeyEsc:
		m.Mode = ModeFileList
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			switch key.Runes[0] {
			case 'j':
				if m.FmMenuIdx < len(menuItems)-1 {
					m.FmMenuIdx++
				}
			case 'k':
				if m.FmMenuIdx > 0 {
					m.FmMenuIdx--
				}
			case 'q':
				m.Mode = ModeFileList
			}
		}
	}
	return nil
}

func (m *Model) handleKeyFmDeleteConfirm(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			ch := key.Runes[0]
			if ch == 'y' || ch == 'Y' {
				m.executeFmOperation()
				return nil
			}
		}
		m.Mode = ModeFileList
		m.FmOperation = nil
	case tea.KeyEsc, tea.KeyEnter:
		m.Mode = ModeFileList
		m.FmOperation = nil
	}
	return nil
}

func (m *Model) handleKeyFmInput(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEnter:
		m.executeFmOperation()
	case tea.KeyEsc:
		m.Mode = ModeFileList
		m.FmOperation = nil
		m.FmInput = ""
		m.FmCursor = 0
	case tea.KeyBackspace:
		if m.FmCursor > 0 {
			runes := []rune(m.FmInput)
			runes = append(runes[:m.FmCursor-1], runes[m.FmCursor:]...)
			m.FmInput = string(runes)
			m.FmCursor--
		}
	case tea.KeyLeft:
		if m.FmCursor > 0 {
			m.FmCursor--
		}
	case tea.KeyRight:
		if m.FmCursor < len([]rune(m.FmInput)) {
			m.FmCursor++
		}
	case tea.KeyHome:
		m.FmCursor = 0
	case tea.KeyEnd:
		m.FmCursor = len([]rune(m.FmInput))
	case tea.KeyRunes:
		runes := []rune(m.FmInput)
		newRunes := make([]rune, len(runes)+len(key.Runes))
		copy(newRunes, runes[:m.FmCursor])
		copy(newRunes[m.FmCursor:], key.Runes)
		copy(newRunes[m.FmCursor+len(key.Runes):], runes[m.FmCursor:])
		m.FmInput = string(newRunes)
		m.FmCursor += len(key.Runes)
	}
	return nil
}

func (m *Model) executeFmOperation() {
	if m.FmOperation == nil {
		return
	}
	op := *m.FmOperation
	src := m.SelectedPath()

	switch op {
	case FmOpCopy:
		dst := m.FmInput
		if err := copyWithOverwriteCheck(src, dst, m); err != nil {
			m.FmError = err.Error()
		} else {
			m.SetStatusSuccess("복사 완료")
			m.fmRefreshFileList()
			m.Mode = ModeFileList
			m.FmOperation = nil
		}
	case FmOpMove:
		dst := m.FmInput
		if err := moveWithOverwriteCheck(src, dst, m); err != nil {
			m.FmError = err.Error()
		} else {
			m.SetStatusSuccess("이동 완료")
			m.fmRefreshFileList()
			m.Mode = ModeFileList
			m.FmOperation = nil
		}
	case FmOpRename:
		if _, err := renameFile(src, m.FmInput); err != nil {
			m.FmError = err.Error()
		} else {
			m.SetStatusSuccess("이름 변경 완료")
			m.fmRefreshFileList()
			m.Mode = ModeFileList
			m.FmOperation = nil
		}
	case FmOpDelete:
		if err := deleteFile(src); err != nil {
			m.FmError = err.Error()
		} else {
			m.SetStatusSuccess("삭제 완료")
			m.fmRefreshFileList()
			m.Mode = ModeFileList
			m.FmOperation = nil
		}
	case FmOpNewDir:
		if _, err := createDir(m.CurrentDir, m.FmInput); err != nil {
			m.FmError = err.Error()
		} else {
			m.SetStatusSuccess("폴더 생성 완료: " + m.FmInput)
			m.fmRefreshFileList()
			m.Mode = ModeFileList
			m.FmOperation = nil
		}
	}
}

func (m *Model) executeFmOperationForce() {
	if m.FmOperation == nil {
		return
	}
	op := *m.FmOperation
	src := m.SelectedPath()
	dst := m.FmOverwriteTarget
	m.FmOverwriteTarget = ""

	switch op {
	case FmOpCopy:
		if err := fsCopy(src, dst); err != nil {
			m.FmError = err.Error()
		} else {
			m.SetStatusSuccess("복사 완료 (덮어씀)")
			m.fmRefreshFileList()
			m.Mode = ModeFileList
			m.FmOperation = nil
		}
	case FmOpMove:
		if err := fsMove(src, dst); err != nil {
			m.FmError = err.Error()
		} else {
			m.SetStatusSuccess("이동 완료 (덮어씀)")
			m.fmRefreshFileList()
			m.Mode = ModeFileList
			m.FmOperation = nil
		}
	}
}

// File operation helpers that use fs package.
func copyWithOverwriteCheck(src, dst string, m *Model) error {
	if fileExists(dst) {
		m.FmOverwriteTarget = dst
		return nil // handled by overwrite confirm
	}
	return fsCopy(src, dst)
}

func moveWithOverwriteCheck(src, dst string, m *Model) error {
	if fileExists(dst) {
		m.FmOverwriteTarget = dst
		return nil
	}
	return fsMove(src, dst)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fsCopy(src, dst string) error {
	return fs.CopyFile(src, dst)
}

func fsMove(src, dst string) error {
	return fs.MoveFile(src, dst)
}

func renameFile(src, newName string) (string, error) {
	return fs.RenameFile(src, newName)
}

func deleteFile(path string) error {
	return fs.DeleteFile(path)
}

func createDir(parent, name string) (string, error) {
	return fs.CreateDir(parent, name)
}
