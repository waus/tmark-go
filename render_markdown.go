package tmark

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

// ToMarkdown renders GFM; htmlLess disables raw HTML tags and comments.
func ToMarkdown(content RichBlocks, htmlLess bool) ([]byte, error) {
	renderer := markdownRenderer{htmlLess: htmlLess}
	if err := renderer.blocks(content); err != nil {
		return nil, err
	}
	return bytes.TrimRight(renderer.out.Bytes(), "\n"), nil
}

type markdownRenderer struct {
	htmlLess bool
	out      bytes.Buffer
}

func (r *markdownRenderer) blocks(blocks RichBlocks) error {
	for i, block := range blocks {
		if i > 0 {
			r.out.WriteString("\n")
			if _, current := block.(List); current {
				if _, previous := blocks[i-1].(List); previous {
					if r.htmlLess {
						r.out.WriteString("---\n\n")
					} else {
						r.out.WriteString("<!-- -->\n\n")
					}
				}
			}
		}
		if err := r.block(block); err != nil {
			return err
		}
		if r.out.Len() > 0 && r.out.Bytes()[r.out.Len()-1] != '\n' {
			r.out.WriteByte('\n')
		}
	}
	return nil
}

func (r *markdownRenderer) block(node any) error {
	switch n := node.(type) {
	case Paragraph:
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("\n")
	case Header:
		level := n.Size
		if level < 1 || level > 6 {
			level = 4
		}
		r.out.WriteString(strings.Repeat("#", level))
		r.out.WriteByte(' ')
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("\n")
	case Blockquote:
		nested := markdownRenderer{htmlLess: r.htmlLess}
		if err := nested.blocks(n.Blocks); err != nil {
			return err
		}
		if len(n.Credit) > 0 {
			if nested.out.Len() > 0 {
				nested.out.WriteByte('\n')
			}
			nested.out.WriteString("— ")
			if err := nested.rich(n.Credit); err != nil {
				return err
			}
			nested.out.WriteByte('\n')
		}
		for _, line := range strings.Split(strings.TrimRight(nested.out.String(), "\n"), "\n") {
			r.out.WriteString("> ")
			r.out.WriteString(line)
			r.out.WriteByte('\n')
		}
	case PullQuote:
		return r.block(NewBlockquote(NewParagraph(n.Text...)).WithCredit(n.Credit...))
	case List:
		return r.list(n)
	case Divider:
		r.out.WriteString("---\n")
	case Preformatted:
		fence := codeFence(string(n.Text))
		r.out.WriteString(fence)
		if n.Language != nil {
			r.out.WriteString(strings.ReplaceAll(strings.Join(strings.Fields(string(*n.Language)), " "), "`", ""))
		}
		r.out.WriteByte('\n')
		r.out.WriteString(string(n.Text))
		if !strings.HasSuffix(string(n.Text), "\n") {
			r.out.WriteByte('\n')
		}
		r.out.WriteString(fence)
		r.out.WriteByte('\n')
	case MathBlock:
		return r.block(NewPreformatted(n.Expression))
	case Anchor:
		if r.htmlLess {
			return nil
		}
		r.out.WriteString("<a id=\"")
		r.out.WriteString(escapeHTMLAttr(string(n.Name)))
		r.out.WriteString("\"></a>\n")
	case Image:
		if n.HasSpoiler && !r.htmlLess {
			return r.htmlBlock(n)
		}
		r.out.WriteString("![](")
		r.out.WriteString(escapeURL(string(n.Src)))
		r.out.WriteString(")\n")
		return r.caption(n.Caption)
	case Video:
		if n.HasSpoiler && !r.htmlLess {
			return r.htmlBlock(n)
		}
		r.out.WriteString("[video](")
		r.out.WriteString(escapeURL(string(n.Src)))
		r.out.WriteString(")\n")
		return r.caption(n.Caption)
	case Audio:
		r.out.WriteString("[audio](")
		r.out.WriteString(escapeURL(string(n.Src)))
		r.out.WriteString(")\n")
		return r.caption(n.Caption)
	case Collage:
		return r.slides(n.Children, n.Caption)
	case Slideshow:
		return r.slides(n.Children, n.Caption)
	case Table:
		return r.table(n)
	case Details:
		if r.htmlLess {
			if len(n.Summary) > 0 {
				if err := r.wrap("**", "**", n.Summary); err != nil {
					return err
				}
				r.out.WriteString("\n\n")
			}
			return r.blocks(n.Children)
		}
		r.out.WriteString("<details")
		if n.IsOpen {
			r.out.WriteString(" open")
		}
		r.out.WriteString(">\n<summary>")
		summary := htmlRenderer{}
		if err := summary.rich(n.Summary); err != nil {
			return err
		}
		r.out.Write(summary.out.Bytes())
		r.out.WriteString("</summary>\n\n")
		if err := r.blocks(n.Children); err != nil {
			return err
		}
		r.out.WriteString("\n</details>\n")
	default:
		return r.fallback(node, true)
	}
	return nil
}

