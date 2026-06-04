package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/pickmoment/drf/git"
	"github.com/pickmoment/drf/state"
)

// GitParams holds all state needed to render the Git view.
type GitParams struct {
	Width, Height int
	GitState      *state.GitState
	// Status message (forwarded from app model so it's visible in git mode).
	HasStatus  bool
	StatusText string
	StatusKind int
}

// RenderGit renders the full-screen git view.
func RenderGit(p *GitParams) string {
	if p.GitState == nil {
		return "Git 상태를 불러올 수 없습니다."
	}
	g := p.GitState
	w, h := p.Width, p.Height

	header := renderGitHeader(g, w)
	hintBar := renderGitHintBar(g, w)

	// Reserve lines: header(1) + hint(1) + optional status(1)
	reserved := 2
	var statusBar string
	if p.HasStatus {
		reserved++
		statusBar = RenderStatus(p.StatusText, p.StatusKind, w)
	}
	contentH := h - reserved
	if contentH < 1 {
		contentH = 1
	}

	var content string
	if g.DiffFullscreen {
		content = renderDiffFullscreen(g.Diff, g.CommitShow, g.DiffScroll, g.DiffHScroll, g.DiffWrap, w, contentH)
	} else if g.ShowLog {
		content = renderGitLogView(g, w, contentH)
	} else {
		content = renderGitFilePanels(g, w, contentH)
	}

	var parts []string
	parts = append(parts, header, content)
	if p.HasStatus {
		parts = append(parts, statusBar)
	}
	parts = append(parts, hintBar)
	result := strings.Join(parts, "\n")

	// Overlays
	if g.BranchPanelOpen {
		modal := renderBranchPanel(g.Branches, g.BranchIdx, g.BranchInputActive, g.BranchInput, w, h)
		result = PlaceOverlay(result, modal, w, h)
	}
	if g.IsCommitting {
		modal := renderCommitInputModal(g.CommitInput, w)
		result = PlaceOverlay(result, modal, w, h)
	}
	if g.Confirm != nil {
		modal := renderConfirmModal(g.Confirm, w)
		result = PlaceOverlay(result, modal, w, h)
	}
	if g.AsyncData != nil && g.AsyncCmd != nil {
		modal := renderAsyncProgressModal(g.AsyncData, g.SpinnerTick, w)
		result = PlaceOverlay(result, modal, w, h)
	}

	return result
}

func renderGitHeader(g *state.GitState, w int) string {
	bg := lipgloss.Color("235")
	bgStyle := lipgloss.NewStyle().Background(bg)

	if g.Status == nil {
		line := padRight(" Git: (상태 없음)", w)
		return bgStyle.Render(line)
	}
	s := g.Status

	// Brand label
	brand := lipgloss.NewStyle().
		Background(lipgloss.Color("27")).
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Render(" Git ")

	// Branch name
	branchText := " ⎇ " + s.Branch
	branchStyled := lipgloss.NewStyle().
		Background(bg).Foreground(lipgloss.Color("39")).Bold(true).
		Render(branchText)

	// Ahead / behind indicators
	var abParts []string
	if s.Ahead > 0 {
		abParts = append(abParts,
			lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("46")).
				Render(fmt.Sprintf(" ↑%d", s.Ahead)))
	}
	if s.Behind > 0 {
		abParts = append(abParts,
			lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("196")).
				Render(fmt.Sprintf(" ↓%d", s.Behind)))
	}
	ab := strings.Join(abParts, "")

	// Staged / unstaged counts
	stagedStyled := lipgloss.NewStyle().Background(bg).Foreground(ColorGitStagedFg).
		Render(fmt.Sprintf("  +%d staged", len(s.Staged)))
	unstagedStyled := lipgloss.NewStyle().Background(bg).Foreground(ColorGitUnstagedFg).
		Render(fmt.Sprintf("  ~%d unstaged", len(s.Unstaged)))

	// Remote URL (right-aligned hint)
	var remoteStyled string
	if s.RemoteURL != "" {
		remoteStyled = lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("240")).
			Render("  " + s.RemoteURL)
	}

	content := brand + branchStyled + ab + stagedStyled + unstagedStyled + remoteStyled

	// Truncate if the assembled content is wider than w (rare on narrow terminals).
	if lipgloss.Width(content) > w {
		plain := truncateStr(stripANSI(content), w)
		return lipgloss.NewStyle().Width(w).Background(bg).Render(plain)
	}

	// Use Width(w) so lipgloss handles the right-padding with the correct background.
	return lipgloss.NewStyle().Width(w).Background(bg).Render(content)
}

