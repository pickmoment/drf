package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/pickmoment/drf/config"
	"github.com/pickmoment/drf/fs"
	"github.com/pickmoment/drf/state"
)

// Colors used throughout the UI
const (
	ColorBorder        = lipgloss.Color("240")
	ColorBorderFocused = lipgloss.Color("39")
	ColorBorderAlt     = lipgloss.Color("33")
	ColorSelected      = lipgloss.Color("24")
	ColorDirFg         = lipgloss.Color("33")
	ColorSearchFg      = lipgloss.Color("220")
	ColorStatusOkFg    = lipgloss.Color("46")
	ColorStatusErrFg   = lipgloss.Color("196")
	ColorGitStagedFg   = lipgloss.Color("46")
	ColorGitUnstagedFg = lipgloss.Color("202")
	ColorGitAddedFg    = lipgloss.Color("46")
	ColorGitDeletedFg  = lipgloss.Color("196")
	ColorGitHunkFg     = lipgloss.Color("33")
	ColorHintKey       = lipgloss.Color("250")
	ColorHintDesc      = lipgloss.Color("244")
)

// MainPanelsParams holds all data needed to render the 3-panel main view.
type MainPanelsParams struct {
	Width, Height int
	// Bookmarks panel
	ShowBookmarks    bool
	Bookmarks        []string
	BookmarkIndex    int
	// File list panel
	Entries          []fs.FileEntry
	FilteredIndices  []int
	SelectedIndex    int
	CurrentDir       string
	IsSearching      bool
	SearchQuery      string
	FocusedPanel     int // 0=FileList, 1=Bookmarks, 2=PathClipboard
	ShowIcons        bool
	// Preview panel
	ShowPreview        bool
	PreviewLines       []string
	PreviewScroll      int
	PreviewHScroll     int
	PreviewWrap        bool
	PreviewLineNumbers bool
	PreviewFileName    string
	// Path clipboard sidebar
	ShowPathClipboard bool
	PathClipboard     []string
	PathClipboardIdx  int
	// Layout-computed sizes (set by RenderMainPanels as out-parameters)
	FileListHeightOut  *int
	ViewerHeightOut    *int
	PreviewWidthOut    *int // inner content width of the preview panel
	// Git status marker map
	GitFileMap map[string][2]byte
	GitRoot    string
	// Config
	Config config.Config
}

