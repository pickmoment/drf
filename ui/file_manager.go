package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FmOp constants (mirrored from app/model.go to avoid circular imports)
const (
	FmOpCopy   = 0
	FmOpMove   = 1
	FmOpRename = 2
	FmOpDelete = 3
	FmOpNewDir = 4
)

// FileManagerParams holds state for the file manager overlay.
type FileManagerParams struct {
	MenuIdx         int
	Input           string
	Cursor          int
	Operation       *int // nil = menu, non-nil = operation type
	Error           string
	OverwriteTarget string // non-empty = overwrite confirm dialog
	SourcePath      string
	TargetDir       string
	Width           int
	Height          int
}

// RenderFileManager renders the file manager overlay.
func RenderFileManager(p *FileManagerParams) string {
	if p.OverwriteTarget != "" {
		return renderOverwriteConfirm(p.OverwriteTarget, p.Width)
	}

	if p.Operation == nil {
		return renderFmMenu(p.MenuIdx, p.Width)
	}

	if p.Error != "" {
		return renderFmError(p.Error, p.Width)
	}

	return renderFmInput(*p.Operation, p.Input, p.Cursor, p.SourcePath, p.TargetDir, p.Width)
}

func renderFmMenu(menuIdx, w int) string {
	innerW := 40
	if innerW > w-4 {
		innerW = w - 4
	}

	items := []string{
		"📋 복사",
		"✂️  이동",
		"✏️  이름 변경",
		"🗑  삭제",
		"📁 새 폴더",
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("파일 관리"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")

	for i, item := range items {
		label := padRight(truncateStr(item, innerW), innerW)
		if i == menuIdx {
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

func renderFmInput(op int, input string, cursor int, srcPath, targetDir string, w int) string {
	innerW := 60
	if innerW > w-4 {
		innerW = w - 4
	}

	title := ""
	prompt := ""
	hint := ""
	switch op {
	case FmOpCopy:
		title = "복사"
		prompt = "복사 대상 경로:"
		hint = "현재: " + srcPath
	case FmOpMove:
		title = "이동"
		prompt = "이동 대상 경로:"
		hint = "현재: " + srcPath
	case FmOpRename:
		title = "이름 변경"
		prompt = "새 이름:"
		hint = "현재: " + srcPath
	case FmOpDelete:
		title = "삭제 확인"
		prompt = "삭제할까요? (y/n)"
		hint = srcPath
	case FmOpNewDir:
		title = "새 폴더"
		prompt = "폴더 이름:"
		hint = "위치: " + targetDir
	}

	// Build cursor display
	inputRunes := []rune(input)
	displayInput := ""
	if cursor <= len(inputRunes) {
		before := string(inputRunes[:cursor])
		after := ""
		if cursor < len(inputRunes) {
			after = string(inputRunes[cursor:])
		}
		displayInput = before + "█" + after
	} else {
		displayInput = input + "█"
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(title))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(truncateStr(hint, innerW)))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")
	sb.WriteString(prompt + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorSearchFg).Render(displayInput))
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter: 확인  Esc: 취소"))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorderFocused).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}

func renderFmError(errMsg string, w int) string {
	innerW := 55
	if innerW > w-4 {
		innerW = w - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("오류"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")
	sb.WriteString(truncateStr(errMsg, innerW))
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("Esc: 닫기  Enter: 계속"))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}

func renderOverwriteConfirm(target string, w int) string {
	innerW := 55
	if innerW > w-4 {
		innerW = w - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("파일 덮어쓰기"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", innerW))
	sb.WriteString("\n")
	sb.WriteString("이미 존재합니다: ")
	sb.WriteString(truncateStr(target, innerW-16))
	sb.WriteString("\n")
	sb.WriteString("덮어쓸까요?")
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("y: 덮어쓰기  n/Esc: 취소"))

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("220")).
		Padding(0, 1).
		Width(innerW + 2)
	return style.Render(sb.String())
}