func renderGitHintBar(g *state.GitState, w int) string {
	var hints []hintPair
	if g.ShowLog {
		hints = []hintPair{
			{"j/k", "이동"}, {"Tab/l", "패널전환"}, {"h", "뒤로"}, {"Enter", "전체화면"}, {"q", "닫기"},
		}
	} else {
		hints = []hintPair{
			{"s/u", "스테이지"}, {"a/A", "전체"}, {"r", "복원"}, {"c", "커밋"},
			{"p", "푸시"}, {"F", "강제푸시"}, {"P", "풀"}, {"f", "페치"},
			{"L", "로그"}, {"b", "브랜치"}, {"Tab", "전환"}, {"q", "닫기"},
		}
	}
	return formatHintBar(hints, w)
}

func renderGitFilePanels(g *state.GitState, w, h int) string {
	leftW := w * 32 / 100
	if leftW < 20 {
		leftW = 20
	}
	rightW := w - leftW

	// Determine the currently selected file for the diff panel title.
	diffTitle := "diff"
	if g.Status != nil {
		switch g.Section {
		case state.GitSectionStaged:
			if g.StagedIdx < len(g.Status.Staged) {
				diffTitle = g.Status.Staged[g.StagedIdx].Path
			}
		case state.GitSectionUnstaged:
			if g.UnstagedIdx < len(g.Status.Unstaged) {
				diffTitle = g.Status.Unstaged[g.UnstagedIdx].Path
			}
		}
	}

	left := renderGitFileList(g, leftW, h)
	right := renderDiffPanel(g.Diff, g.DiffScroll, g.DiffHScroll, g.DiffWrap, diffTitle, rightW, h)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func renderGitFileList(g *state.GitState, w, h int) string {
	innerW := w - 2
	innerH := h - 2

	stagedCount := 0
	unstagedCount := 0
	if g.Status != nil {
		stagedCount = len(g.Status.Staged)
		unstagedCount = len(g.Status.Unstaged)
	}

	// Allocate height proportionally to content, with priority to the focused section.
	// Fixed overhead: 1 staged-header + 1 separator + 1 unstaged-header = 3 lines.
	overhead := 3
	available := innerH - overhead
	if available < 2 {
		available = 2
	}

	// Give each section at least 1 row, then distribute remaining rows proportionally.
	// Empty sections get minimal space (max 2 rows: header + "(없음)").
	var stagedH, unstagedH int
	if stagedCount == 0 && unstagedCount == 0 {
		stagedH = available / 2
		unstagedH = available - stagedH
	} else if stagedCount == 0 {
		// No staged files: give staged just 1 row (for "(없음)") + priority to unstaged.
		stagedH = 1
		unstagedH = available - 1
	} else if unstagedCount == 0 {
		unstagedH = 1
		stagedH = available - 1
	} else {
		// Both sections have content: distribute by count, with focused getting priority.
		total := stagedCount + unstagedCount
		if g.Section == state.GitSectionStaged {
			// Focused staged gets at least 60%.
			stagedH = available * 60 / 100
			if stagedCount*available/total > stagedH {
				stagedH = stagedCount * available / total
			}
		} else {
			// Focused unstaged gets at least 60%.
			unstagedH = available * 60 / 100
			if unstagedCount*available/total > unstagedH {
				unstagedH = unstagedCount * available / total
			}
		}
		if stagedH == 0 {
			stagedH = available - unstagedH
		}
		if unstagedH == 0 {
			unstagedH = available - stagedH
		}
	}
	if stagedH < 1 {
		stagedH = 1
	}
	if unstagedH < 1 {
		unstagedH = 1
	}

	var lines []string

	// ─── Staged section ──────────────────────────────────────────────────────
	stagedFocused := g.Section == state.GitSectionStaged
	stagedHeaderStyle := lipgloss.NewStyle().Foreground(ColorGitStagedFg).Bold(true)
	if stagedFocused {
		stagedHeaderStyle = stagedHeaderStyle.Underline(true)
	}
	stagedTitle := fmt.Sprintf("staged (%d)", stagedCount)
	lines = append(lines, padRight(stagedHeaderStyle.Render(stagedTitle), innerW))

	if g.Status != nil && len(g.Status.Staged) > 0 {
		// Virtual scroll: ensure selected item is visible.
		scroll := 0
		if stagedFocused && g.StagedIdx >= stagedH {
			scroll = g.StagedIdx - stagedH + 1
		}
		shown := 0
		for i := scroll; i < len(g.Status.Staged) && shown < stagedH; i++ {
			f := g.Status.Staged[i]
			label := gitStatusLabel(f.X, f.Y, true) + " " + f.Path
			label = padRight(truncateStr(label, innerW), innerW)
			if stagedFocused && i == g.StagedIdx {
				lines = append(lines, lipgloss.NewStyle().Background(ColorSelected).Render(label))
			} else {
				lines = append(lines, label)
			}
			shown++
		}
	} else {
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render(padRight("  (없음)", innerW)))
	}
	// Pad staged section to stagedH rows.
	for len(lines) < 1+stagedH {
		lines = append(lines, strings.Repeat(" ", innerW))
	}

	// Separator
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", innerW)))

	// ─── Unstaged section ────────────────────────────────────────────────────
	unstagedFocused := g.Section == state.GitSectionUnstaged
	unstagedHeaderStyle := lipgloss.NewStyle().Foreground(ColorGitUnstagedFg).Bold(true)
	if unstagedFocused {
		unstagedHeaderStyle = unstagedHeaderStyle.Underline(true)
	}
	unstagedTitle := fmt.Sprintf("unstaged (%d)", unstagedCount)
	lines = append(lines, padRight(unstagedHeaderStyle.Render(unstagedTitle), innerW))

	if g.Status != nil && len(g.Status.Unstaged) > 0 {
		scroll := 0
		if unstagedFocused && g.UnstagedIdx >= unstagedH {
			scroll = g.UnstagedIdx - unstagedH + 1
		}
		shown := 0
		for i := scroll; i < len(g.Status.Unstaged) && shown < unstagedH; i++ {
			f := g.Status.Unstaged[i]
			label := gitStatusLabel(f.X, f.Y, false) + " " + f.Path
			label = padRight(truncateStr(label, innerW), innerW)
			if unstagedFocused && i == g.UnstagedIdx {
				lines = append(lines, lipgloss.NewStyle().Background(ColorSelected).Render(label))
			} else {
				lines = append(lines, label)
			}
			shown++
		}
	} else {
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render(padRight("  (없음)", innerW)))
	}

	content := joinLines(lines, innerH, innerW)
	return renderBox("파일", content, w, h, ColorBorder)
}