// RenderMainPanels renders the 3-panel layout and returns the complete string.
func RenderMainPanels(p *MainPanelsParams) string {
	// Layout: [bookmarks?] [file list] [path-clipboard?] [preview?]
	totalW := p.Width

	// Compute panel widths
	bkW := 0
	if p.ShowBookmarks {
		bkW = totalW * 15 / 100
		if bkW < 16 {
			bkW = 16
		}
	}
	pcW := 0
	if p.ShowPathClipboard {
		pcW = totalW * 18 / 100
		if pcW < 18 {
			pcW = 18
		}
	}
	previewW := 0
	remaining := totalW - bkW - pcW
	flW := remaining
	if p.ShowPreview {
		previewW = remaining * 65 / 100
		flW = remaining - previewW
	}
	if flW < 10 {
		flW = 10
	}

	panelH := p.Height
	if p.FileListHeightOut != nil {
		*p.FileListHeightOut = panelH - 2 // inner height (minus border)
	}
	if p.ViewerHeightOut != nil {
		*p.ViewerHeightOut = panelH - 2
	}
	if p.PreviewWidthOut != nil && p.ShowPreview {
		*p.PreviewWidthOut = previewW - 2 // inner width (minus border)
	}

	var cols []string

	if p.ShowBookmarks {
		cols = append(cols, renderBookmarksPanel(p.Bookmarks, p.BookmarkIndex, bkW, panelH, p.FocusedPanel == 1))
	}

	fileListFocused := p.FocusedPanel == 0
	cols = append(cols, RenderFileList(&FileListParams{
		Entries:         p.Entries,
		FilteredIndices: p.FilteredIndices,
		SelectedIndex:   p.SelectedIndex,
		CurrentDir:      p.CurrentDir,
		IsSearching:     p.IsSearching,
		SearchQuery:     p.SearchQuery,
		Width:           flW,
		Height:          panelH,
		Focused:         fileListFocused,
		ShowIcons:       p.ShowIcons,
		GitFileMap:      p.GitFileMap,
		GitRoot:         p.GitRoot,
	}))

	if p.ShowPathClipboard {
		cols = append(cols, renderClipboardSidebar(p.PathClipboard, p.PathClipboardIdx, pcW, panelH, p.FocusedPanel == 2))
	}

	if p.ShowPreview {
		cols = append(cols, renderPreviewPanel(p.PreviewLines, p.PreviewScroll, p.PreviewHScroll, p.PreviewWrap, p.PreviewLineNumbers, p.PreviewFileName, previewW, panelH))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}

// renderBookmarksPanel renders the bookmarks sidebar.
func renderBookmarksPanel(bookmarks []string, selectedIdx, w, h int, focused bool) string {
	borderColor := ColorBorder
	if focused {
		borderColor = ColorBorderFocused
	}
	innerW := w - 2
	if innerW < 1 {
		innerW = 1
	}
	innerH := h - 2
	if innerH < 1 {
		innerH = 1
	}

	selStyle := lipgloss.NewStyle().Background(ColorSelected)
	normalStyle := lipgloss.NewStyle()

	var lines []string
	for i, bm := range bookmarks {
		label := padRight(truncateLeft(bm, innerW), innerW)
		if i == selectedIdx {
			lines = append(lines, selStyle.Render(label))
		} else {
			lines = append(lines, normalStyle.Render(label))
		}
	}
	if len(bookmarks) == 0 {
		empty := truncateStr("(즐겨찾기 없음)", innerW)
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render(padRight(empty, innerW)))
	}

	content := joinLines(lines, innerH, innerW)
	return renderBox("즐겨찾기", content, w, h, borderColor)
}

// renderClipboardSidebar renders the path clipboard sidebar.
func renderClipboardSidebar(items []string, selectedIdx, w, h int, focused bool) string {
	borderColor := ColorBorder
	if focused {
		borderColor = ColorBorderFocused
	}
	innerW := w - 2
	if innerW < 1 {
		innerW = 1
	}
	innerH := h - 2
	if innerH < 1 {
		innerH = 1
	}

	selStyle := lipgloss.NewStyle().Background(ColorSelected)
	normalStyle := lipgloss.NewStyle()

	var lines []string
	for i, item := range items {
		label := padRight(truncateLeft(item, innerW), innerW)
		if i == selectedIdx {
			lines = append(lines, selStyle.Render(label))
		} else {
			lines = append(lines, normalStyle.Render(label))
		}
	}
	if len(items) == 0 {
		empty := truncateStr("(비어 있음)", innerW)
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render(padRight(empty, innerW)))
	}

	content := joinLines(lines, innerH, innerW)
	return renderBox("경로 클립보드", content, w, h, borderColor)
}

