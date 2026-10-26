package tmark

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ToTerminal renders static terminal output with ANSI styles and no terminal I/O.
// plainText disables ANSI styles; revealSpoilers includes hidden text and media.
func ToTerminal(content RichBlocks, plainText, revealSpoilers bool) ([]byte, error) {
	r := terminalRenderer{plainText: plainText, revealSpoilers: revealSpoilers}
	if err := r.blocks(content); err != nil {
		return nil, err
	}
	return r.out.Bytes(), nil
}

type terminalRenderer struct {
	plainText      bool
	revealSpoilers bool
	out            bytes.Buffer
	style          string
}

// All document strings go through text: terminal controls must never come from input.
func (r *terminalRenderer) text(value string) {
	for _, c := range value {
		switch {
		case c == '\n':
			r.out.WriteByte('\n')
		case c == '\t':
			r.out.WriteString("    ")
		case !unicode.IsControl(c):
			r.out.WriteRune(c)
		}
	}
}

func (r *terminalRenderer) styled(style string, children RichText) error {
	if r.plainText || len(children) == 0 {
		return r.rich(children)
	}
	previous := r.style
	r.style += "\x1b[" + style + "m"
	r.out.WriteString("\x1b[" + style + "m")
	if err := r.rich(children); err != nil {
		return err
	}
	r.style = previous
	r.out.WriteString("\x1b[0m" + previous)
	return nil
}

func (r *terminalRenderer) blocks(blocks RichBlocks) error {
	for _, block := range blocks {
		nested := terminalRenderer{plainText: r.plainText, revealSpoilers: r.revealSpoilers}
		if err := nested.block(block); err != nil {
			return err
		}
		value := strings.TrimRight(nested.out.String(), "\n")
		if value == "" {
			continue
		}
		if r.out.Len() > 0 {
			r.out.WriteString("\n\n")
		}
		r.out.WriteString(value)
	}
	return nil
}

func (r *terminalRenderer) block(node any) error {
	switch n := node.(type) {
	case Paragraph:
		return r.rich(n.Children)
	case Header:
		level := n.Size
		if level < 1 || level > 6 {
			level = 4
		}
		r.out.WriteString(strings.Repeat("#", level) + " ")
		return r.styled("1;36", n.Children)
	case Divider:
		r.out.WriteString("────────────────────────────────")
	case Anchor:
		return nil
	case Blockquote:
		nested := terminalRenderer{plainText: r.plainText, revealSpoilers: r.revealSpoilers}
		if err := nested.blocks(n.Blocks); err != nil {
			return err
		}
		if len(n.Credit) > 0 {
			nested.out.WriteString("\n\n— ")
			if err := nested.rich(n.Credit); err != nil {
				return err
			}
		}
		r.indent(nested.out.String(), "│ ", "│ ")
	case PullQuote:
		return r.block(NewBlockquote(NewParagraph(n.Text...)).WithCredit(n.Credit...))
	case List:
		next := 1
		ordered := hasOrderedItems(n)
		for i, item := range n.Items {
			if i > 0 {
				r.out.WriteByte('\n')
			}
			marker := "• "
			if item.Order != nil {
				next = *item.Order
			}
			if ordered {
				marker = strconv.Itoa(next) + ". "
				next++
			}
			if item.Checked != nil {
				if *item.Checked {
					marker += "[x] "
				} else {
					marker += "[ ] "
				}
			}
			nested := terminalRenderer{plainText: r.plainText, revealSpoilers: r.revealSpoilers}
			if err := nested.blocks(item.Children); err != nil {
				return err
			}
			r.indent(nested.out.String(), marker, strings.Repeat(" ", len([]rune(marker))))
		}
	case Preformatted:
		r.out.WriteString("┌─ code")
		if n.Language != nil {
			r.out.WriteByte(' ')
			r.text(string(*n.Language))
		}
		r.out.WriteByte('\n')
		nested := terminalRenderer{plainText: r.plainText, revealSpoilers: r.revealSpoilers}
		nested.text(string(n.Text))
		r.indent(strings.TrimSuffix(nested.out.String(), "\n"), "│ ", "│ ")
		r.out.WriteString("\n└─")
	case MathBlock:
		r.out.WriteString("math:\n")
		nested := terminalRenderer{plainText: r.plainText, revealSpoilers: r.revealSpoilers}
		nested.text(string(n.Expression))
		r.indent(nested.out.String(), "  ", "  ")
	case Details:
		r.out.WriteString("▾ ")
		if err := r.styled("1", n.Summary); err != nil {
			return err
		}
		nested := terminalRenderer{plainText: r.plainText, revealSpoilers: r.revealSpoilers}
		if err := nested.blocks(n.Children); err != nil {
			return err
		}
		if nested.out.Len() > 0 {
			r.out.WriteByte('\n')
			r.indent(nested.out.String(), "  ", "  ")
		}
	case Image:
		return r.media("image", n.Src, n.HasSpoiler, n.Caption)
	case Video:
		return r.media("video", n.Src, n.HasSpoiler, n.Caption)
	case Audio:
		return r.media("audio", n.Src, false, n.Caption)
	case Map:
		r.text(fmt.Sprintf("[map] %.6f, %.6f\nhttps://www.openstreetmap.org/?mlat=%g&mlon=%g", n.Lat, n.Lon, n.Lat, n.Lon))
		return r.caption(n.Caption)
	case Collage:
		return r.slides("collage", n.Children, n.Caption)
	case Slideshow:
		return r.slides("slideshow", n.Children, n.Caption)
	case Table:
		return r.table(n)
	default:
		return r.fallback(node, true)
	}
	return nil
}