func gitStatusLabel(x, y byte, staged bool) string {
	var ch byte
	if staged {
		ch = x
	} else {
		ch = y
	}
	switch ch {
	case 'M':
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("M")
	case 'A':
		return lipgloss.NewStyle().Foreground(ColorGitAddedFg).Render("A")
	case 'D':
		return lipgloss.NewStyle().Foreground(ColorGitDeletedFg).Render("D")
	case 'R':
		return lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("R")
	case 'C':
		return lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("C")
	case '?':
		return lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Render("?")
	default:
		return string(ch)
	}
}

func renderDiffPanel(diffLines []string, scroll, hScroll int, wrap bool, title string, w, h int) string {
	innerW := w - 2
	innerH := h - 2

	if title == "" {
		title = "diff"
	}

	// Append plain-text scroll indicator (no ANSI — renderBox title must stay plain).
	if len(diffLines) > 0 {
		total := len(diffLines)
		pct := (scroll + 1) * 100 / total
		title = title + fmt.Sprintf(" %d/%d (%d%%)", scroll+1, total, pct)
	}

	var lines []string
	for i := scroll; i < len(diffLines) && len(lines) < innerH; i++ {
		rawLine := diffLines[i]
		// Determine the diff prefix character BEFORE any transformation.
		var prefixChar byte
		if len(rawLine) > 0 {
			prefixChar = rawLine[0]
		}

		// Expand tabs in the plain content so width calculations match terminal rendering.
		plain := expandTabs(stripANSI(rawLine), 4)

		if !wrap && hScroll > 0 {
			runes := []rune(plain)
			if hScroll < len(runes) {
				plain = string(runes[hScroll:])
			} else {
				plain = ""
			}
		}

		displayLine := padRight(truncateStr(plain, innerW), innerW)

		// Apply syntax color based on the original first character.
		switch prefixChar {
		case '+':
			displayLine = lipgloss.NewStyle().Foreground(ColorGitAddedFg).Render(displayLine)
		case '-':
			displayLine = lipgloss.NewStyle().Foreground(ColorGitDeletedFg).Render(displayLine)
		case '@':
			displayLine = lipgloss.NewStyle().Foreground(ColorGitHunkFg).Render(displayLine)
		}
		lines = append(lines, displayLine)
	}

	content := joinLines(lines, innerH, innerW)
	return renderBox(title, content, w, h, ColorBorder)
}