// renderPreviewPanel renders the file preview panel.
func renderPreviewPanel(lines []string, scroll, hScroll int, wrap, lineNumbers bool, filename string, w, h int) string {
	innerW := w - 2
	if innerW < 1 {
		innerW = 1
	}
	innerH := h - 2
	if innerH < 1 {
		innerH = 1
	}

	// Show placeholder when no file is selected
	if len(lines) == 0 {
		placeholder := "파일을 선택하면 미리보기가 표시됩니다."
		hint := "Enter 또는 → 로 열기"
		phStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		hintStyleP := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
		var phLines []string
		midY := innerH / 2
		for i := 0; i < innerH; i++ {
			if i == midY {
				phLines = append(phLines, padRight(phStyle.Render(truncateStr(placeholder, innerW)), innerW))
			} else if i == midY+1 {
				phLines = append(phLines, padRight(hintStyleP.Render(truncateStr(hint, innerW)), innerW))
			} else {
				phLines = append(phLines, strings.Repeat(" ", innerW))
			}
		}
		return renderBox("미리보기", strings.Join(phLines, "\n"), w, h, ColorBorder)
	}

	// Line number gutter width: digits needed to represent the last line + " │ "
	lnW := 0
	if lineNumbers && len(lines) > 0 {
		lnW = len(fmt.Sprintf("%d", len(lines))) + 2 // digits + " │"
	}
	contentW := innerW - lnW
	if contentW < 1 {
		contentW = 1
	}
	lnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	lnSepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

	var visibleLines []string
	for i := scroll; i < len(lines) && len(visibleLines) < innerH; i++ {
		rawLine := lines[i]
		plain := sanitizeControlChars(expandTabs(stripANSI(rawLine), 4))

		var rendered []string
		isANSI := strings.Contains(rawLine, "\x1b")
		if wrap {
			expanded := expandTabsANSI(rawLine, 4)
			// Don't word-wrap box-drawing table lines; they have fixed visual structure.
			if startsWithBoxDrawing(plain) {
				rendered = append(rendered, padRight(truncateANSI(expanded, contentW), contentW))
			} else {
				wrapped := wrapLineANSI(expanded, contentW)
				for _, wl := range wrapped {
					rendered = append(rendered, padRight(truncateANSI(wl, contentW), contentW))
				}
			}
		} else if isANSI {
			expanded := expandTabsANSI(rawLine, 4)
			if hScroll > 0 {
				expanded = hScrollANSI(expanded, hScroll)
			}
			rendered = append(rendered, padRight(truncateANSI(expanded, contentW), contentW))
		} else {
			if hScroll > 0 {
				runes := []rune(plain)
				if hScroll < len(runes) {
					plain = string(runes[hScroll:])
				} else {
					plain = ""
				}
			}
			rendered = append(rendered, padRight(truncateStr(plain, contentW), contentW))
		}

		for subIdx, rl := range rendered {
			if len(visibleLines) >= innerH {
				break
			}
			if lineNumbers {
				var gutter string
				if subIdx == 0 {
					numStr := fmt.Sprintf("%*d", lnW-2, i+1)
					gutter = lnStyle.Render(numStr) + lnSepStyle.Render("│")
				} else {
					gutter = strings.Repeat(" ", lnW-1) + lnSepStyle.Render("│")
				}
				visibleLines = append(visibleLines, gutter+rl)
			} else {
				visibleLines = append(visibleLines, rl)
			}
		}
	}

	content := joinLines(visibleLines, innerH, innerW)
	title := "미리보기"
	if filename != "" {
		title = truncateStr(filename, innerW-4)
	}
	return renderBox(title, content, w, h, ColorBorder)
}

// RenderTabBar renders a path bar at the top with git branch and file count.
func RenderTabBar(currentDir, gitBranch string, gitDirty bool, fileCount, width int) string {
	bg := lipgloss.Color("236")
	logoStyle := lipgloss.NewStyle().Background(lipgloss.Color("27")).Foreground(lipgloss.Color("255")).Bold(true)
	sepStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("244"))
	pathStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("252"))
	branchStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("114"))
	dirtyStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("208"))
	countStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("244"))

	logo := logoStyle.Render(" drf ")
	sep := sepStyle.Render(" │ ")

	// Right section: branch + dirty indicator + count
	right := ""
	if gitBranch != "" {
		branch := "⎇ " + gitBranch
		if gitDirty {
			right += "  " + branchStyle.Render(branch) + dirtyStyle.Render(" ●")
		} else {
			right += "  " + branchStyle.Render(branch)
		}
	}
	if fileCount >= 0 {
		right += "  " + countStyle.Render(fmt.Sprintf("%d 파일", fileCount))
	}
	right += " "
	rightW := lipgloss.Width(right)

	// Left section: logo + sep + path
	leftFixed := lipgloss.Width(logo) + lipgloss.Width(sep)
	pathMaxW := width - leftFixed - rightW
	if pathMaxW < 4 {
		pathMaxW = 4
	}
	path := pathStyle.Render(" " + truncateStr(currentDir, pathMaxW-1))

	// Assemble: [logo][sep][path][padding][right]
	usedW := leftFixed + lipgloss.Width(path) + rightW
	pad := ""
	if usedW < width {
		pad = lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-usedW))
	}
	return logo + sep + path + pad + right
}

