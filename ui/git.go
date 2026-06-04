package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/pickmoment/drf/git"
	"github.com/pickmoment/drf/state"
)

// GitParams holds all state needed to render the Git view.
type GitParams struct {
	Width, Height int
	GitState      *state.GitState
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
	contentH := h - 2 // header + hint

	var content string
	if g.DiffFullscreen {
		content = renderDiffFullscreen(g.Diff, g.CommitShow, g.DiffScroll, g.DiffHScroll, g.DiffWrap, w, contentH)
	} else if g.ShowLog {
		content = renderGitLogView(g, w, contentH)
	} else {
		content = renderGitFilePanels(g, w, contentH)
	}

	result := header + "\n" + content + "\n" + hintBar

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
	if g.Status == nil {
		return padRight("Git: (상태 없음)", w)
	}
	s := g.Status
	branch := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true).Render(s.Branch)
	remote := ""
	if s.RemoteURL != "" {
		remote = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Render(s.RemoteURL)
	}
	ab := ""
	if s.Ahead > 0 || s.Behind > 0 {
		ab = fmt.Sprintf(" ↑%d↓%d", s.Ahead, s.Behind)
	}
	counts := fmt.Sprintf("  staged:%d  unstaged:%d", len(s.Staged), len(s.Unstaged))

	header := " Git: " + branch + remote + ab + counts
	style := lipgloss.NewStyle().Width(w).Background(lipgloss.Color("235"))
	return style.Render(truncateStr(header, w))
}

func renderGitHintBar(g *state.GitState, w int) string {
	var hints []hintPair
	if g.ShowLog {
		hints = []hintPair{
			{"j/k", "이동"}, {"l", "파일목록"}, {"h", "뒤로"}, {"q", "닫기"},
		}
	} else {
		hints = []hintPair{
			{"s/u", "스테이지"}, {"c", "커밋"}, {"p", "푸시"},
			{"P", "풀"}, {"f", "페치"}, {"L", "로그"},
			{"b", "브랜치"}, {"Tab", "전환"}, {"q", "닫기"},
		}
	}
	return formatHintBar(hints, w)
}

