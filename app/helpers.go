package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/pickmoment/drf/fs"
	"github.com/pickmoment/drf/git"
	"github.com/pickmoment/drf/state"
	"github.com/pickmoment/drf/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// navigateTo changes the current directory and reloads the file list.
// Git status is refreshed asynchronously; returns the async tea.Cmd.
func (m *Model) navigateTo(dir string) tea.Cmd {
	entries, err := fs.ListDir(dir)
	if err != nil {
		m.SetStatusError("디렉토리 접근 실패: " + err.Error())
		return nil
	}
	m.CurrentDir = dir
	m.FileEntries = entries
	m.SelectedIndex = 0
	m.IsSearching = false
	m.SearchQuery = ""
	m.FilteredIndices = makeRange(len(entries))
	if !m.Config.General.ShowHidden {
		m.applyHiddenFilter()
	}
	m.PreviewScroll = 0
	m.PreviewHScroll = 0
	return gitRefreshAsync(dir)
}

// gitRefreshAsync runs git status in a goroutine and returns a tea.Cmd.
func gitRefreshAsync(dir string) tea.Cmd {
	return func() tea.Msg {
		return gitStatusMsg{status: git.GetStatus(dir), dir: dir}
	}
}

// goParent navigates to the parent directory.
func (m *Model) goParent() tea.Cmd {
	parent := filepath.Dir(m.CurrentDir)
	if parent == m.CurrentDir {
		return nil
	}
	prev := filepath.Base(m.CurrentDir)
	cmd := m.navigateTo(parent)
	for i, idx := range m.FilteredIndices {
		if idx < len(m.FileEntries) && m.FileEntries[idx].Name == prev {
			m.SelectedIndex = i
			break
		}
	}
	return cmd
}

// enterOrOpen shows the open choice dialog for the selected entry.
func (m *Model) enterOrOpen() tea.Cmd {
	entry := m.SelectedEntry()
	if entry == nil {
		return nil
	}
	m.Mode = ModeOpenChoice
	m.OpenChoiceIndex = 0
	m.OpenChoiceIsDir = entry.IsDir
	return nil
}

// openInVSCode opens a file or directory in VS Code.
func (m *Model) openInVSCode(path string) tea.Cmd {
	cmd := exec.Command("code", path)
	if err := cmd.Start(); err != nil {
		m.SetStatusError("VS Code 열기 실패: " + err.Error())
		return nil
	}
	m.SetStatusSuccess("VS Code로 열었습니다")
	return nil
}

// openInViewer opens a file in the fullscreen viewer.
func (m *Model) openInViewer(path string) tea.Cmd {
	entry, _ := fs.FromPath(path)
	maxWidth := 0
	if m.PreviewWrap {
		maxWidth = m.viewerContentWidth()
	}
	lines := m.buildPreviewLines(path, &entry, maxWidth)
	m.Mode = ModeViewer
	m.ViewerLines = lines
	m.ViewerPath = path
	m.PreviewScroll = 0
	m.PreviewHScroll = 0
	m.ViewerSearchQuery = ""
	m.ViewerIsSearching = false
	m.ViewerSearchMatches = nil
	m.ViewerSearchIdx = 0
	return nil
}

// updateFilter re-applies search filter to file entries.
func (m *Model) updateFilter() {
	if m.SearchQuery == "" {
		filtered := makeRange(len(m.FileEntries))
		if !m.Config.General.ShowHidden {
			var visible []int
			for _, i := range filtered {
				if !m.FileEntries[i].IsHidden {
					visible = append(visible, i)
				}
			}
			m.FilteredIndices = visible
		} else {
			m.FilteredIndices = filtered
		}
		return
	}
	query := strings.ToLower(m.SearchQuery)
	var result []int
	for i, e := range m.FileEntries {
		if !m.Config.General.ShowHidden && e.IsHidden {
			continue
		}
		if strings.Contains(strings.ToLower(e.Name), query) {
			result = append(result, i)
		}
	}
	m.FilteredIndices = result
	m.SelectedIndex = 0
}

// applyHiddenFilter filters out hidden files.
func (m *Model) applyHiddenFilter() {
	var visible []int
	for i, e := range m.FileEntries {
		if !e.IsHidden {
			visible = append(visible, i)
		}
	}
	m.FilteredIndices = visible
}