func (r *markdownRenderer) list(list List) error {
	ordered := hasOrderedItems(list)
	next := 1
	for _, item := range list.Items {
		marker := "- "
		if item.Order != nil {
			next = *item.Order
		}
		if ordered {
			marker = strconv.Itoa(next) + ". "
			next++
		}
		prefix := marker
		if item.Checked != nil {
			if *item.Checked {
				prefix += "[x] "
			} else {
				prefix += "[ ] "
			}
		}
		nested := markdownRenderer{htmlLess: r.htmlLess}
		if err := nested.blocks(item.Children); err != nil {
			return err
		}
		for i, line := range strings.Split(strings.TrimRight(nested.out.String(), "\n"), "\n") {
			if i == 0 {
				r.out.WriteString(prefix)
			} else if line != "" {
				r.out.WriteString(strings.Repeat(" ", len(marker)))
			}
			r.out.WriteString(line)
			r.out.WriteByte('\n')
		}
	}
	return nil
}

func (r *markdownRenderer) slides(children []slideshowContent, caption *Caption) error {
	for _, child := range children {
		if err := r.block(child); err != nil {
			return err
		}
		r.out.WriteByte('\n')
	}
	return r.caption(caption)
}

func (r *markdownRenderer) table(table Table) error {
	for _, row := range table.Rows {
		for _, cell := range row.Cells {
			if !r.htmlLess && (cell.Colspan != nil && *cell.Colspan > 1 || cell.Rowspan != nil && *cell.Rowspan > 1) {
				return r.htmlBlock(table)
			}
		}
	}
	if len(table.Caption) > 0 {
		if err := r.rich(table.Caption); err != nil {
			return err
		}
		r.out.WriteString("\n\n")
	}
	if len(table.Rows) == 0 {
		return nil
	}
	columns := 0
	for _, row := range table.Rows {
		columns = max(columns, len(row.Cells))
	}
	if columns == 0 {
		return nil
	}
	header := make([]Cell, columns)
	rows := table.Rows
	for i, cell := range rows[0].Cells {
		header[i].Align = cell.Align
	}
	for _, cell := range rows[0].Cells {
		if cell.IsHeader {
			copy(header, rows[0].Cells)
			rows = rows[1:]
			break
		}
	}
	if err := r.tableRow(header); err != nil {
		return err
	}
	r.tableSeparator(header)
	for _, row := range rows {
		cells := make([]Cell, columns)
		copy(cells, row.Cells)
		if err := r.tableRow(cells); err != nil {
			return err
		}
	}
	return nil
}

func (r *markdownRenderer) tableRow(row []Cell) error {
	r.out.WriteByte('|')
	for _, cell := range row {
		text, err := r.tableText(cell.Children)
		if err != nil {
			return err
		}
		r.out.WriteByte(' ')
		r.out.WriteString(text)
		r.out.WriteString(" |")
	}
	r.out.WriteByte('\n')
	return nil
}

