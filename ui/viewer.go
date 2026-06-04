package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/pickmoment/drf/fs"
	"github.com/pickmoment/drf/preview"
)

// ViewerParams holds state for the fullscreen viewer.
type ViewerParams struct {
	FilePath        string
	Lines           []string // pre-rendered lines (with ANSI)
	Scroll          int
	HScroll         int
	Wrap            bool
	LineNumbers     bool
	Width           int
	Height          int
	IsSearching     bool
	SearchQuery     string
	SearchMatches   []int
	SearchIdx       int
	IsGoto          bool
	GotoInput       string
	ViewerHeightOut *int
}

// RenderFullscreen renders the fullscreen viewer.
func RenderFullscreen(p *ViewerParams) string {
	toolbarH := 1
	statusH := 1
	searchBarH := 0
	if p.IsSearching || p.IsGoto {
		searchBarH = 1
	}
	contentH := p.Height - toolbarH - statusH - searchBarH
	if contentH < 1 {
		contentH = 1
	}
	if p.ViewerHeightOut != nil {
		*p.ViewerHeightOut = contentH
	}

	toolbar := renderViewerToolbar(p.FilePath, p.Wrap, p.LineNumbers, p.Width)
	content := renderTextFile(p.Lines, p.Scroll, p.HScroll, p.Wrap, p.LineNumbers, p.SearchMatches, p.SearchIdx, p.Width, contentH)
	status := renderViewerStatus(p.Scroll, len(p.Lines), p.Width)

	parts := []string{toolbar, content}
	if p.IsGoto {
		parts = append(parts, lipgloss.NewStyle().Foreground(ColorSearchFg).Render(
			padRight(":"+p.GotoInput, p.Width)))
	} else if p.IsSearching {
		parts = append(parts, lipgloss.NewStyle().Foreground(ColorSearchFg).Render(
			padRight("/"+p.SearchQuery, p.Width)))
	}
	parts = append(parts, status)
	return strings.Join(parts, "\n")
}

// renderViewerToolbar renders the top toolbar with filename and key hints.
func renderViewerToolbar(path string, wrap, lineNumbers bool, w int) string {
	name := filepath.Base(path)
	wrapIndicator := ""
	if wrap {
		wrapIndicator = " [줄바꿈]"
	}
	lnIndicator := ""
	if lineNumbers {
		lnIndicator = " [줄번호]"
	}
	left := " " + name + wrapIndicator + lnIndicator
	right := " q:닫기  /:검색  W:줄바꿈  L:줄번호 "
	rightW := lipgloss.Width(right)
	leftW := w - rightW
	if leftW < 4 {
		leftW = 4
	}
	left = truncateStr(left, leftW)
	left = padRight(left, leftW)

	style := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("252"))
	return style.Render(left + right)
}

// renderViewerStatus renders the bottom status line.
func renderViewerStatus(scroll, totalLines, w int) string {
	pct := 0
	if totalLines > 0 {
		pct = (scroll + 1) * 100 / totalLines
	}
	msg := fmt.Sprintf(" %d/%d (%d%%)", scroll+1, totalLines, pct)
	msg = padRight(msg, w)
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("246"))
	return style.Render(msg)
}

// renderTextFile renders source lines with optional search highlighting and line numbers.
func renderTextFile(lines []string, scroll, hScroll int, wrap, lineNumbers bool, matches []int, searchIdx int, w, h int) string {
	matchSet := map[int]bool{}
	for _, m := range matches {
		matchSet[m] = true
	}
	currentMatch := -1
	if searchIdx >= 0 && searchIdx < len(matches) {
		currentMatch = matches[searchIdx]
	}

	// Line number gutter: digits + " │"
	lnW := 0
	if lineNumbers && len(lines) > 0 {
		lnW = len(fmt.Sprintf("%d", len(lines))) + 2
	}
	contentW := w - lnW
	if contentW < 1 {
		contentW = 1
	}
	lnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	lnSepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

	renderGutter := func(srcLine int, isContinuation bool) string {
		if !lineNumbers {
			return ""
		}
		if isContinuation {
			return strings.Repeat(" ", lnW-1) + lnSepStyle.Render("│")
		}
		numStr := fmt.Sprintf("%*d", lnW-2, srcLine+1)
		return lnStyle.Render(numStr) + lnSepStyle.Render("│")
	}

	var visible []string
	lineNum := scroll
	for lineNum < len(lines) && len(visible) < h {
		rawLine := lines[lineNum]
		plain := sanitizeControlChars(expandTabs(stripANSI(rawLine), 4))
		isMatch := matchSet[lineNum]

		isANSI := strings.Contains(rawLine, "\x1b")

		if wrap {
			expanded := expandTabsANSI(rawLine, 4)
			// Don't word-wrap box-drawing table lines.
			if startsWithBoxDrawing(plain) {
				wl := padRight(truncateANSI(expanded, contentW), contentW)
				if isMatch {
					wl = renderHighlightedLine(stripANSI(wl), currentMatch == lineNum)
				}
				visible = append(visible, renderGutter(lineNum, false)+wl)
			} else {
				wrapped := wrapLineANSI(expanded, contentW)
				for subIdx, wl := range wrapped {
					if len(visible) >= h {
						break
					}
					wl = padRight(truncateANSI(wl, contentW), contentW)
					if isMatch {
						wl = renderHighlightedLine(stripANSI(wl), currentMatch == lineNum)
					}
					visible = append(visible, renderGutter(lineNum, subIdx > 0)+wl)
				}
			}
			lineNum++
			continue
		}

		var line string
		if hScroll > 0 {
			if isANSI {
				expanded := expandTabsANSI(rawLine, 4)
				line = padRight(truncateANSI(hScrollANSI(expanded, hScroll), contentW), contentW)
			} else {
				runes := []rune(plain)
				if hScroll < len(runes) {
					plain = string(runes[hScroll:])
				} else {
					plain = ""
				}
				line = padRight(truncateStr(plain, contentW), contentW)
			}
			if isMatch {
				line = renderHighlightedLine(line, currentMatch == lineNum)
			}
		} else if isMatch {
			line = renderHighlightedLine(padRight(truncateStr(plain, contentW), contentW), currentMatch == lineNum)
		} else if isANSI {
			expanded := expandTabsANSI(rawLine, 4)
			line = padRight(truncateANSI(expanded, contentW), contentW)
		} else {
			line = padRight(truncateStr(plain, contentW), contentW)
		}

		visible = append(visible, renderGutter(lineNum, false)+line)
		lineNum++
	}

	// Pad remaining
	emptyLine := strings.Repeat(" ", w)
	for len(visible) < h {
		visible = append(visible, emptyLine)
	}

	return strings.Join(visible, "\n")
}

