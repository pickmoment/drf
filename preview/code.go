package preview

import (
	"bytes"
	"strings"

	chromaLexers "github.com/alecthomas/chroma/v2/lexers"
	chromaFormatters "github.com/alecthomas/chroma/v2/formatters"
	chromaStyles "github.com/alecthomas/chroma/v2/styles"
)

// HighlightCode uses chroma to syntax-highlight the given source code.
// lang is the language name (e.g. "go", "python"). theme is the chroma style name.
// Returns a slice of ANSI-colored lines.
func HighlightCode(src, lang, theme string) []string {
	lexer := chromaLexers.Get(lang)
	if lexer == nil {
		lexer = chromaLexers.Fallback
	}

	style := chromaStyles.Get(theme)
	if style == nil {
		style = chromaStyles.Fallback
	}

	formatter := chromaFormatters.Get("terminal16m")
	if formatter == nil {
		formatter = chromaFormatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, src)
	if err != nil {
		return plainLines(src)
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return plainLines(src)
	}

	raw := buf.String()
	// Remove trailing newline to avoid extra blank line
	raw = strings.TrimRight(raw, "\n")
	return strings.Split(raw, "\n")
}

func plainLines(src string) []string {
	src = strings.TrimRight(src, "\n")
	return strings.Split(src, "\n")
}