func renderGitFilePanels(g *state.GitState, w, h int) string {
	leftW := w * 35 / 100
	if leftW < 20 {
		leftW = 20
	}
	rightW := w - leftW

	left := renderGitFileList(g, leftW, h)
	right := renderDiffPanel(g.Diff, g.DiffScroll, g.DiffHScroll, g.DiffWrap, rightW, h)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func renderGitFileList(g *state.GitState, w, h int) string {
	innerW := w - 2
	innerH := h - 2

	var lines []string

	// Staged section
	stagedHeader := lipgloss.NewStyle().Foreground(ColorGitStagedFg).Bold(true).Render("스테이지됨")
	lines = append(lines, padRight(stagedHeader, innerW))

	if g.Status != nil {
		for i, f := range g.Status.Staged {
			label := gitStatusLabel(f.X, f.Y, true) + " " + f.Path
			label = padRight(truncateStr(label, innerW), innerW)
			if g.Section == state.GitSectionStaged && i == g.StagedIdx {
				lines = append(lines, lipgloss.NewStyle().Background(ColorSelected).Render(label))
			} else {
				lines = append(lines, label)
			}
		}
		if len(g.Status.Staged) == 0 {
			lines = append(lines, lipgloss.NewStyle().Faint(true).Render(padRight("  (없음)", innerW)))
		}
	}

	lines = append(lines, "")

	// Unstaged section
	unstagedHeader := lipgloss.NewStyle().Foreground(ColorGitUnstagedFg).Bold(true).Render("변경됨 (스테이지 안됨)")
	lines = append(lines, padRight(unstagedHeader, innerW))

	if g.Status != nil {
		for i, f := range g.Status.Unstaged {
			label := gitStatusLabel(f.X, f.Y, false) + " " + f.Path
			label = padRight(truncateStr(label, innerW), innerW)
			if g.Section == state.GitSectionUnstaged && i == g.UnstagedIdx {
				lines = append(lines, lipgloss.NewStyle().Background(ColorSelected).Render(label))
			} else {
				lines = append(lines, label)
			}
		}
		if len(g.Status.Unstaged) == 0 {
			lines = append(lines, lipgloss.NewStyle().Faint(true).Render(padRight("  (없음)", innerW)))
		}
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

func renderDiffPanel(diffLines []string, scroll, hScroll int, wrap bool, w, h int) string {
	innerW := w - 2
	innerH := h - 2

	var lines []string
	for i := scroll; i < len(diffLines) && len(lines) < innerH; i++ {
		line := diffLines[i]
		colored := colorDiffLine(line)
		plain := stripANSI(colored)
		if !wrap {
			runes := []rune(plain)
			if hScroll < len(runes) {
				plain = string(runes[hScroll:])
			} else {
				plain = ""
			}
			colored = colorDiffLine(plain)
		}
		line = padRight(truncateStr(stripANSI(colored), innerW), innerW)
		// Re-apply color
		if len(diffLines[i]) > 0 {
			switch diffLines[i][0] {
			case '+':
				line = lipgloss.NewStyle().Foreground(ColorGitAddedFg).Render(line)
			case '-':
				line = lipgloss.NewStyle().Foreground(ColorGitDeletedFg).Render(line)
			case '@':
				line = lipgloss.NewStyle().Foreground(ColorGitHunkFg).Render(line)
			}
		}
		lines = append(lines, line)
	}

	content := joinLines(lines, innerH, innerW)
	return renderBox("diff", content, w, h, ColorBorder)
}

func colorDiffLine(line string) string {
	if len(line) == 0 {
		return line
	}
	switch line[0] {
	case '+':
		return lipgloss.NewStyle().Foreground(ColorGitAddedFg).Render(line)
	case '-':
		return lipgloss.NewStyle().Foreground(ColorGitDeletedFg).Render(line)
	case '@':
		return lipgloss.NewStyle().Foreground(ColorGitHunkFg).Render(line)
	}
	return line
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
	logW := w * 30 / 100
	if logW < 20 {
		logW = 20
	}
	filesW := w * 25 / 100
	if filesW < 18 {
		filesW = 18
	}
	diffW := w - logW - filesW

	logPanel := renderLogPanel(g.Log, g.LogIdx, g.LogFocused, logW, h)
	filesPanel := renderCommitFilesPanel(g.CommitFiles, g.CommitFileIdx, g.LogFileFocused, filesW, h)
	diffPanel := renderDiffPanel(g.CommitShow, g.CommitShowScroll, g.DiffHScroll, g.DiffWrap, diffW, h)

	return lipgloss.JoinHorizontal(lipgloss.Top, logPanel, filesPanel, diffPanel)
}

func renderLogPanel(logLines []string, idx int, focused bool, w, h int) string {
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
	for i := start; i < len(logLines) && len(lines) < innerH; i++ {
		label := padRight(truncateStr(logLines[i], innerW), innerW)
		if i == idx {
			lines = append(lines, lipgloss.NewStyle().Background(ColorSelected).Render(label))
		} else {
			lines = append(lines, label)
		}
	}

	content := joinLines(lines, innerH, innerW)
	return renderBox("커밋 로그", content, w, h, borderColor)
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
	innerW := 60
	if innerW > w-4 {
		innerW = w - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("브랜치 목록"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
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
		curr := "  "
		if b.IsCurrent {
			curr = "* "
		}
		label := curr + b.Name
		if b.Subject != "" {
			label += " - " + b.Subject
		}
		label = padRight(truncateStr(label, innerW), innerW)
		if i == idx {
			sb.WriteString(lipgloss.NewStyle().Background(ColorSelected).Render(label))
		} else {
			sb.WriteString(label)
		}
		sb.WriteString("\n")
	}
	if len(branches) == 0 {
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render("(없음)"))
		sb.WriteString("\n")
	}

	if inputActive {
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", innerW))
		sb.WriteString("\n")
		sb.WriteString("새 브랜치: ")
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorSearchFg).Render(input + "█"))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter:전환  n:새브랜치  d:삭제  Esc:닫기"))

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