func renderDiffFullscreen(diff, commitShow []string, scroll, hScroll int, wrap bool, w, h int) string {
	lines := diff
	if len(commitShow) > 0 {
		lines = commitShow
	}
	return renderTextFile(lines, scroll, hScroll, wrap, false, nil, -1, w, h)
}

func renderGitLogView(g *state.GitState, w, h int) string {
	// 3 columns: log | files | diff
	logW := w * 38 / 100
	if logW < 24 {
		logW = 24
	}
	filesW := w * 22 / 100
	if filesW < 18 {
		filesW = 18
	}
	diffW := w - logW - filesW

	// Build diff title from selected commit file.
	diffTitle := "diff"
	if g.LogIdx < len(g.Log) {
		diffTitle = g.Log[g.LogIdx].ShortHash
		if g.CommitFileIdx < len(g.CommitFiles) {
			diffTitle += ": " + g.CommitFiles[g.CommitFileIdx].Path
		}
	}

	logPanel := renderLogPanel(g.Log, g.LogIdx, g.LogFocused, logW, h)
	filesPanel := renderCommitFilesPanel(g.CommitFiles, g.CommitFileIdx, g.LogFileFocused, filesW, h)
	diffPanel := renderDiffPanel(g.CommitShow, g.CommitShowScroll, g.DiffHScroll, g.DiffWrap, diffTitle, diffW, h)

	return lipgloss.JoinHorizontal(lipgloss.Top, logPanel, filesPanel, diffPanel)
}

func renderLogPanel(entries []git.LogEntry, idx int, focused bool, w, h int) string {
	innerW := w - 2
	innerH := h - 2

	borderColor := ColorBorder
	if focused {
		borderColor = ColorBorderFocused
	}

	start := 0
	if idx >= innerH && innerH > 0 {
		start = idx - innerH + 1
	}

	var lines []string
	for i := start; i < len(entries) && len(lines) < innerH; i++ {
		e := entries[i]
		if i == idx {
			// Selected row: strip ANSI then apply selection background.
			plain := formatLogEntryPlain(e, innerW)
			plain = padRight(truncateStr(plain, innerW), innerW)
			lines = append(lines, lipgloss.NewStyle().Background(ColorSelected).Render(plain))
		} else {
			styled := formatLogEntry(e, innerW)
			// Ensure exact visual width.
			sw := lipgloss.Width(styled)
			if sw > innerW {
				styled = truncateANSI(styled, innerW)
			} else if sw < innerW {
				styled += strings.Repeat(" ", innerW-sw)
			}
			lines = append(lines, styled)
		}
	}

	content := joinLines(lines, innerH, innerW)
	return renderBox("커밋 로그", content, w, h, borderColor)
}