func renderHighlightedLine(line string, isCurrent bool) string {
	if isCurrent {
		return lipgloss.NewStyle().Background(lipgloss.Color("220")).Foreground(lipgloss.Color("0")).Render(line)
	}
	return lipgloss.NewStyle().Background(lipgloss.Color("58")).Render(line)
}

// RenderPreviewContent renders file content based on file type.
// fileType: 0=text, 1=markdown, 2=code, 5=csv, 8=unknown (previewable types only).
// Non-previewable types (3=image, 4=pdf, 6=parquet, 7=archive, 9=binary) should be
// handled via RenderFileInfo before calling this function.
// maxWidth: maximum content width for wrap-aware rendering (0 = unlimited).
func RenderPreviewContent(path, rawContent string, fileType int, codeLang, syntaxTheme string, markdownRender bool, maxWidth int) []string {
	switch fileType {
	case 1: // markdown
		return preview.RenderMarkdown(rawContent, maxWidth)
	case 2: // code
		return preview.HighlightCode(rawContent, codeLang, syntaxTheme)
	case 5: // csv
		return preview.RenderCSV(rawContent)
	default:
		// text or unknown: plain lines
		raw := strings.TrimRight(rawContent, "\n")
		return strings.Split(raw, "\n")
	}
}

// RenderFileInfo renders a rich file information panel for non-previewable files.
// fileType: 3=image, 4=pdf, 6=parquet, 7=archive, 9=binary
func RenderFileInfo(entry fs.FileEntry, fileType int) []string {
	// Icon and type label (safe ASCII variants to avoid CJK width issues)
	typeIcon := "[ ]"
	typeLabel := "파일"
	switch fileType {
	case 9: // binary
		typeIcon = "[bin]"
		typeLabel = "바이너리 / 실행 파일"
	case 3: // image
		typeIcon = "[img]"
		typeLabel = "이미지 파일"
	case 4: // pdf
		typeIcon = "[pdf]"
		typeLabel = "PDF 문서"
	case 7: // archive
		typeIcon = "[zip]"
		typeLabel = "압축 파일"
	case 6: // parquet
		typeIcon = "[pqt]"
		typeLabel = "Parquet 데이터 파일"
	}

	// Stat for permissions
	perms := ""
	modTime := entry.Modified.Format("2006-01-02 15:04")
	if info, err := os.Stat(entry.Path); err == nil {
		perms = info.Mode().String()
	}

	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	dimStyle := lipgloss.NewStyle().Faint(true)
	divStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

	div := divStyle.Render(strings.Repeat("─", 36))

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+div)
	lines = append(lines, "")
	lines = append(lines, "  "+iconStyle.Render(typeIcon)+"  "+nameStyle.Render(entry.Name))
	lines = append(lines, "  "+typeStyle.Render(typeLabel))
	lines = append(lines, "")
	lines = append(lines, "  "+labelStyle.Render("크기    ")+"  "+valueStyle.Render(fs.FormatSize(entry.Size)))
	lines = append(lines, "  "+labelStyle.Render("수정일  ")+"  "+valueStyle.Render(modTime))
	if perms != "" {
		lines = append(lines, "  "+labelStyle.Render("권한    ")+"  "+dimStyle.Render(perms))
	}
	lines = append(lines, "")
	lines = append(lines, "  "+div)
	lines = append(lines, "")
	lines = append(lines, "  "+dimStyle.Render("이 파일은 텍스트로 미리보기할 수 없습니다."))
	lines = append(lines, "  "+dimStyle.Render("[o] 시스템 앱으로 열기  [e] 편집기로 열기"))

	return lines
}
