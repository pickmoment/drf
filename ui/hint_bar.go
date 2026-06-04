package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AppMode constants (mirrored from app, no import needed)
const (
	AppModeFileList      = 0
	AppModeViewer        = 1
	AppModeOpenWith      = 2
	AppModeSettings      = 3
	AppModeCommandPalette = 4
	AppModeHelp          = 5
	AppModeGit           = 6
	AppModeFileManager   = 7
	AppModePathClipboard = 8
)

type hintPair struct {
	key  string
	desc string
}

// RenderHintBar renders context-sensitive key hints at the bottom of the screen.
func RenderHintBar(mode, focusedPanel int, isSearching, isGitLog bool, w int) string {
	hints := getHints(mode, focusedPanel, isSearching, isGitLog)
	return formatHintBar(hints, w)
}

func getHints(mode, focusedPanel int, isSearching, isGitLog bool) []hintPair {
	switch mode {
	case AppModeViewer:
		if isSearching {
			return []hintPair{
				{"Enter", "다음"}, {"Shift+Enter", "이전"}, {"Esc", "검색취소"},
			}
		}
		return []hintPair{
			{"q", "닫기"}, {"/", "검색"}, {"g", "상단"}, {"G", "하단"},
			{"W", "줄바꿈"}, {"j/k", "스크롤"},
		}
	case AppModeGit:
		if isGitLog {
			return []hintPair{
				{"j/k", "이동"}, {"l", "파일목록"}, {"h", "뒤로"},
				{"q", "종료"},
			}
		}
		return []hintPair{
			{"s/u", "스테이지"}, {"c", "커밋"}, {"p", "푸시"},
			{"P", "풀"}, {"f", "페치"}, {"L", "로그"}, {"b", "브랜치"},
			{"q", "종료"},
		}
	case AppModeFileManager:
		return []hintPair{
			{"Enter", "실행"}, {"Esc", "취소"},
		}
	case AppModePathClipboard:
		return []hintPair{
			{"Enter", "이동"}, {"y", "복사"}, {"d", "삭제"}, {"Esc", "닫기"},
		}
	case AppModeOpenWith:
		return []hintPair{
			{"Enter", "열기"}, {"Esc", "취소"},
		}
	case AppModeHelp:
		return []hintPair{
			{"q/Esc", "닫기"},
		}
	case AppModeCommandPalette:
		return []hintPair{
			{"Enter", "실행"}, {"Esc", "취소"}, {"↑↓", "이동"},
		}
	default: // FileList
		if isSearching {
			return []hintPair{
				{"Enter", "확정"}, {"Esc", "취소"},
			}
		}
		switch focusedPanel {
		case 1: // Bookmarks
			return []hintPair{
				{"Enter", "이동"}, {"d", "삭제"}, {"Tab", "파일목록"}, {"q", "종료"},
			}
		default:
			return []hintPair{
				{"j/k", "이동"}, {"Enter", "열기"}, {"/", "검색"},
				{"y", "경로복사"}, {"b", "즐겨찾기"}, {"g", "Git"}, {"?", "도움말"}, {"Q", "종료"},
			}
		}
	}
}

func formatHintBar(hints []hintPair, w int) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorHintKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ColorHintDesc)
	bracketStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("235"))

	var rendered []string
	for i, h := range hints {
		piece := bracketStyle.Render("[") + keyStyle.Render(h.key) + bracketStyle.Render("]") + " " + descStyle.Render(h.desc)
		if i > 0 {
			rendered = append(rendered, sepStyle.Render("  ")+piece)
		} else {
			rendered = append(rendered, piece)
		}
	}
	result := strings.Join(rendered, "")
	// Truncate to fit
	plainW := lipgloss.Width(result)
	if plainW > w {
		result = truncateStr(stripANSI(result), w)
	}
	// Pad to width
	plainW = lipgloss.Width(result)
	if plainW < w {
		result += strings.Repeat(" ", w-plainW)
	}
	return bgStyle.Render(result)
}
