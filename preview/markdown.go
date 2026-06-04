package preview

import (
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"
)

// RenderMarkdown converts a markdown string into a slice of display lines
// with ANSI styling for terminals, including syntax-highlighted code blocks.
// maxWidth constrains table column widths for wrap-aware rendering (0 = unlimited).
func RenderMarkdown(src string, maxWidth int) []string {
	lines := strings.Split(src, "\n")
	var result []string
	inCodeBlock := false
	codeLang := ""
	var codeLines []string
	var tableBuffer []string

	flushTable := func() {
		if len(tableBuffer) > 0 {
			result = append(result, renderTable(tableBuffer, maxWidth)...)
			tableBuffer = nil
		}
	}

	isTableLine := func(s string) bool {
		return strings.HasPrefix(s, "|") && strings.HasSuffix(s, "|")
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Code block fence
		if strings.HasPrefix(line, "```") {
			flushTable()
			if !inCodeBlock {
				inCodeBlock = true
				codeLang = strings.TrimSpace(strings.TrimPrefix(line, "```"))
				codeLines = nil
				result = append(result, "\x1b[38;5;237m"+line+"\x1b[0m")
			} else {
				inCodeBlock = false
				// Syntax-highlight collected block if language is known
				if codeLang != "" && len(codeLines) > 0 {
					highlighted := HighlightCode(strings.Join(codeLines, "\n"), codeLang, "monokai")
					result = append(result, highlighted...)
				} else {
					for _, cl := range codeLines {
						result = append(result, "\x1b[2m"+cl+"\x1b[0m")
					}
				}
				result = append(result, "\x1b[38;5;237m"+line+"\x1b[0m")
				codeLines = nil
				codeLang = ""
			}
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		// Table: buffer consecutive pipe-delimited rows, flush when table ends
		if isTableLine(line) {
			tableBuffer = append(tableBuffer, line)
			continue
		}
		flushTable()

		// Horizontal rule
		if line == "---" || line == "***" || line == "___" {
			result = append(result, "\x1b[2m"+strings.Repeat("─", 40)+"\x1b[0m")
			continue
		}

		// Headings
		if strings.HasPrefix(line, "###### ") {
			result = append(result, "\x1b[1;35m"+strings.TrimPrefix(line, "###### ")+"\x1b[0m")
			continue
		}
		if strings.HasPrefix(line, "##### ") {
			result = append(result, "\x1b[1;35m"+strings.TrimPrefix(line, "##### ")+"\x1b[0m")
			continue
		}
		if strings.HasPrefix(line, "#### ") {
			result = append(result, "\x1b[1;34m"+strings.TrimPrefix(line, "#### ")+"\x1b[0m")
			continue
		}
		if strings.HasPrefix(line, "### ") {
			result = append(result, "\x1b[1;34m"+strings.TrimPrefix(line, "### ")+"\x1b[0m")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			result = append(result, "\x1b[1;33m"+strings.TrimPrefix(line, "## ")+"\x1b[0m")
			continue
		}
		if strings.HasPrefix(line, "# ") {
			result = append(result, "\x1b[1;32m"+strings.TrimPrefix(line, "# ")+"\x1b[0m")
			continue
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") {
			text := strings.TrimPrefix(line, "> ")
			const prefixW = 2 // "│ " visual width
			if maxWidth > prefixW+1 {
				for _, wl := range wordWrapText(text, maxWidth-prefixW) {
					result = append(result, "\x1b[2m│ \x1b[0m"+renderInline(wl))
				}
			} else {
				result = append(result, "\x1b[2m│ "+renderInline(text)+"\x1b[0m")
			}
			continue
		}

		// Unordered list
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ") {
			bullet := "  • "
			text := line[2:]
			result = append(result, "\x1b[36m"+bullet+"\x1b[0m"+renderInline(text))
			continue
		}
		if strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "  * ") {
			bullet := "    ◦ "
			text := line[4:]
			result = append(result, "\x1b[36m"+bullet+"\x1b[0m"+renderInline(text))
			continue
		}

		// Ordered list: simple check for "N. "
		if len(line) > 2 && line[0] >= '0' && line[0] <= '9' {
			dotIdx := strings.Index(line, ". ")
			if dotIdx > 0 && dotIdx < 4 {
				num := line[:dotIdx]
				text := line[dotIdx+2:]
				result = append(result, "\x1b[36m  "+num+". \x1b[0m"+renderInline(text))
				continue
			}
		}

		// Empty line
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
			continue
		}

		// Regular paragraph
		result = append(result, renderInline(line))
	}

	flushTable()
	return result
}