// toggleBookmark adds or removes the current directory from bookmarks.
func (m *Model) toggleBookmark() {
	dir := m.CurrentDir
	for i, bm := range m.Config.Bookmarks {
		if bm == dir {
			m.Config.Bookmarks = append(m.Config.Bookmarks[:i], m.Config.Bookmarks[i+1:]...)
			_ = m.Config.Save()
			m.SetStatusSuccess("즐겨찾기에서 제거됨")
			return
		}
	}
	m.Config.Bookmarks = append(m.Config.Bookmarks, dir)
	_ = m.Config.Save()
	m.SetStatusSuccess("즐겨찾기에 추가됨")
}

// bookmarkDelete removes the currently selected bookmark.
func (m *Model) bookmarkDelete() {
	if m.BookmarkIndex < len(m.Config.Bookmarks) {
		removed := m.Config.Bookmarks[m.BookmarkIndex]
		m.Config.Bookmarks = append(m.Config.Bookmarks[:m.BookmarkIndex], m.Config.Bookmarks[m.BookmarkIndex+1:]...)
		_ = m.Config.Save()
		m.SetStatusSuccess("즐겨찾기 삭제됨: " + removed)
		if m.BookmarkIndex >= len(m.Config.Bookmarks) && m.BookmarkIndex > 0 {
			m.BookmarkIndex--
		}
	}
}

// navigateToBookmark navigates to the currently selected bookmark.
func (m *Model) navigateToBookmark() tea.Cmd {
	if m.BookmarkIndex < len(m.Config.Bookmarks) {
		cmd := m.navigateTo(m.Config.Bookmarks[m.BookmarkIndex])
		m.FocusedPanel = PanelFileList
		return cmd
	}
	return nil
}

// togglePathClipboard adds or removes the selected path from the clipboard.
func (m *Model) togglePathClipboard() {
	path := m.SelectedPath()
	if path == "" {
		return
	}
	for i, p := range m.PathClipboard {
		if p == path {
			m.PathClipboard = append(m.PathClipboard[:i], m.PathClipboard[i+1:]...)
			m.SetStatusSuccess("경로 클립보드에서 제거됨")
			return
		}
	}
	m.PathClipboard = append(m.PathClipboard, path)
	m.SetStatusSuccess("경로 클립보드에 추가됨")
}

// moveUp moves selection up by 1.
func (m *Model) moveUp() {
	if m.SelectedIndex > 0 {
		m.SelectedIndex--
	}
}

// moveDown moves selection down by 1.
func (m *Model) moveDown() {
	if m.SelectedIndex < len(m.FilteredIndices)-1 {
		m.SelectedIndex++
	}
}

// jumpUp moves selection up by a page.
func (m *Model) jumpUp() {
	pageSize := m.FileListHeight
	if pageSize < 1 {
		pageSize = 10
	}
	m.SelectedIndex -= pageSize
	if m.SelectedIndex < 0 {
		m.SelectedIndex = 0
	}
}

// jumpDown moves selection down by a page.
func (m *Model) jumpDown() {
	pageSize := m.FileListHeight
	if pageSize < 1 {
		pageSize = 10
	}
	m.SelectedIndex += pageSize
	if m.SelectedIndex >= len(m.FilteredIndices) {
		m.SelectedIndex = max(0, len(m.FilteredIndices)-1)
	}
}

// clearViewerSearch clears the viewer search state.
func (m *Model) clearViewerSearch() {
	m.ViewerIsSearching = false
	m.ViewerSearchQuery = ""
	m.ViewerSearchMatches = nil
	m.ViewerSearchIdx = 0
}

// viewerDoSearch performs a search in the viewer.
func (m *Model) viewerDoSearch(query string) {
	m.ViewerSearchQuery = query
	m.ViewerSearchMatches = nil
	if query == "" {
		return
	}
	lower := strings.ToLower(query)
	for i, line := range m.ViewerLines {
		if strings.Contains(strings.ToLower(ui.StripANSIExport(line)), lower) {
			m.ViewerSearchMatches = append(m.ViewerSearchMatches, i)
		}
	}
	m.ViewerSearchIdx = 0
	if len(m.ViewerSearchMatches) > 0 {
		m.viewerJumpToMatch()
	}
}

// viewerJumpToMatch scrolls to the current search match.
func (m *Model) viewerJumpToMatch() {
	if len(m.ViewerSearchMatches) == 0 {
		return
	}
	idx := m.ViewerSearchIdx
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.ViewerSearchMatches) {
		idx = len(m.ViewerSearchMatches) - 1
	}
	m.ViewerSearchIdx = idx
	lineNo := m.ViewerSearchMatches[idx]
	h := m.ViewerHeight
	if h < 1 {
		h = 20
	}
	m.PreviewScroll = lineNo - h/2
	if m.PreviewScroll < 0 {
		m.PreviewScroll = 0
	}
}