// formatLogEntry returns a coloured single-line entry for a log panel row.
// Layout: hash(7) space date(10) space author(pad14) space subject(rest)
func formatLogEntry(e git.LogEntry, innerW int) string {
	const hashW = 7
	const dateW = 10
	const authorW = 14
	const spacing = 1

	prefixW := hashW + spacing + dateW + spacing + authorW + spacing
	subjectW := innerW - prefixW
	if subjectW < 4 {
		subjectW = 4
	}

	// hash – always ASCII
	hash := e.ShortHash
	for len(hash) < hashW {
		hash += " "
	}
	if len(hash) > hashW {
		hash = hash[:hashW]
	}

	// date – always ASCII (YYYY-MM-DD)
	date := e.Date
	for len(date) < dateW {
		date += " "
	}
	if len(date) > dateW {
		date = date[:dateW]
	}

	// author – may contain CJK
	author := runewidth.Truncate(e.Author, authorW, "…")
	for runewidth.StringWidth(author) < authorW {
		author += " "
	}

	// subject – may contain CJK
	subject := runewidth.Truncate(e.Subject, subjectW, "…")

	hashStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("136")).Render(hash)
	dateStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(date)
	authorStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("109")).Render(author)

	return hashStyled + " " + dateStyled + " " + authorStyled + " " + subject
}

// formatLogEntryPlain returns a plain (no ANSI) version of a log entry line.
func formatLogEntryPlain(e git.LogEntry, innerW int) string {
	const hashW = 7
	const dateW = 10
	const authorW = 14

	hash := e.ShortHash
	for len(hash) < hashW {
		hash += " "
	}
	if len(hash) > hashW {
		hash = hash[:hashW]
	}

	date := e.Date
	for len(date) < dateW {
		date += " "
	}
	if len(date) > dateW {
		date = date[:dateW]
	}

	author := runewidth.Truncate(e.Author, authorW, "…")
	for runewidth.StringWidth(author) < authorW {
		author += " "
	}

	prefixW := hashW + 1 + dateW + 1 + authorW + 1
	subjectW := innerW - prefixW
	if subjectW < 4 {
		subjectW = 4
	}
	subject := runewidth.Truncate(e.Subject, subjectW, "…")

	return hash + " " + date + " " + author + " " + subject
}

func renderCommitFilesPanel(files []git.CommitFileEntry, idx int, focused bool, w, h int) string {
	innerW := w - 2
	innerH := h - 2

	borderColor := ColorBorder
	if focused {
		borderColor = ColorBorderFocused
	}

	var lines []string
	for i, f := range files {
		status := string(f.Status)
		label := padRight(truncateStr(status+" "+f.Path, innerW), innerW)
		if i == idx {
			lines = append(lines, lipgloss.NewStyle().Background(ColorSelected).Render(label))
		} else {
			lines = append(lines, label)
		}
	}
	if len(files) == 0 {
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render(padRight("(없음)", innerW)))
	}

	content := joinLines(lines, innerH, innerW)
	return renderBox("파일", content, w, h, borderColor)
}

