package preview

import (
	"bytes"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func Highlight(source, filename, themeName string) (string, string) {
	matched := lexers.Match(filename)

	var lang string
	var lexer chroma.Lexer
	if matched == nil {
		lexer = chroma.Coalesce(lexers.Fallback)
		lang = "plaintext"
	} else {
		lexer = chroma.Coalesce(matched)
		lang = lexer.Config().Name
	}

	style := styles.Get(themeName)
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source, "plaintext"
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return source, "plaintext"
	}

	return buf.String(), lang
}