func (r *terminalRenderer) indent(value, first, rest string) {
	for i, line := range strings.Split(value, "\n") {
		if i > 0 {
			r.out.WriteByte('\n')
			r.out.WriteString(rest)
		} else {
			r.out.WriteString(first)
		}
		r.out.WriteString(line)
	}
}

func (r *terminalRenderer) media(kind string, src Text, spoiler bool, caption *Caption) error {
	if spoiler && !r.revealSpoilers {
		r.out.WriteString("[spoiler: " + kind + "]")
		return nil
	}
	r.out.WriteString("[" + kind + "] ")
	r.text(string(src))
	return r.caption(caption)
}

func (r *terminalRenderer) caption(caption *Caption) error {
	if caption == nil {
		return nil
	}
	if len(caption.Text) > 0 {
		r.out.WriteByte('\n')
		if err := r.styled("2;3", caption.Text); err != nil {
			return err
		}
	}
	if len(caption.Credit) > 0 {
		r.out.WriteString("\n— ")
		return r.styled("2", caption.Credit)
	}
	return nil
}

func (r *terminalRenderer) slides(kind string, children []slideshowContent, caption *Caption) error {
	r.out.WriteString("[" + kind + "]")
	for i, child := range children {
		r.out.WriteByte('\n')
		nested := terminalRenderer{plainText: r.plainText, revealSpoilers: r.revealSpoilers}
		if err := nested.block(child); err != nil {
			return err
		}
		prefix := strconv.Itoa(i+1) + ". "
		r.indent(nested.out.String(), prefix, strings.Repeat(" ", len(prefix)))
	}
	return r.caption(caption)
}

