package app

import (
	"path/filepath"
	"strings"

	"github.com/pickmoment/drf/ui"

	"github.com/charmbracelet/lipgloss"
)

// View renders the full TUI.
func (m *Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "로딩 중..."
	}
	if m.ShouldQuit {
		return ""
	}

	switch m.Mode {
	case ModeViewer:
		return m.renderViewerMode()
	case ModeGit:
		return m.renderGitMode()
	default:
		return m.renderMainMode()
	}
}

func (m *Model) renderViewerMode() string {
	p := &ui.ViewerParams{
		FilePath:        m.ViewerPath,
		Lines:           m.ViewerLines,
		Scroll:          m.PreviewScroll,
		HScroll:         m.PreviewHScroll,
		Wrap:            m.PreviewWrap,
		LineNumbers:     m.PreviewLineNumbers,
		Width:           m.Width,
		Height:          m.ViewerContentHeight(),
		IsSearching:     m.ViewerIsSearching,
		SearchQuery:     m.ViewerSearchQuery,
		SearchMatches:   m.ViewerSearchMatches,
		SearchIdx:       m.ViewerSearchIdx,
		IsGoto:          m.ViewerIsGoto,
		GotoInput:       m.ViewerGotoInput,
		ViewerHeightOut: &m.ViewerHeight,
	}
	return ui.RenderFullscreen(p)
}

func (m *Model) ViewerContentHeight() int {
	// RenderFullscreen subtracts toolbar(1) + status(1) + optional search(1)
	// from this value, so we pass the full terminal height.
	if m.Height < 1 {
		return 1
	}
	return m.Height
}

func (m *Model) renderGitMode() string {
	p := &ui.GitParams{
		Width:    m.Width,
		Height:   m.Height,
		GitState: m.Git,
	}
	if m.Status != nil && !m.Status.IsExpired() {
		p.HasStatus = true
		p.StatusText = m.Status.Text
		p.StatusKind = int(m.Status.Kind)
	}
	return ui.RenderGit(p)
}

