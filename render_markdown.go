package tmark

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func ToMarkdown(content RichBlocks, options ...RenderOption) ([]byte, error) {
	renderer := markdownRenderer{cfg: newRenderConfig(options)}
	if err := renderer.blocks(content); err != nil {
		return nil, err
	}
	return bytes.TrimRight(renderer.out.Bytes(), "\n"), nil
}

type markdownRenderer struct {
	cfg renderConfig
	out bytes.Buffer
}

func (r *markdownRenderer) blocks(blocks RichBlocks) error {
	for i, block := range blocks {
		if i > 0 {
			r.out.WriteString("\n")
		}
		if err := r.block(block); err != nil {
			return err
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
		var nested markdownRenderer
		nested.cfg = r.cfg
		if err := nested.blocks(n.Blocks); err != nil {
			return err
		}
		if len(n.Credit) > 0 {
			if nested.out.Len() > 0 {
				nested.out.WriteByte('\n')
			}
			nested.out.WriteString("- ")
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
		r.out.WriteString("> ")
		if err := r.rich(n.Text); err != nil {
			return err
		}
		if len(n.Credit) > 0 {
			r.out.WriteString("\n> - ")
			if err := r.rich(n.Credit); err != nil {
				return err
			}
		}
		r.out.WriteByte('\n')
	case List:
		return r.list(n)
	case Divider:
		r.out.WriteString("---\n")
	case Preformatted:
		fence := codeFence(string(n.Text))
		r.out.WriteString(fence)
		if n.Language != nil {
			r.out.WriteString(string(*n.Language))
		}
		r.out.WriteByte('\n')
		r.out.WriteString(string(n.Text))
		if !strings.HasSuffix(string(n.Text), "\n") {
			r.out.WriteByte('\n')
		}
		r.out.WriteString(fence)
		r.out.WriteByte('\n')
	case MathBlock:
		r.out.WriteString("$$\n")
		r.out.WriteString(string(n.Expression))
		r.out.WriteString("\n$$\n")
	case Anchor:
		r.out.WriteString("<a id=\"")
		r.out.WriteString(escapeHTMLAttr(string(n.Name)))
		r.out.WriteString("\"></a>\n")
	case Map:
		return r.unsupported("map", "markdown has no map block")
	case Image:
		r.out.WriteString("![](")
		r.out.WriteString(escapeURL(string(n.Src)))
		r.out.WriteString(")\n")
		return r.caption(n.Caption)
	case Video:
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
		r.out.WriteString("<details")
		if n.IsOpen {
			r.out.WriteString(" open")
		}
		r.out.WriteString(">\n<summary>")
		if err := r.rich(n.Summary); err != nil {
			return err
		}
		r.out.WriteString("</summary>\n\n")
		if err := r.blocks(n.Children); err != nil {
			return err
		}
		r.out.WriteString("</details>\n")
	default:
		return r.unsupported(fmt.Sprintf("%T", node), "unsupported block node")
	}
	return nil
}

func (r *markdownRenderer) list(list List) error {
	ordered := hasOrderedItems(list)
	for index, item := range list.Items {
		marker := "- "
		if item.Order != nil {
			marker = strconv.Itoa(*item.Order) + ". "
		} else if ordered {
			marker = strconv.Itoa(index+1) + ". "
		}
		r.out.WriteString(marker)
		if item.Checked != nil {
			if *item.Checked {
				r.out.WriteString("[x] ")
			} else {
				r.out.WriteString("[ ] ")
			}
		}
		if len(item.Children) == 1 {
			if p, ok := item.Children[0].(Paragraph); ok {
				if err := r.rich(p.Children); err != nil {
					return err
				}
				r.out.WriteByte('\n')
				continue
			}
		}
		r.out.WriteByte('\n')
		var nested markdownRenderer
		nested.cfg = r.cfg
		if err := nested.blocks(item.Children); err != nil {
			return err
		}
		for _, line := range strings.Split(strings.TrimRight(nested.out.String(), "\n"), "\n") {
			r.out.WriteString("  ")
			r.out.WriteString(line)
			r.out.WriteByte('\n')
		}
	}
	return nil
}

func (r *markdownRenderer) slides(children []slideshowContent, caption *Caption) error {
	for _, child := range children {
		switch n := child.(type) {
		case Image:
			if err := r.block(n); err != nil {
				return err
			}
		case Video:
			if err := r.block(n); err != nil {
				return err
			}
		default:
			return r.unsupported(fmt.Sprintf("%T", n), "unsupported slideshow node")
		}
	}
	return r.caption(caption)
}

func (r *markdownRenderer) table(table Table) error {
	if len(table.Caption) > 0 {
		if err := r.rich(table.Caption); err != nil {
			return err
		}
		r.out.WriteString("\n\n")
	}
	if len(table.Rows) == 0 {
		return nil
	}
	header := table.Rows[0].Cells
	if err := r.tableRow(header); err != nil {
		return err
	}
	r.tableSeparator(header)
	for _, row := range table.Rows[1:] {
		if err := r.tableRow(row.Cells); err != nil {
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
	r.out.WriteByte('*')
	if err := r.rich(caption.Text); err != nil {
		return err
	}
	if len(caption.Credit) > 0 {
		r.out.WriteString(" - ")
		if err := r.rich(caption.Credit); err != nil {
			return err
		}
	}
	r.out.WriteString("*\n")
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
		r.out.WriteByte('[')
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("](#")
		r.out.WriteString(escapeURL(string(n.AnchorName)))
		r.out.WriteByte(')')
	case Reference:
		r.out.WriteString("<span id=\"")
		r.out.WriteString(escapeHTMLAttr(string(n.Name)))
		r.out.WriteString("\">")
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("</span>")
	case ReferenceLink:
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
		return r.wrap("==", "==", n.Children)
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
		r.out.WriteString(strconv.FormatInt(n.Unix, 10))
	case Code:
		r.out.WriteString(inlineCode(string(n.Text)))
	case Math:
		r.out.WriteByte('$')
		r.out.WriteString(strings.ReplaceAll(string(n.Expression), "$", "\\$"))
		r.out.WriteByte('$')
	case Icon:
		r.out.WriteString("![")
		if n.AlternativeText != nil {
			r.out.WriteString(escapeMarkdownText(string(*n.AlternativeText)))
		}
		r.out.WriteString("](")
		r.out.WriteString(escapeURL(string(n.Src)))
		r.out.WriteByte(')')
	default:
		return r.unsupported(fmt.Sprintf("%T", node), "unsupported rich text node")
	}
	return nil
}

func (r *markdownRenderer) wrap(open, close string, children RichText) error {
	r.out.WriteString(open)
	if err := r.rich(children); err != nil {
		return err
	}
	r.out.WriteString(close)
	return nil
}

func (r *markdownRenderer) htmlWrap(tag string, children RichText) error {
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

func (r *markdownRenderer) unsupported(tag string, reason string) error {
	if !r.cfg.soft {
		return unsupportedRenderTag(tag, reason)
	}
	r.out.WriteString(r.cfg.placeholder)
	r.out.WriteByte('\n')
	return nil
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
	renderer := markdownRenderer{cfg: r.cfg}
	if err := renderer.rich(nodes); err != nil {
		return "", err
	}
	text := strings.TrimSpace(strings.ReplaceAll(renderer.out.String(), "\n", " "))
	return strings.ReplaceAll(text, "|", "\\|"), nil
}

func inlineCode(value string) string {
	fence := "`"
	for strings.Contains(value, fence) {
		fence += "`"
	}
	if strings.HasPrefix(value, "`") || strings.HasSuffix(value, "`") {
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
		"|", "\\|",
	)
	return strings.ReplaceAll(replacer.Replace(value), "\n", "  \n")
}

func escapeURL(value string) string {
	return strings.ReplaceAll(value, ")", "%29")
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
