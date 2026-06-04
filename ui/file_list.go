package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/pickmoment/drf/fs"
)

// FileListParams holds all data for rendering the file list panel.
type FileListParams struct {
	Entries         []fs.FileEntry
	FilteredIndices []int
	SelectedIndex   int
	CurrentDir      string
	IsSearching     bool
	SearchQuery     string
	Width           int
	Height          int
	Focused         bool
	ShowIcons       bool
	GitFileMap      map[string][2]byte
	GitRoot         string
}

// RenderFileList renders the file list panel as a bordered box.
func RenderFileList(p *FileListParams) string {
	borderColor := lipgloss.Color("240")
	if p.Focused {
		borderColor = lipgloss.Color("39")
	}

	innerW := p.Width - 2
	if innerW < 2 {
		innerW = 2
	}

	// Reserve 2 columns for the cursor indicator ("▶ " or "  ")
	const cursorW = 2
	contentW := innerW - cursorW
	if contentW < 1 {
		contentW = 1
	}

	// Reserve 1 line for search bar if searching
	listH := p.Height - 2
	if listH < 1 {
		listH = 1
	}
	if p.IsSearching {
		listH--
	}

	// Compute visible window
	start := 0
	if p.SelectedIndex >= listH && listH > 0 {
		start = p.SelectedIndex - listH + 1
	}

	selStyle := lipgloss.NewStyle().Background(ColorSelected)
	cursorStyle := lipgloss.NewStyle().Foreground(ColorBorderFocused).Bold(true)

	var lines []string
	for i := start; i < len(p.FilteredIndices) && len(lines) < listH; i++ {
		idx := p.FilteredIndices[i]
		if idx >= len(p.Entries) {
			continue
		}
		entry := p.Entries[idx]
		line := formatFileEntry(entry, contentW, p.ShowIcons, p.GitFileMap, p.GitRoot)
		line = padRight(line, contentW)
		content := truncateToWidth(line, contentW)
		if i == p.SelectedIndex {
			cursor := cursorStyle.Render("▶ ")
			lines = append(lines, cursor+selStyle.Render(content))
		} else {
			lines = append(lines, "  "+content)
		}
	}

	// Pad remaining lines
	for len(lines) < listH {
		lines = append(lines, strings.Repeat(" ", innerW))
	}

	// Search bar
	if p.IsSearching {
		bar := "/" + p.SearchQuery
		bar = padRight(truncateStr(bar, innerW), innerW)
		searchStyle := lipgloss.NewStyle().Foreground(ColorSearchFg)
		lines = append(lines, searchStyle.Render(bar))
	}

	// Build title
	title := fmt.Sprintf("파일 목록 (%d)", len(p.FilteredIndices))

	content := strings.Join(lines, "\n")
	return renderBox(title, content, p.Width, p.Height, borderColor)
}

// formatFileEntry formats a single file entry line for the list.
// Layout: [icon(2)][name ............ ][dim size(6)]
//
// Nerd Font icons are assumed to occupy exactly 2 display columns each
// (either a 2-wide glyph, or a 1-wide glyph + 1 trailing space).
// Using a fixed iconDisplayW avoids runewidth mismatches with wide glyphs.
func formatFileEntry(entry fs.FileEntry, width int, showIcons bool, gitMap map[string][2]byte, gitRoot string) string {
	// Icon — fixed 2-column display budget regardless of glyph width.
	icon := ""
	const iconDisplayW = 2
	if showIcons {
		if entry.IsDir {
			icon = " " // folder glyph is 2-wide; no trailing space needed
		} else {
			icon = fileIcon(entry.Name) + " " // 1-wide glyph + space
		}
	}

	// Size area: " 9.7M " — 6 cols reserved at the right end
	const sizeW = 6
	var rawSize string
	if !entry.IsDir {
		rawSize = fmt.Sprintf(" %4s ", fs.FormatSize(entry.Size))
	} else {
		rawSize = "      "
	}
	dimSize := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(rawSize)

	// Name gets remaining width
	iconW := 0
	if showIcons {
		iconW = iconDisplayW
	}
	nameAreaW := width - iconW - sizeW
	if nameAreaW < 1 {
		nameAreaW = 1
	}
	truncatedName := runewidth.Truncate(entry.Name, nameAreaW, "…")

	var styledName string
	switch {
	case entry.IsDir:
		styledName = lipgloss.NewStyle().Foreground(ColorDirFg).Render(truncatedName)
	case entry.IsHidden:
		styledName = lipgloss.NewStyle().Faint(true).Render(truncatedName)
	default:
		styledName = truncatedName
	}

	// Pad name to fill nameAreaW so size is flush-right
	nameActualW := lipgloss.Width(styledName)
	if nameActualW < nameAreaW {
		styledName += strings.Repeat(" ", nameAreaW-nameActualW)
	}

	return icon + styledName + dimSize
}

func gitMarker(x, y byte) string {
	staged := x != ' ' && x != '?'
	unstaged := y != ' ' || (x == '?' && y == '?')
	if staged && unstaged {
		return "\x1b[33m±\x1b[0m "
	} else if staged {
		return "\x1b[32m+\x1b[0m "
	} else if unstaged {
		return "\x1b[31m!\x1b[0m "
	}
	return "  "
}

func fileIcon(name string) string {
	lower := strings.ToLower(name)
	ext := ""
	if i := strings.LastIndex(lower, "."); i >= 0 {
		ext = lower[i+1:]
	}
	switch ext {
	case "go":
		return ""
	case "rs":
		return ""
	case "py":
		return ""
	case "js", "mjs", "cjs":
		return ""
	case "ts", "tsx":
		return "󰛦"
	case "html", "htm":
		return ""
	case "css":
		return ""
	case "json":
		return ""
	case "yaml", "yml":
		return ""
	case "toml":
		return ""
	case "md", "markdown":
		return ""
	case "txt":
		return ""
	case "sh", "bash", "zsh", "fish":
		return ""
	case "png", "jpg", "jpeg", "gif", "webp", "svg":
		return ""
	case "pdf":
		return ""
	case "zip", "tar", "gz", "bz2", "xz", "7z", "rar":
		return ""
	case "csv", "tsv":
		return "󰙩"
	case "parquet":
		return "󰙩"
	case "mp3", "wav", "flac", "ogg":
		return ""
	case "mp4", "mkv", "avi", "mov":
		return "󰕧"
	default:
		return ""
	}
}

func relPath(root, path string) string {
	if strings.HasPrefix(path, root+"/") {
		return path[len(root)+1:]
	}
	return path
}

// truncateToWidth truncates s to at most maxW visual columns (stripping ANSI for measurement).
func truncateToWidth(s string, maxW int) string {
	visible := stripANSI(s)
	if lipgloss.Width(visible) <= maxW {
		return s
	}
	return truncateStr(visible, maxW)
}