// RenderHelpOverlay renders the help modal content.
func RenderHelpOverlay(w, h int) string {
	type entry struct{ key, desc string }
	entries := []entry{
		{"j / k / ↑ / ↓", "항목 이동"},
		{"Enter", "열기 / 디렉토리 이동"},
		{"h / ← / Backspace", "상위 디렉토리"},
		{"l / →", "미리보기 열기"},
		{"Space", "전체화면 뷰어"},
		{"/", "검색 시작"},
		{"Esc", "검색 취소"},
		{"b", "즐겨찾기 추가/제거"},
		{"B", "즐겨찾기 패널"},
		{"g", "Git 패널"},
		{"p", "경로 클립보드 추가/제거"},
		{"P", "경로 클립보드 보기"},
		{"o", "열기 선택"},
		{"e", "편집기로 열기"},
		{"n", "새 폴더 생성"},
		{"r", "이름 변경"},
		{"d", "삭제"},
		{"c", "복사"},
		{"m", "이동"},
		{"W", "줄바꿈 토글"},
		{".", "숨김파일 토글"},
		{"Q / Ctrl+C", "종료"},
	}

	keyStyle := lipgloss.NewStyle().Foreground(ColorHintKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ColorHintDesc)

	const keyColW = 20 // visual columns reserved for key part
	var lines []string
	for _, e := range entries {
		k := padRight(keyStyle.Render(e.key), keyColW)
		d := descStyle.Render(e.desc)
		lines = append(lines, k+d)
	}

	content := strings.Join(lines, "\n")
	title := lipgloss.NewStyle().Bold(true).Foreground(ColorBorderFocused).Render("도움말")
	hint := lipgloss.NewStyle().Faint(true).Render("  q / Esc: 닫기")

	// divider width: keyColW + max desc visual width (~22) = ~42
	divider := lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", keyColW+22))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorderFocused).
		Padding(0, 1)

	return style.Render(title + hint + "\n" + divider + "\n" + content)
}

// RenderPaletteOverlay renders the command palette modal.
func RenderPaletteOverlay(query string, results []string, selectedIdx, w, h int) string {
	innerW := 50
	if innerW > w-4 {
		innerW = w - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorderFocused).Render("명령 팔레트"))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorSearchFg).Render("> " + query))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")

	maxItems := 10
	if h-6 < maxItems {
		maxItems = h - 6
	}
	start := 0
	if selectedIdx >= maxItems {
		start = selectedIdx - maxItems + 1
	}
	for i := start; i < len(results) && i < start+maxItems; i++ {
		label := truncateStr(results[i], innerW)
		label = padRight(label, innerW)
		if i == selectedIdx {
			sb.WriteString(lipgloss.NewStyle().Background(ColorSelected).Render(label))
		} else {
			sb.WriteString(label)
		}
		sb.WriteString("\n")
	}
	if len(results) == 0 {
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render("결과 없음"))
		sb.WriteString("\n")
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorderFocused).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}