// renderBranchPanel renders the branch selection overlay.
func renderBranchPanel(branches []git.BranchInfo, idx int, inputActive bool, input string, w, h int) string {
	innerW := 64
	if innerW > w-4 {
		innerW = w - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("브랜치"))
	sb.WriteString("  ")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
		fmt.Sprintf("(%d개)", len(branches))))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", innerW)))
	sb.WriteString("\n")

	maxItems := h - 10
	if maxItems < 5 {
		maxItems = 5
	}
	start := 0
	if idx >= maxItems {
		start = idx - maxItems + 1
	}

	for i := start; i < len(branches) && i < start+maxItems; i++ {
		b := branches[i]

		// Marker prefix
		marker := "  "
		if b.IsCurrent {
			marker = "* "
		} else if b.IsRemote {
			marker = "→ "
		}

		// Build the display label: marker + name + optional subject hint.
		name := b.Name
		plainBase := marker + name
		// Attempt to append a subject hint when there is room.
		var combined string
		if b.Subject != "" && i != idx {
			baseW := runewidth.StringWidth(plainBase)
			gap := 2
			subjectAvail := innerW - baseW - gap
			if subjectAvail > 4 {
				subj := runewidth.Truncate(b.Subject, subjectAvail, "…")
				combined = padRight(plainBase+"  "+subj, innerW)
			}
		}
		if combined == "" {
			combined = padRight(truncateStr(plainBase, innerW), innerW)
		}

		var styledLabel string
		if i == idx {
			plain := padRight(truncateStr(plainBase, innerW), innerW)
			styledLabel = lipgloss.NewStyle().Background(ColorSelected).Render(plain)
		} else if b.IsCurrent {
			styledLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).Render(combined)
		} else if b.IsRemote {
			styledLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(combined)
		} else {
			styledLabel = combined
		}

		sb.WriteString(styledLabel)
		sb.WriteString("\n")
	}
	if len(branches) == 0 {
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render("(없음)"))
		sb.WriteString("\n")
	}

	if inputActive {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", innerW)))
		sb.WriteString("\n")
		sb.WriteString("새 브랜치: ")
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorSearchFg).Render(input + "█"))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter:전환  n:새브랜치  d:삭제  D:강제삭제  Esc:닫기"))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorderFocused).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}

// renderCommitInputModal renders the commit message input modal.
func renderCommitInputModal(input string, w int) string {
	innerW := 60
	if innerW > w-4 {
		innerW = w - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("커밋 메시지 입력"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")
	sb.WriteString("> ")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorSearchFg).Render(input + "█"))
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter: 커밋  Esc: 취소"))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorderFocused).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}

// renderConfirmModal renders a yes/no confirmation modal.
func renderConfirmModal(confirm *state.ConfirmData, w int) string {
	innerW := 50
	if innerW > w-4 {
		innerW = w - 4
	}

	title := "확인"
	msg := ""
	switch confirm.Kind {
	case state.ConfirmDeleteBranchSoft:
		title = "브랜치 삭제"
		msg = "브랜치를 삭제할까요? " + confirm.Name
	case state.ConfirmDeleteBranchForce:
		title = "브랜치 강제 삭제"
		msg = "브랜치를 강제 삭제할까요? " + confirm.Name
	case state.ConfirmCheckoutFile:
		title = "파일 되돌리기"
		msg = "변경 사항을 되돌릴까요? " + confirm.Name
	case state.ConfirmForcePush:
		title = "강제 푸시"
		msg = "강제 푸시할까요? (--force-with-lease)"
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render(title))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")
	sb.WriteString(truncateStr(msg, innerW))
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("y: 확인  n/Esc: 취소"))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}

// renderAsyncProgressModal renders an in-progress modal for async git operations.
func renderAsyncProgressModal(data *state.AsyncData, tick int, w int) string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spin := spinners[tick%len(spinners)]

	opName := ""
	switch data.Kind {
	case state.AsyncPush:
		opName = "푸시"
	case state.AsyncPull:
		opName = "풀"
	case state.AsyncFetch:
		opName = "페치"
	}
	if data.Force {
		opName += " (강제)"
	}

	innerW := 40
	if innerW > w-4 {
		innerW = w - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Git " + opName + " 중..."))
	sb.WriteString("\n\n")
	sb.WriteString("  ")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorderFocused).Render(spin))
	sb.WriteString("  진행 중입니다...")

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorderFocused).
		Padding(1, 2).
		Width(innerW + 4)
	return style.Render(sb.String())
}