func (m *Model) renderMainMode() string {
	// Layout:
	//   [tab bar 1 line]
	//   [main panels Height-2 or Height-3 if status]
	//   [status bar 0 or 1 line]
	//   [hint bar 1 line]
	tabH := 1
	hintH := 1
	if !m.Config.Ui.ShowHintBar {
		hintH = 0
	}
	statusH := 0
	if m.Status != nil {
		statusH = 1
	}
	panelH := m.Height - tabH - hintH - statusH
	if panelH < 3 {
		panelH = 3
	}

	gitBranch := ""
	gitDirty := false
	if m.Git != nil && m.Git.Status != nil {
		gitBranch = m.Git.Status.Branch
		gitDirty = len(m.Git.Status.Staged) > 0 || len(m.Git.Status.Unstaged) > 0
	}
	tabBar := ui.RenderTabBar(m.CurrentDir, gitBranch, gitDirty, len(m.FilteredIndices), m.Width)

	// Determine preview lines for the panel
	var previewLines []string
	var previewFileName string
	if m.Config.Ui.ShowPreviewPanel {
		if len(m.ViewerLines) > 0 {
			previewLines = m.ViewerLines
			previewFileName = filepath.Base(m.ViewerPath)
		}
	}

	panelParams := &ui.MainPanelsParams{
		Width:             m.Width,
		Height:            panelH,
		ShowBookmarks:     m.Config.Ui.ShowBookmarksPanel,
		Bookmarks:         m.Config.Bookmarks,
		BookmarkIndex:     m.BookmarkIndex,
		Entries:           m.FileEntries,
		FilteredIndices:   m.FilteredIndices,
		SelectedIndex:     m.SelectedIndex,
		CurrentDir:        m.CurrentDir,
		IsSearching:       m.IsSearching,
		SearchQuery:       m.SearchQuery,
		FocusedPanel:      int(m.FocusedPanel),
		ShowIcons:         m.Config.Ui.ShowIcons,
		ShowPreview:       m.Config.Ui.ShowPreviewPanel,
		PreviewLines:       previewLines,
		PreviewScroll:      m.PreviewScroll,
		PreviewHScroll:     m.PreviewHScroll,
		PreviewWrap:        m.PreviewWrap,
		PreviewLineNumbers: m.PreviewLineNumbers,
		PreviewFileName:    previewFileName,
		ShowPathClipboard: false,
		PathClipboard:     m.PathClipboard,
		PathClipboardIdx:  m.PathClipboardIdx,
		GitFileMap:        nil,
		GitRoot:           "",
		Config:            m.Config,
		FileListHeightOut:  &m.FileListHeight,
		ViewerHeightOut:    &m.ViewerHeight,
		PreviewWidthOut:    &m.PreviewContentWidth,
	}
	panels := ui.RenderMainPanels(panelParams)

	// Hint bar
	var hintBar string
	if hintH > 0 {
		hintBar = ui.RenderHintBar(int(m.Mode), int(m.FocusedPanel), m.IsSearching, false, m.Width)
	}

	// Status bar
	var statusBar string
	if statusH > 0 && m.Status != nil {
		statusBar = ui.RenderStatus(m.Status.Text, int(m.Status.Kind), m.Width)
	}

	// Assemble
	parts := []string{tabBar, panels}
	if statusH > 0 {
		parts = append(parts, statusBar)
	}
	if hintH > 0 {
		parts = append(parts, hintBar)
	}
	base := strings.Join(parts, "\n")

	// Overlays
	switch m.Mode {
	case ModeHelp:
		modal := ui.RenderHelpOverlay(m.Width, m.Height)
		return ui.PlaceOverlay(base, modal, m.Width, m.Height)
	case ModeCommandPalette:
		modal := ui.RenderPaletteOverlay("", nil, 0, m.Width, m.Height)
		return ui.PlaceOverlay(base, modal, m.Width, m.Height)
	case ModeOpenChoice:
		entry := m.SelectedEntry()
		name := ""
		if entry != nil {
			name = entry.Name
		}
		modal := ui.RenderOpenChoiceOverlay(m.OpenChoiceIndex, m.OpenChoiceIsDir, name, m.Width, m.Height)
		return ui.PlaceOverlay(base, modal, m.Width, m.Height)
	case ModeOpenWith:
		openers := make([]string, len(m.Config.Openers))
		for i, o := range m.Config.Openers {
			openers[i] = o.Name
		}
		modal := ui.RenderOpenWithOverlay(openers, m.OpenWithIndex, filepath.Base(m.SelectedPath()), m.Width, m.Height)
		return ui.PlaceOverlay(base, modal, m.Width, m.Height)
	case ModePathClipboard:
		modal := ui.RenderPathClipboardOverlay(m.PathClipboard, m.PathClipboardIdx, m.Width, m.Height)
		return ui.PlaceOverlay(base, modal, m.Width, m.Height)
	case ModeFileManager:
		opPtr := (*int)(nil)
		if m.FmOperation != nil {
			v := int(*m.FmOperation)
			opPtr = &v
		}
		fmp := &ui.FileManagerParams{
			MenuIdx:         m.FmMenuIdx,
			Input:           m.FmInput,
			Cursor:          m.FmCursor,
			Operation:       opPtr,
			Error:           m.FmError,
			OverwriteTarget: m.FmOverwriteTarget,
			SourcePath:      m.SelectedPath(),
			TargetDir:       m.CurrentDir,
			Width:           m.Width,
			Height:          m.Height,
		}
		modal := ui.RenderFileManager(fmp)
		return ui.PlaceOverlay(base, modal, m.Width, m.Height)
	}

	return base
}

// Ensure we use lipgloss somewhere (avoid import errors if not used elsewhere)
var _ = lipgloss.Color("")