func (r *markdownRenderer) tableSeparator(row []Cell) {
	r.out.WriteByte('|')
	for _, cell := range row {
		switch {
		case cell.Align != nil && *cell.Align == TableCellAlignLeft:
			r.out.WriteString(" :--- |")
		case cell.Align != nil && *cell.Align == TableCellAlignCenter:
			r.out.WriteString(" :---: |")
		case cell.Align != nil && *cell.Align == TableCellAlignRight:
			r.out.WriteString(" ---: |")
		default:
			r.out.WriteString(" --- |")
		}
	}
	r.out.WriteByte('\n')
}

func (r *markdownRenderer) caption(caption *Caption) error {
	if caption == nil {
		return nil
	}
	r.out.WriteByte('\n')
	if err := r.wrap("_", "_", caption.Text); err != nil {
		return err
	}
	if len(caption.Credit) > 0 {
		if len(caption.Text) > 0 {
			r.out.WriteString("  \n")
		}
		if err := r.wrap("_", "_", caption.Credit); err != nil {
			return err
		}
	}
	r.out.WriteByte('\n')
	return nil
}

func (r *markdownRenderer) rich(nodes RichText) error {
	for _, node := range nodes {
		if err := r.richNode(node); err != nil {
			return err
		}
	}
	return nil
}

func (r *markdownRenderer) richNode(node any) error {
	switch n := node.(type) {
	case Text:
		r.out.WriteString(escapeMarkdownText(string(n)))
	case Link:
		r.out.WriteByte('[')
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("](")
		r.out.WriteString(escapeURL(string(n.Href)))
		r.out.WriteByte(')')
	case AnchorLink:
		if r.htmlLess {
			return r.rich(n.Children)
		}
		r.out.WriteByte('[')
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("](#")
		r.out.WriteString(escapeURL(string(n.AnchorName)))
		r.out.WriteByte(')')
	case Reference:
		if r.htmlLess {
			return r.rich(n.Children)
		}
		r.out.WriteString("<span id=\"")
		r.out.WriteString(escapeHTMLAttr(string(n.Name)))
		r.out.WriteString("\">")
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("</span>")
	case ReferenceLink:
		if r.htmlLess {
			return r.rich(n.Children)
		}
		r.out.WriteByte('[')
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("](#")
		r.out.WriteString(escapeURL(string(n.ReferenceName)))
		r.out.WriteByte(')')
	case Bold:
		return r.wrap("**", "**", n.Children)
	case Italic:
		return r.wrap("*", "*", n.Children)
	case Marked:
		return r.htmlWrap("mark", n.Children)
	case Underline:
		return r.htmlWrap("u", n.Children)
	case Strikethrough:
		return r.wrap("~~", "~~", n.Children)
	case Spoiler:
		return r.htmlWrapClass("span", "spoiler", n.Children)
	case Subscript:
		return r.htmlWrap("sub", n.Children)
	case Superscript:
		return r.htmlWrap("sup", n.Children)
	case DateTime:
		location, err := time.LoadLocation(string(n.Timezone))
		if err != nil {
			return r.fallback(n, false)
		}
		r.out.WriteString(escapeMarkdownText(time.Unix(n.Unix, 0).In(location).Format("2006-01-02 15:04:05 MST")))
	case Code:
		r.out.WriteString(r.inlineCode(string(n.Text)))
	case Math:
		r.out.WriteString(r.inlineCode(string(n.Expression)))
	case Icon:
		r.out.WriteString("![")
		if n.AlternativeText != nil {
			r.out.WriteString(escapeMarkdownText(string(*n.AlternativeText)))
		}
		r.out.WriteString("](")
		r.out.WriteString(escapeURL(string(n.Src)))
		r.out.WriteByte(')')
	default:
		return r.fallback(node, false)
	}
	return nil
}