// viewerNextMatch moves to the next search match.
func (m *Model) viewerNextMatch() {
	if len(m.ViewerSearchMatches) == 0 {
		return
	}
	m.ViewerSearchIdx = (m.ViewerSearchIdx + 1) % len(m.ViewerSearchMatches)
	m.viewerJumpToMatch()
}

// viewerPrevMatch moves to the previous search match.
func (m *Model) viewerPrevMatch() {
	if len(m.ViewerSearchMatches) == 0 {
		return
	}
	m.ViewerSearchIdx--
	if m.ViewerSearchIdx < 0 {
		m.ViewerSearchIdx = len(m.ViewerSearchMatches) - 1
	}
	m.viewerJumpToMatch()
}

// fmStart enters the file manager mode.
func (m *Model) fmStart() {
	m.Mode = ModeFileManager
	m.FmMenuIdx = 0
	m.FmInput = ""
	m.FmCursor = 0
	m.FmOperation = nil
	m.FmError = ""
	m.FmOverwriteTarget = ""
}

// fmRefreshFileList refreshes the file list after a file manager operation.
func (m *Model) fmRefreshFileList() {
	m.refreshFileList()
}

// refreshFileList reloads the current directory.
func (m *Model) refreshFileList() {
	entries, err := fs.ListDir(m.CurrentDir)
	if err != nil {
		return
	}
	m.FileEntries = entries
	m.updateFilter()
	if m.SelectedIndex >= len(m.FilteredIndices) && len(m.FilteredIndices) > 0 {
		m.SelectedIndex = len(m.FilteredIndices) - 1
	}
}

// openWithDefaultApp opens the selected file with the system default app.
func (m *Model) openWithDefaultApp(path string) tea.Cmd {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		m.SetStatusError("지원하지 않는 OS입니다")
		return nil
	}
	if err := cmd.Start(); err != nil {
		m.SetStatusError("열기 실패: " + err.Error())
		return nil
	}
	m.SetStatusSuccess("열었습니다: " + filepath.Base(path))
	return nil
}

// writeToClipboard writes text to the system clipboard via pbcopy/xclip/xsel.
func (m *Model) writeToClipboard(text string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
		if _, err := exec.LookPath("xclip"); err != nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	default:
		m.SetStatusError("클립보드 지원 안됨")
		return
	}
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		m.SetStatusError("클립보드 복사 실패")
		return
	}
	m.SetStatusSuccess("클립보드에 복사됨")
}

func readFromClipboard() (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-out")
		} else {
			cmd = exec.Command("xsel", "--clipboard", "--output")
		}
	default:
		return "", fmt.Errorf("클립보드 지원 안됨")
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// copyDragSelection handles mouse drag text selection copy.
func (m *Model) copyDragSelection(startLine, endLine int) {
	if len(m.ViewerLines) == 0 {
		return
	}
	if startLine > endLine {
		startLine, endLine = endLine, startLine
	}
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(m.ViewerLines) {
		endLine = len(m.ViewerLines) - 1
	}
	var sb strings.Builder
	for i := startLine; i <= endLine; i++ {
		sb.WriteString(ui.StripANSIExport(m.ViewerLines[i]))
		if i < endLine {
			sb.WriteString("\n")
		}
	}
	m.writeToClipboard(sb.String())
}

// readFileCapped reads a file up to maxSize bytes.
func readFileCapped(path string, maxSize int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024
	}
	buf := make([]byte, maxSize+1)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return "", err
	}
	data := buf[:n]
	if int64(n) > maxSize {
		data = buf[:maxSize]
	}
	return decodeBOM(data), nil
}

