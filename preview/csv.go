package preview

import (
	"encoding/csv"
	"strings"

	"github.com/mattn/go-runewidth"
)

// RenderCSV parses CSV data and returns a colored, aligned table as lines.
func RenderCSV(data string) []string {
	r := csv.NewReader(strings.NewReader(data))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil || len(records) == 0 {
		if err != nil {
			return []string{"  CSV 파싱 실패: " + err.Error()}
		}
		return []string{}
	}

	colCount := 0
	for _, row := range records {
		if len(row) > colCount {
			colCount = len(row)
		}
	}
	if colCount == 0 {
		return []string{}
	}

	// Use runewidth.StringWidth for correct CJK column width
	widths := make([]int, colCount)
	for _, row := range records {
		for j, cell := range row {
			if j < colCount {
				w := runewidth.StringWidth(cell)
				if w > widths[j] {
					widths[j] = w
				}
			}
		}
	}

	const (
		ansiHeader = "\x1b[1;38;5;39m"  // bold blue for header
		ansiAltRow = "\x1b[38;5;244m"   // dim for alternate rows
		ansiSep    = "\x1b[38;5;237m"   // dark separator
		ansiReset  = "\x1b[0m"
	)

	var lines []string
	for i, row := range records {
		var sb strings.Builder
		for j := 0; j < colCount; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			cellW := runewidth.StringWidth(cell)
			padded := cell + strings.Repeat(" ", widths[j]-cellW)
			if j > 0 {
				sb.WriteString(ansiSep + " │ " + ansiReset)
			}
			if i == 0 {
				sb.WriteString(ansiHeader + padded + ansiReset)
			} else if i%2 == 0 {
				sb.WriteString(ansiAltRow + padded + ansiReset)
			} else {
				sb.WriteString(padded)
			}
		}
		lines = append(lines, sb.String())

		// Header separator row
		if i == 0 {
			var sep strings.Builder
			for j := 0; j < colCount; j++ {
				if j > 0 {
					sep.WriteString(ansiSep + "─┼─" + ansiReset)
				}
				sep.WriteString(ansiSep + strings.Repeat("─", widths[j]) + ansiReset)
			}
			lines = append(lines, sep.String())
		}
	}

	return lines
}