// Cards preserve multiline cells without guessing terminal width or Unicode glyph widths.
// Spans are annotated instead of expanding a potentially unbounded grid.
func (r *terminalRenderer) table(table Table) error {
	if err := r.styled("1", table.Caption); err != nil {
		return err
	}
	var labels []string
	if len(table.Rows) > 1 {
		for _, cell := range table.Rows[0].Cells {
			if !cell.IsHeader {
				labels = nil
				break
			}
			nested := terminalRenderer{plainText: true, revealSpoilers: r.revealSpoilers}
			if err := nested.rich(cell.Children); err != nil {
				return err
			}
			labels = append(labels, strings.Join(strings.Fields(nested.out.String()), " "))
		}
		for _, row := range table.Rows {
			for _, cell := range row.Cells {
				if cell.Colspan != nil && *cell.Colspan > 1 || cell.Rowspan != nil && *cell.Rowspan > 1 {
					labels = nil
				}
			}
		}
	}
	if len(labels) > 0 {
		table.Rows = table.Rows[1:]
	}
	for i, row := range table.Rows {
		if r.out.Len() > 0 {
			r.out.WriteByte('\n')
		}
		fmt.Fprintf(&r.out, "┌─ row %d", i+1)
		for j, cell := range row.Cells {
			r.out.WriteString("\n│ ")
			if j < len(labels) && labels[j] != "" {
				r.out.WriteString(labels[j])
			} else {
				fmt.Fprintf(&r.out, "%d", j+1)
			}
			if cell.Colspan != nil && *cell.Colspan > 1 {
				fmt.Fprintf(&r.out, " [cols=%d]", *cell.Colspan)
			}
			if cell.Rowspan != nil && *cell.Rowspan > 1 {
				fmt.Fprintf(&r.out, " [rows=%d]", *cell.Rowspan)
			}
			r.out.WriteString(": ")
			nested := terminalRenderer{plainText: r.plainText, revealSpoilers: r.revealSpoilers}
			if cell.IsHeader {
				if err := nested.styled("1", cell.Children); err != nil {
					return err
				}
			} else if err := nested.rich(cell.Children); err != nil {
				return err
			}
			r.indent(nested.out.String(), "", "│    ")
		}
		r.out.WriteString("\n└─")
	}
	return nil
}

func (r *terminalRenderer) rich(nodes RichText) error {
	for _, node := range nodes {
		if err := r.richNode(node); err != nil {
			return err
		}
	}
	return nil
}

func (r *terminalRenderer) richNode(node any) error {
	switch n := node.(type) {
	case Text:
		r.text(string(n))
	case Bold:
		return r.styled("1", n.Children)
	case Italic:
		return r.styled("3", n.Children)
	case Underline:
		return r.styled("4", n.Children)
	case Strikethrough:
		return r.styled("9", n.Children)
	case Marked:
		return r.styled("30;43", n.Children)
	case Spoiler:
		if !r.revealSpoilers {
			r.out.WriteString("[spoiler]")
			return nil
		}
		return r.rich(n.Children)
	case Subscript:
		r.out.WriteString("_(")
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteByte(')')
	case Superscript:
		r.out.WriteString("^(")
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteByte(')')
	case Link:
		if err := r.styled("4;36", n.Children); err != nil {
			return err
		}
		r.out.WriteString(" <")
		r.text(string(n.Href))
		r.out.WriteByte('>')
	case AnchorLink:
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString(" [#")
		r.text(string(n.AnchorName))
		r.out.WriteByte(']')
	case Reference:
		return r.rich(n.Children)
	case ReferenceLink:
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString(" [ref: ")
		r.text(string(n.ReferenceName))
		r.out.WriteByte(']')
	case Code:
		r.out.WriteByte('`')
		r.text(string(n.Text))
		r.out.WriteByte('`')
	case Math:
		r.out.WriteString("⟦")
		r.text(string(n.Expression))
		r.out.WriteString("⟧")
	case Icon:
		r.out.WriteByte('[')
		if n.AlternativeText != nil {
			r.text(string(*n.AlternativeText))
		} else {
			r.out.WriteString("icon: ")
			r.text(string(n.Src))
		}
		r.out.WriteByte(']')
	case DateTime:
		location, err := time.LoadLocation(string(n.Timezone))
		if err != nil {
			return r.fallback(n, false)
		}
		r.text(time.Unix(n.Unix, 0).In(location).Format("2006-01-02 15:04:05 MST"))
	default:
		return r.fallback(node, false)
	}
	return nil
}

func (r *terminalRenderer) fallback(node any, block bool) error {
	data, err := Marshal(node)
	if err != nil {
		return err
	}
	if block {
		return r.block(NewPreformatted(Text(data)))
	}
	return r.richNode(NewCode(Text(data)))
}