// renderTable renders pipe-delimited markdown table rows as a box-drawing table.
// maxWidth > 0 caps column widths so the table fits within that visual width,
// wrapping long cell content onto continuation lines.
func renderTable(rows []string, maxWidth int) []string {
	// Parse rows into plain-text cells; record the header separator position.
	var dataRows [][]string
	separatorAfter := -1

	for _, row := range rows {
		inner := strings.Trim(row, "| ")
		isSep := true
		for _, cell := range strings.Split(inner, "|") {
			cell = strings.TrimSpace(cell)
			if cell != "" && strings.Trim(cell, ":-") != "" {
				isSep = false
				break
			}
		}
		if isSep {
			if len(dataRows) > 0 && separatorAfter < 0 {
				separatorAfter = len(dataRows) - 1
			}
			continue
		}
		cells := strings.Split(inner, "|")
		trimmed := make([]string, len(cells))
		for j, c := range cells {
			trimmed[j] = strings.TrimSpace(c)
		}
		dataRows = append(dataRows, trimmed)
	}
	if len(dataRows) == 0 {
		return nil
	}

	numCols := 0
	for _, row := range dataRows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	// Natural column widths (visual width of rendered inline content).
	colW := make([]int, numCols)
	for _, row := range dataRows {
		for j := 0; j < numCols; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			w := runewidth.StringWidth(mdStripANSI(renderInline(cell)))
			if w > colW[j] {
				colW[j] = w
			}
		}
	}
	for j := range colW {
		if colW[j] < 1 {
			colW[j] = 1
		}
	}

	// Cap column widths when maxWidth is set and the table would overflow.
	// Total width = 1 (left border) + numCols*(colW+2+1) = 1 + sum(colW+3)
	if maxWidth > 0 {
		totalW := 1
		for _, w := range colW {
			totalW += w + 3
		}
		if totalW > maxWidth {
			// Available chars for all column content combined.
			avail := maxWidth - 1 - numCols*3
			if avail < numCols {
				avail = numCols
			}
			// Distribute: keep small columns at natural width, shrink large ones.
			order := make([]int, numCols)
			for i := range order {
				order[i] = i
			}
			sort.Slice(order, func(a, b int) bool {
				return colW[order[a]] < colW[order[b]]
			})
			capped := make([]int, numCols)
			remaining := avail
			for rank, idx := range order {
				share := remaining / (numCols - rank)
				if colW[idx] <= share {
					capped[idx] = colW[idx]
				} else {
					capped[idx] = max(1, share)
				}
				remaining -= capped[idx]
			}
			colW = capped
		}
	}

	const (
		dimANSI   = "\x1b[2m"
		boldANSI  = "\x1b[1m"
		resetANSI = "\x1b[0m"
	)

	borderLine := func(left, mid, right, fill string) string {
		var sb strings.Builder
		sb.WriteString(dimANSI + left)
		for j, w := range colW {
			sb.WriteString(strings.Repeat(fill, w+2))
			if j < numCols-1 {
				sb.WriteString(mid)
			}
		}
		sb.WriteString(right + resetANSI)
		return sb.String()
	}

	var out []string
	out = append(out, borderLine("┌", "┬", "┐", "─"))

	isHeader := func(rowIdx int) bool {
		return rowIdx == 0 && separatorAfter == 0
	}

	for rowIdx, row := range dataRows {
		// Word-wrap each cell's plain text to its column width, then apply inline fmt.
		cellLines := make([][]string, numCols)
		maxLines := 1
		for j := 0; j < numCols; j++ {
			raw := ""
			if j < len(row) {
				raw = row[j]
			}
			wrapLines := wordWrapText(raw, colW[j])
			rendered := make([]string, len(wrapLines))
			for k, wl := range wrapLines {
				rendered[k] = renderInline(wl)
			}
			cellLines[j] = rendered
			if len(rendered) > maxLines {
				maxLines = len(rendered)
			}
		}

		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			var sb strings.Builder
			sb.WriteString(dimANSI + "│" + resetANSI)
			for j := 0; j < numCols; j++ {
				var rendered string
				if lineIdx < len(cellLines[j]) {
					rendered = cellLines[j][lineIdx]
				}
				visW := runewidth.StringWidth(mdStripANSI(rendered))
				pad := colW[j] - visW
				if pad < 0 {
					pad = 0
				}
				if lineIdx == 0 && isHeader(rowIdx) {
					sb.WriteString(" " + boldANSI + rendered + resetANSI + strings.Repeat(" ", pad+1))
				} else {
					sb.WriteString(" " + rendered + resetANSI + strings.Repeat(" ", pad+1))
				}
				sb.WriteString(dimANSI + "│" + resetANSI)
			}
			out = append(out, sb.String())
		}

		if rowIdx == separatorAfter {
			out = append(out, borderLine("├", "┼", "┤", "─"))
		}
	}

	out = append(out, borderLine("└", "┴", "┘", "─"))
	return out
}