// RenderOpenChoiceOverlay renders the open-choice modal (Enter key on files/dirs).
func RenderOpenChoiceOverlay(selectedIdx int, isDir bool, filename string, w, h int) string {
	innerW := 36
	if innerW > w-4 {
		innerW = w - 4
	}

	items := []string{"기본 앱으로 열기", "VS Code로 열기"}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("파일 열기"))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(truncateStr(filename, innerW)))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")

	for i, item := range items {
		label := padRight(truncateStr(item, innerW), innerW)
		if i == selectedIdx {
			sb.WriteString(lipgloss.NewStyle().Background(ColorSelected).Render(label))
		} else {
			sb.WriteString(label)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter: 선택  Esc: 취소"))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorderFocused).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}

// RenderOpenWithOverlay renders the "open with" selection modal.
func RenderOpenWithOverlay(items []string, selectedIdx int, filename string, w, h int) string {
	innerW := 44
	if innerW > w-4 {
		innerW = w - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("프로그램으로 열기"))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(truncateStr(filename, innerW)))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")

	for i, item := range items {
		label := truncateStr(item, innerW)
		label = padRight(label, innerW)
		if i == selectedIdx {
			sb.WriteString(lipgloss.NewStyle().Background(ColorSelected).Render(label))
		} else {
			sb.WriteString(label)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter: 선택  Esc: 취소"))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorderFocused).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}

// RenderPathClipboardOverlay renders the path clipboard modal.
func RenderPathClipboardOverlay(items []string, selectedIdx, w, h int) string {
	innerW := 60
	if innerW > w-4 {
		innerW = w - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("경로 클립보드"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")

	maxItems := 15
	if h-6 < maxItems {
		maxItems = h - 6
	}
	start := 0
	if selectedIdx >= maxItems {
		start = selectedIdx - maxItems + 1
	}

	for i := start; i < len(items) && i < start+maxItems; i++ {
		label := padRight(truncateLeft(items[i], innerW), innerW)
		if i == selectedIdx {
			sb.WriteString(lipgloss.NewStyle().Background(ColorSelected).Render(label))
		} else {
			sb.WriteString(label)
		}
		sb.WriteString("\n")
	}

	if len(items) == 0 {
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render("(비어 있음)"))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter: 이동  y: 복사  d: 삭제  Esc: 닫기"))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorderFocused).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}

// PlaceOverlay centers a modal over a base view by directly replacing characters
// in the appropriate lines. This works correctly with bubbletea's renderer which
// truncates lines at terminal width, stripping any appended ANSI cursor sequences.
func PlaceOverlay(base, modal string, w, h int) string {
	baseLines := strings.Split(base, "\n")
	modalLines := strings.Split(modal, "\n")

	oh := len(modalLines)
	ow := 0
	for _, l := range modalLines {
		lw := lipgloss.Width(l)
		if lw > ow {
			ow = lw
		}
	}

	startY := (h - oh) / 2
	startX := (w - ow) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for i, modalLine := range modalLines {
		baseIdx := startY + i
		if baseIdx < 0 || baseIdx >= len(baseLines) {
			continue
		}
		left := overlayLeft(baseLines[baseIdx], startX)
		right := hScrollANSI(baseLines[baseIdx], startX+ow)
		baseLines[baseIdx] = left + modalLine + right
	}

	return strings.Join(baseLines, "\n")
}

// overlayLeft returns the first maxW visual columns of an ANSI-decorated string,
// padding with spaces to exactly maxW columns and closing with a reset.
func overlayLeft(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	type tok struct {
		ansi string
		r    rune
		w    int
	}
	var toks []tok
	i, pending := 0, ""
	for i < len(s) {
		if i+1 < len(s) && s[i] == '\x1b' && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			pending += s[i:j]
			i = j
			continue
		}
		r, sz := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && sz == 1 {
			i++
			continue
		}
		toks = append(toks, tok{ansi: pending, r: r, w: runewidth.RuneWidth(r)})
		pending = ""
		i += sz
	}

	col := 0
	var b strings.Builder
	for _, t := range toks {
		if col+t.w > maxW {
			break
		}
		b.WriteString(t.ansi)
		b.WriteRune(t.r)
		col += t.w
	}
	// Pad to exactly maxW if short
	for col < maxW {
		b.WriteByte(' ')
		col++
	}
	if b.Len() > 0 {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// renderBox draws a bordered box with a title and content.
func renderBox(title, content string, w, h int, borderColor lipgloss.Color) string {
	innerW := w - 2
	if innerW < 1 {
		innerW = 1
	}

	bc := lipgloss.NewStyle().Foreground(borderColor)

	// Top border with title
	// Use lipgloss.Width so ANSI-decorated titles (e.g. with colour indicators)
	// are measured correctly.
	titleStr := ""
	if title != "" {
		titleStr = " " + title + " "
	}
	topLineLen := innerW - lipgloss.Width(titleStr)
	if topLineLen < 0 {
		topLineLen = 0
		// Strip ANSI then truncate to avoid cutting inside an escape sequence.
		titleStr = " " + truncateStr(stripANSI(title), innerW-2) + " "
	}
	leftLen := 0
	rightLen := topLineLen
	top := bc.Render("┌"+strings.Repeat("─", leftLen)) + titleStr + bc.Render(strings.Repeat("─", rightLen)+"┐")

	// Bottom border
	bottom := bc.Render("└" + strings.Repeat("─", innerW) + "┘")

	// Content lines
	contentLines := strings.Split(content, "\n")
	innerH := h - 2
	if innerH < 0 {
		innerH = 0
	}

	var bodyLines []string
	for i := 0; i < innerH; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		}
		// Pad to innerW (visual width)
		vw := lipgloss.Width(line)
		if vw < innerW {
			line += strings.Repeat(" ", innerW-vw)
		}
		bodyLines = append(bodyLines, bc.Render("│")+line+bc.Render("│"))
	}

	rows := []string{top}
	rows = append(rows, bodyLines...)
	rows = append(rows, bottom)
	return strings.Join(rows, "\n")
}

// joinLines joins lines, padding/truncating to fit innerH rows, each of innerW visual width.
func joinLines(lines []string, innerH, innerW int) string {
	var result []string
	for i := 0; i < innerH; i++ {
		if i < len(lines) {
			result = append(result, lines[i])
		} else {
			result = append(result, strings.Repeat(" ", innerW))
		}
	}
	return strings.Join(result, "\n")
}

// truncateStr truncates a plain string to at most maxW visual columns.
// Appends "…" (1 column) when truncation occurs.
func truncateStr(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	return runewidth.Truncate(s, maxW, "…")
}

// truncateLeft truncates a plain string from the LEFT to at most maxW visual
// columns, prepending "…" so the trailing (most specific) part is visible.
// For path strings it prefers to cut at a '/' boundary for readability.
func truncateLeft(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxW {
		return s
	}
	runes := []rune(s)
	// Walk from the right to find the character-level cut point (maxW-1 cols, 1 for "…")
	col := 0
	cutIdx := len(runes) // fallback: keep everything (won't happen after the width check above)
	for i := len(runes) - 1; i >= 0; i-- {
		rw := runewidth.RuneWidth(runes[i])
		if col+rw > maxW-1 {
			cutIdx = i + 1
			break
		}
		col += rw
	}
	// Prefer cutting at a '/' so path components stay intact
	for j := cutIdx; j < len(runes); j++ {
		if runes[j] == '/' {
			tail := string(runes[j:])
			if 1+runewidth.StringWidth(tail) <= maxW {
				return "…" + tail
			}
			break
		}
	}
	return "…" + string(runes[cutIdx:])
}

// padRight pads s (which may contain ANSI escape codes) to exactly w visual
// columns by appending spaces. lipgloss.Width handles ANSI stripping and CJK.
func padRight(s string, w int) string {
	cur := lipgloss.Width(s)
	if cur >= w {
		return s
	}
	return s + strings.Repeat(" ", w-cur)
}

// hScrollANSI skips the first `skip` visual columns of an ANSI-decorated string,
// restoring any active style codes at the start of the returned portion.
func hScrollANSI(s string, skip int) string {
	if skip <= 0 {
		return s
	}
	// Tokenise into (ANSI-prefix, rune, width) entries.
	type tok struct {
		ansi string
		r    rune
		w    int
	}
	var toks []tok
	trailing := ""
	i, pending := 0, ""
	for i < len(s) {
		if i+1 < len(s) && s[i] == '\x1b' && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			pending += s[i:j]
			i = j
			continue
		}
		r, sz := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && sz == 1 {
			i++
			continue
		}
		toks = append(toks, tok{ansi: pending, r: r, w: runewidth.RuneWidth(r)})
		pending = ""
		i += sz
	}
	trailing = pending

	// Find first visible token at or past column `skip`.
	col := 0
	startIdx := len(toks)
	for idx, t := range toks {
		if col+t.w > skip {
			startIdx = idx
			break
		}
		col += t.w
		if col == skip {
			startIdx = idx + 1
			break
		}
	}

	// Reconstruct ANSI state up to startIdx so colours are preserved.
	parseState := func(codes string, active *[]string) {
		j := 0
		for j < len(codes) {
			if j+1 < len(codes) && codes[j] == '\x1b' && codes[j+1] == '[' {
				k := j + 2
				for k < len(codes) && codes[k] != 'm' {
					k++
				}
				if k < len(codes) {
					k++
				}
				c := codes[j:k]
				if c == "\x1b[0m" {
					*active = (*active)[:0]
				} else {
					*active = append(*active, c)
				}
				j = k
			} else {
				j++
			}
		}
	}
	var active []string
	for idx := 0; idx < startIdx; idx++ {
		parseState(toks[idx].ansi, &active)
	}

	var b strings.Builder
	for _, c := range active {
		b.WriteString(c)
	}
	for idx := startIdx; idx < len(toks); idx++ {
		b.WriteString(toks[idx].ansi)
		b.WriteRune(toks[idx].r)
	}
	b.WriteString(trailing)
	if b.Len() > 0 {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// startsWithBoxDrawing returns true if the first non-space rune of a plain
// string is a Unicode Box Drawing character (U+2500–U+257F).
func startsWithBoxDrawing(plain string) bool {
	for _, r := range plain {
		if r == ' ' {
			continue
		}
		return r >= 0x2500 && r <= 0x257F
	}
	return false
}

// wrapLine wraps a plain string at maxW visual columns.
func wrapLine(s string, maxW int) []string {
	if maxW <= 0 {
		return []string{s}
	}
	if runewidth.StringWidth(s) <= maxW {
		return []string{s}
	}
	wrapped := runewidth.Wrap(s, maxW)
	return strings.Split(wrapped, "\n")
}

// wrapLineANSI wraps a string that may contain ANSI escape codes at maxW visual
// columns, breaking at word boundaries and re-asserting active styles on each new line.
func wrapLineANSI(s string, maxW int) []string {
	if maxW <= 0 {
		return []string{s}
	}
	if runewidth.StringWidth(stripANSI(s)) <= maxW {
		return []string{s}
	}

	// Tokenise: each entry is one visible rune with any preceding ANSI codes.
	type tok struct {
		ansi string
		r    rune
		w    int
	}
	var toks []tok
	trailing := ""
	i := 0
	pending := ""
	for i < len(s) {
		if i+1 < len(s) && s[i] == '\x1b' && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			pending += s[i:j]
			i = j
			continue
		}
		r, sz := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && sz == 1 {
			i++
			continue
		}
		toks = append(toks, tok{ansi: pending, r: r, w: runewidth.RuneWidth(r)})
		pending = ""
		i += sz
	}
	trailing = pending

	// Track active ANSI codes for re-assertion at each line start.
	var active []string
	parseActive := func(codes string) {
		j := 0
		for j < len(codes) {
			if j+1 < len(codes) && codes[j] == '\x1b' && codes[j+1] == '[' {
				k := j + 2
				for k < len(codes) && codes[k] != 'm' {
					k++
				}
				if k < len(codes) {
					k++
				}
				code := codes[j:k]
				if code == "\x1b[0m" {
					active = active[:0]
				} else {
					active = append(active, code)
				}
				j = k
			} else {
				j++
			}
		}
	}

	var lines []string
	var cur strings.Builder
	col := 0

	restoreActive := func() {
		for _, c := range active {
			cur.WriteString(c)
		}
	}
	commitLine := func() {
		cur.WriteString("\x1b[0m")
		lines = append(lines, cur.String())
		cur.Reset()
		col = 0
		restoreActive()
	}

	for idx := 0; idx < len(toks); idx++ {
		t := toks[idx]
		parseActive(t.ansi)

		if t.r == ' ' {
			if col == 0 {
				continue // drop leading space on a wrapped line
			}
			// Look ahead to see if the next word fits on this line.
			nextW := 0
			for j := idx + 1; j < len(toks) && toks[j].r != ' '; j++ {
				nextW += toks[j].w
			}
			if col+1+nextW > maxW {
				commitLine()
				continue // don't emit the space
			}
			cur.WriteString(t.ansi)
			cur.WriteRune(t.r)
			col++
			continue
		}

		if col+t.w > maxW && col > 0 {
			// Hard-wrap (word wider than maxW, or missed a boundary).
			// active already includes t.ansi, so commitLine restores it.
			commitLine()
		} else {
			cur.WriteString(t.ansi)
		}
		cur.WriteRune(t.r)
		col += t.w
	}

	if cur.Len() > 0 || trailing != "" {
		cur.WriteString(trailing)
		cur.WriteString("\x1b[0m")
		lines = append(lines, cur.String())
	}
	if len(lines) == 0 {
		return []string{s}
	}
	return lines
}

// expandTabs replaces tab characters with spaces aligned to tabStop-column boundaries.
// runewidth treats '\t' as zero-width, so without expansion tabs cause lines to
// appear wider in the terminal than our width calculations expect.
func expandTabs(s string, tabStop int) string {
	if tabStop <= 0 {
		tabStop = 4
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			spaces := tabStop - (col % tabStop)
			b.WriteString(strings.Repeat(" ", spaces))
			col += spaces
		} else {
			b.WriteRune(r)
			col += runewidth.RuneWidth(r)
		}
	}
	return b.String()
}

// stripANSI removes ANSI escape sequences from a string for width calculation.
func stripANSI(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until 'm' or other terminator
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// sanitizeControlChars removes C0 control characters that could affect terminal rendering.
// Keeps printable characters and tab (which expandTabs has already handled).
func sanitizeControlChars(s string) string {
	if !strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 && r != '\t' }) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 0x20 || r == '\t' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// GitSectionStaged/Unstaged constants for use without importing state
const (
	GitSectionStaged   = state.GitSectionStaged
	GitSectionUnstaged = state.GitSectionUnstaged
)

// StripANSIExport is a public wrapper around stripANSI for use by other packages.
func StripANSIExport(s string) string {
	return stripANSI(s)
}

// truncateANSI truncates a string that may contain ANSI escape codes to maxW
// visible columns, preserving the escape codes and resetting at the end.
func truncateANSI(s string, maxW int) string {
	if maxW <= 0 {
		return "\x1b[0m"
	}
	var b strings.Builder
	col := 0
	i := 0
	hasEscape := false

	for i < len(s) {
		// ANSI CSI escape: ESC [
		if i+1 < len(s) && s[i] == '\x1b' && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			b.WriteString(s[i:j])
			hasEscape = true
			i = j
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			i++
			continue
		}
		rw := runewidth.RuneWidth(r)
		if col+rw > maxW {
			if col < maxW {
				b.WriteRune('…')
			}
			break
		}
		b.WriteRune(r)
		col += rw
		i += size
	}

	if hasEscape {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// expandTabsANSI expands tab characters in a string that may contain ANSI escape
// codes, correctly tracking visual column position while skipping escape sequences.
func expandTabsANSI(s string, tabStop int) string {
	if tabStop <= 0 {
		tabStop = 4
	}
	var b strings.Builder
	col := 0
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '\x1b' && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			b.WriteString(s[i:j])
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\t' {
			spaces := tabStop - (col % tabStop)
			b.WriteString(strings.Repeat(" ", spaces))
			col += spaces
		} else {
			b.WriteRune(r)
			col += runewidth.RuneWidth(r)
		}
		i += size
	}
	return b.String()
}