func (r *markdownRenderer) wrap(open, close string, children RichText) error {
	nested := markdownRenderer{htmlLess: r.htmlLess}
	if err := nested.rich(children); err != nil {
		return err
	}
	value := nested.out.String()
	text := strings.TrimSpace(value)
	if text == "" {
		r.out.WriteString(value)
		return nil
	}
	start := strings.Index(value, text)
	r.out.WriteString(value[:start])
	r.out.WriteString(open)
	r.out.WriteString(text)
	r.out.WriteString(close)
	r.out.WriteString(value[start+len(text):])
	return nil
}

func (r *markdownRenderer) htmlBlock(node any) error {
	renderer := htmlRenderer{}
	if err := renderer.block(node); err != nil {
		return err
	}
	r.out.Write(renderer.out.Bytes())
	return nil
}

func (r *markdownRenderer) htmlWrap(tag string, children RichText) error {
	if r.htmlLess {
		return r.rich(children)
	}
	r.out.WriteByte('<')
	r.out.WriteString(tag)
	r.out.WriteByte('>')
	if err := r.rich(children); err != nil {
		return err
	}
	r.out.WriteString("</")
	r.out.WriteString(tag)
	r.out.WriteByte('>')
	return nil
}

func (r *markdownRenderer) htmlWrapClass(tag, class string, children RichText) error {
	if r.htmlLess {
		return r.rich(children)
	}
	r.out.WriteByte('<')
	r.out.WriteString(tag)
	r.out.WriteString(" class=\"")
	r.out.WriteString(class)
	r.out.WriteString("\">")
	if err := r.rich(children); err != nil {
		return err
	}
	r.out.WriteString("</")
	r.out.WriteString(tag)
	r.out.WriteByte('>')
	return nil
}

func (r *markdownRenderer) fallback(node any, block bool) error {
	data, err := Marshal(node)
	if err != nil {
		return err
	}
	if block {
		return r.block(NewPreformatted(Text(data)))
	}
	return r.richNode(NewCode(Text(data)))
}

func hasOrderedItems(list List) bool {
	for _, item := range list.Items {
		if item.Order != nil {
			return true
		}
	}
	return false
}

func (r *markdownRenderer) tableText(nodes RichText) (string, error) {
	renderer := markdownRenderer{htmlLess: r.htmlLess}
	if err := renderer.rich(nodes); err != nil {
		return "", err
	}
	lineBreak := "<br>"
	if r.htmlLess {
		lineBreak = " "
	}
	text := strings.TrimSpace(strings.NewReplacer("  \n", lineBreak, "\n", lineBreak).Replace(renderer.out.String()))
	return strings.ReplaceAll(text, "|", "\\|"), nil
}

func (r *markdownRenderer) inlineCode(value string) string {
	if value == "" {
		if r.htmlLess {
			return ""
		}
		return "<code></code>"
	}
	fence := "`"
	for strings.Contains(value, fence) {
		fence += "`"
	}
	if strings.HasPrefix(value, "`") || strings.HasSuffix(value, "`") ||
		strings.HasPrefix(value, " ") && strings.HasSuffix(value, " ") && strings.Trim(value, " ") != "" {
		return fence + " " + value + " " + fence
	}
	return fence + value + fence
}

func codeFence(value string) string {
	fence := "```"
	for strings.Contains(value, fence) {
		fence += "`"
	}
	return fence
}

func escapeMarkdownText(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
		"<", "\\<",
		">", "\\>",
		"#", "\\#",
		"&", "\\&",
		"`", "\\`",
		"~", "\\~",
		"!", "\\!",
		"-", "\\-",
		"+", "\\+",
		"=", "\\=",
		".", "\\.",
		")", "\\)",
	)
	return strings.ReplaceAll(replacer.Replace(value), "\n", "  \n")
}

func escapeURL(value string) string {
	return strings.NewReplacer(
		"\\", "%5C", " ", "%20", "\t", "%09", "\n", "%0A", "\r", "%0D",
		"(", "%28", ")", "%29", "<", "%3C", ">", "%3E", "\"", "%22", "&", "&amp;",
	).Replace(value)
}

func escapeHTMLAttr(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"\"", "&quot;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(value)
}