// wordWrapText wraps plain text at word boundaries to at most maxW visual columns.
func wordWrapText(s string, maxW int) []string {
	if maxW <= 0 || runewidth.StringWidth(s) <= maxW {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	var cur strings.Builder
	curW := 0
	for _, word := range words {
		wordW := runewidth.StringWidth(word)
		if curW == 0 {
			cur.WriteString(word)
			curW = wordW
		} else if curW+1+wordW <= maxW {
			cur.WriteRune(' ')
			cur.WriteString(word)
			curW += 1 + wordW
		} else {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(word)
			curW = wordW
		}
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// mdStripANSI removes ANSI escape sequences for visual-width measurement.
func mdStripANSI(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
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

// runeIndexOf finds the rune-index of sub in runes starting at start.
// Returns -1 if not found.
func runeIndexOf(runes []rune, sub []rune, start int) int {
	if len(sub) == 0 {
		return start
	}
	for i := start; i <= len(runes)-len(sub); i++ {
		found := true
		for j, r := range sub {
			if runes[i+j] != r {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}

// renderInline handles bold, italic, inline code, and links in a line.
// All indexing is done in rune space to avoid Unicode slice panics.
func renderInline(s string) string {
	var b strings.Builder
	i := 0
	runes := []rune(s)
	n := len(runes)

	for i < n {
		// Bold (**text** or __text__)
		if i+1 < n && ((runes[i] == '*' && runes[i+1] == '*') || (runes[i] == '_' && runes[i+1] == '_')) {
			delim := runes[i : i+2]
			end := runeIndexOf(runes, delim, i+2)
			if end >= 0 {
				b.WriteString("\x1b[1m")
				b.WriteString(string(runes[i+2 : end]))
				b.WriteString("\x1b[0m")
				i = end + 2
				continue
			}
		}
		// Italic (*text* or _text_)
		if (runes[i] == '*' || runes[i] == '_') && (i == 0 || runes[i-1] == ' ') {
			delim := runes[i : i+1]
			end := runeIndexOf(runes, delim, i+1)
			if end >= 0 {
				b.WriteString("\x1b[3m")
				b.WriteString(string(runes[i+1 : end]))
				b.WriteString("\x1b[0m")
				i = end + 1
				continue
			}
		}
		// Inline code `text`
		if runes[i] == '`' && i+1 < n {
			end := runeIndexOf(runes, []rune{'`'}, i+1)
			if end >= 0 {
				b.WriteString("\x1b[33m")
				b.WriteString(string(runes[i+1 : end]))
				b.WriteString("\x1b[0m")
				i = end + 1
				continue
			}
		}
		// Link [text](url)
		if runes[i] == '[' {
			closeB := runeIndexOf(runes, []rune{']'}, i+1)
			if closeB > 0 && closeB+1 < n && runes[closeB+1] == '(' {
				closeP := runeIndexOf(runes, []rune{')'}, closeB+2)
				if closeP > 0 {
					linkText := string(runes[i+1 : closeB])
					b.WriteString("\x1b[34;4m")
					b.WriteString(linkText)
					b.WriteString("\x1b[0m")
					i = closeP + 1
					continue
				}
			}
		}

		b.WriteRune(runes[i])
		i++
	}
	return b.String()
}