// decodeBOM converts BOM-prefixed byte slices to UTF-8 strings and normalizes line endings.
// Handles UTF-16 LE (FF FE), UTF-16 BE (FE FF), and UTF-8 BOM (EF BB BF).
func decodeBOM(data []byte) string {
	var s string
	switch {
	case len(data) >= 2 && data[0] == 0xff && data[1] == 0xfe:
		// UTF-16 LE BOM
		out := make([]rune, 0, len(data)/2)
		for i := 2; i+1 < len(data); i += 2 {
			r := rune(uint16(data[i]) | uint16(data[i+1])<<8)
			out = append(out, r)
		}
		s = string(out)
	case len(data) >= 2 && data[0] == 0xfe && data[1] == 0xff:
		// UTF-16 BE BOM
		out := make([]rune, 0, len(data)/2)
		for i := 2; i+1 < len(data); i += 2 {
			r := rune(uint16(data[i])<<8 | uint16(data[i+1]))
			out = append(out, r)
		}
		s = string(out)
	case len(data) >= 3 && data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf:
		// UTF-8 BOM
		s = string(data[3:])
	default:
		s = string(data)
	}
	// Normalize CRLF → LF so \r never reaches the terminal renderer.
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// loadPreviewForSelected loads preview content for the selected file (preview panel).
func (m *Model) loadPreviewForSelected() {
	entry := m.SelectedEntry()
	if entry == nil || entry.IsDir {
		m.ViewerLines = nil
		m.ViewerPath = ""
		return
	}
	maxWidth := 0
	if m.PreviewWrap {
		maxWidth = m.previewContentWidth()
	}
	m.ViewerLines = m.buildPreviewLines(entry.Path, entry, maxWidth)
	m.ViewerPath = entry.Path
	m.PreviewScroll = 0
	m.PreviewHScroll = 0
}

// reloadViewerLines re-renders the current viewer file (e.g. after wrap toggle),
// preserving the scroll position. Uses the full viewer width, not the preview panel width.
func (m *Model) reloadViewerLines() {
	if m.ViewerPath == "" {
		return
	}
	maxWidth := 0
	if m.PreviewWrap {
		maxWidth = m.viewerContentWidth()
	}
	scroll := m.PreviewScroll
	m.ViewerLines = m.buildPreviewLines(m.ViewerPath, nil, maxWidth)
	m.PreviewScroll = scroll
	if m.PreviewScroll >= len(m.ViewerLines) {
		m.PreviewScroll = max(0, len(m.ViewerLines)-1)
	}
}

// buildPreviewLines detects file type, handles binary/non-previewable files,
// and returns rendered lines for display in the preview panel or fullscreen viewer.
// maxWidth: table column wrap limit (0 = unlimited).
func (m *Model) buildPreviewLines(path string, entry *fs.FileEntry, maxWidth int) []string {
	fileType, codeLang := DetectFileType(path)

	// Non-text types: show rich file info without reading content
	switch fileType {
	case FileTypeBinary, FileTypeImage, FileTypePdf, FileTypeParquet, FileTypeArchive:
		return m.renderFileInfo(path, entry, int(fileType))
	}

	// For unknown extension (no extension or unrecognized), probe for binary content.
	// Known extensions are trusted as-is.
	if fileType == FileTypeUnknown {
		if probe, err := readFileCapped(path, 512); err == nil && isBinaryContent(probe) {
			return m.renderFileInfo(path, entry, int(FileTypeBinary))
		}
	}

	content, err := readFileCapped(path, m.Config.Preview.MaxFileSize)
	if err != nil {
		return []string{"  파일을 읽을 수 없습니다: " + err.Error()}
	}

	return ui.RenderPreviewContent(
		path, content,
		int(fileType), string(codeLang),
		m.Config.Preview.SyntaxTheme,
		m.Config.Preview.MarkdownRender,
		maxWidth,
	)
}

func (m *Model) renderFileInfo(path string, entry *fs.FileEntry, fileType int) []string {
	if entry != nil {
		return ui.RenderFileInfo(*entry, fileType)
	}
	if fe, err := fs.FromPath(path); err == nil {
		return ui.RenderFileInfo(fe, fileType)
	}
	return []string{"  미리보기 불가 파일"}
}

// previewContentWidth returns the inner content width of the side preview panel.
// Uses the stored value from the last rendered frame; estimates on first call.
func (m *Model) previewContentWidth() int {
	if m.PreviewContentWidth > 0 {
		return m.PreviewContentWidth
	}
	// Estimate: 65% of total width minus borders
	w := m.Width*65/100 - 2
	if w < 10 {
		w = 10
	}
	return w
}

// viewerContentWidth returns the usable content width for the fullscreen viewer.
// The viewer occupies the full terminal width minus the border.
func (m *Model) viewerContentWidth() int {
	w := m.Width - 2
	if w < 10 {
		w = 10
	}
	return w
}

// isBinaryContent returns true if the content appears to be binary data.
func isBinaryContent(content string) bool {
	if len(content) == 0 {
		return false
	}
	sample := content
	if len(sample) > 512 {
		sample = content[:512]
	}
	// Any null byte → definitely binary
	for i := 0; i < len(sample); i++ {
		if sample[i] == 0 {
			return true
		}
	}
	// Trim incomplete trailing UTF-8 sequence (capped sample may end mid-char)
	for len(sample) > 0 && !utf8.ValidString(sample) {
		sample = sample[:len(sample)-1]
	}
	// Valid UTF-8 is text (covers Korean, Chinese, Japanese, etc.)
	if utf8.ValidString(sample) {
		nonText := 0
		total := 0
		for _, r := range sample {
			total++
			if r != '\t' && r != '\n' && r != '\r' && (r < 32 || r == 127) {
				nonText++
			}
		}
		if total == 0 {
			return false
		}
		return float64(nonText)/float64(total) > 0.15
	}
	// Invalid UTF-8: count non-text bytes by proportion
	nonText := 0
	for i := 0; i < len(sample); i++ {
		b := sample[i]
		switch {
		case b == 9 || b == 10 || b == 13:
		case b < 32 || b == 127:
			nonText++
		case b >= 128:
			nonText++
		}
	}
	return float64(nonText)/float64(len(sample)) > 0.15
}

// gitRefresh refreshes the git state.
func (m *Model) gitRefresh() {
	if m.Git == nil {
		return
	}
	root := m.CurrentDir
	if m.Git.Status != nil {
		root = m.Git.Status.Root
	}
	m.Git.Refresh(root)
	m.Git.LoadDiff()
}

// gitRoot returns the current git root directory.
func (m *Model) gitRoot() string {
	if m.Git != nil && m.Git.Status != nil {
		return m.Git.Status.Root
	}
	return git.FindRoot(m.CurrentDir)
}

// executeOpenWith runs the selected opener on the selected file.
func (m *Model) executeOpenWith() tea.Cmd {
	path := m.SelectedPath()
	if path == "" {
		return nil
	}
	if m.OpenWithIndex >= len(m.Config.Openers) {
		return nil
	}
	opener := m.Config.Openers[m.OpenWithIndex]
	if opener.Terminal {
		// Return a tea.Cmd to open the terminal command
		m.PendingTerminalOpener = &PendingTerminalOpener{
			Cmd:  opener.Command,
			Args: append(opener.Args, path),
		}
		m.Mode = ModeFileList
		return nil
	}
	args := append(opener.Args, path)
	cmd := exec.Command(opener.Command, args...)
	if err := cmd.Start(); err != nil {
		m.SetStatusError("열기 실패: " + err.Error())
		return nil
	}
	m.SetStatusSuccess("열었습니다: " + opener.Name)
	m.Mode = ModeFileList
	return nil
}

// startGitAsync starts an async git operation.
// It spawns exactly one goroutine that waits for the command and sends the result
// to m.AsyncResult. checkAsyncDone polls that channel each tick without re-spawning.
func (m *Model) startGitAsync(data state.AsyncData) tea.Cmd {
	if m.Git == nil || m.Git.Status == nil {
		return nil
	}
	root := m.Git.Status.Root
	cmd := m.Git.StartAsync(data, root)
	m.Git.AsyncCmd = cmd
	if err := cmd.Start(); err != nil {
		m.Git.AsyncData = nil
		m.Git.AsyncCmd = nil
		m.SetStatusError("명령 실행 실패: " + err.Error())
		return nil
	}
	// One goroutine per async operation; buffered so it never blocks.
	m.AsyncResult = make(chan error, 1)
	go func(result chan<- error) {
		result <- cmd.Wait()
	}(m.AsyncResult)
	return nil
}

// checkAsyncDone non-blockingly checks if the async git operation has finished.
func (m *Model) checkAsyncDone() {
	if m.Git == nil || m.Git.AsyncCmd == nil || m.AsyncResult == nil {
		return
	}
	select {
	case err := <-m.AsyncResult:
		if err != nil {
			m.SetStatusError("Git 작업 실패: " + err.Error())
		} else {
			opName := asyncOpName(m.Git.AsyncData)
			m.SetStatusSuccess("Git " + opName + " 완료")
		}
		m.AsyncResult = nil
		m.finishAsync()
	default:
		// Still running
	}
}

func (m *Model) finishAsync() {
	if m.Git == nil {
		return
	}
	m.Git.AsyncData = nil
	m.Git.AsyncCmd = nil
	m.gitRefresh()
}

func asyncOpName(data *state.AsyncData) string {
	if data == nil {
		return "작업"
	}
	switch data.Kind {
	case state.AsyncPush:
		return "푸시"
	case state.AsyncPull:
		return "풀"
	case state.AsyncFetch:
		return "페치"
	}
	return "작업"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
