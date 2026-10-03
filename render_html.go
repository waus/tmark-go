package tmark

import (
	"bytes"
	"fmt"
	"html"
	"strconv"
	"strings"
)

func ToHTML(content RichBlocks, options ...RenderOption) ([]byte, error) {
	renderer := htmlRenderer{cfg: newRenderConfig(options)}
	if err := renderer.blocks(content); err != nil {
		return nil, err
	}
	return renderer.out.Bytes(), nil
}

type htmlRenderer struct {
	cfg renderConfig
	out bytes.Buffer
}

func (r *htmlRenderer) blocks(blocks RichBlocks) error {
	for _, block := range blocks {
		if err := r.block(block); err != nil {
			return err
		}
	}
	return nil
}

func (r *htmlRenderer) block(node any) error {
	switch n := node.(type) {
	case Paragraph:
		r.out.WriteString("<p>")
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("</p>\n")
	case Header:
		level := n.Size
		if level < 1 || level > 6 {
			level = 4
		}
		fmt.Fprintf(&r.out, "<h%d>", level)
		if err := r.rich(n.Children); err != nil {
			return err
		}
		fmt.Fprintf(&r.out, "</h%d>\n", level)
	case Blockquote:
		r.out.WriteString("<blockquote>\n")
		if err := r.blocks(n.Blocks); err != nil {
			return err
		}
		if len(n.Credit) > 0 {
			r.out.WriteString("<cite>")
			if err := r.rich(n.Credit); err != nil {
				return err
			}
			r.out.WriteString("</cite>\n")
		}
		r.out.WriteString("</blockquote>\n")
	case PullQuote:
		r.out.WriteString("<aside class=\"pull-quote\"><blockquote>")
		if err := r.rich(n.Text); err != nil {
			return err
		}
		if len(n.Credit) > 0 {
			r.out.WriteString("<cite>")
			if err := r.rich(n.Credit); err != nil {
				return err
			}
			r.out.WriteString("</cite>")
		}
		r.out.WriteString("</blockquote></aside>\n")
	case List:
		return r.list(n)
	case Divider:
		r.out.WriteString("<hr>\n")
	case Preformatted:
		r.out.WriteString("<pre><code")
		if n.Language != nil {
			r.out.WriteString(" class=\"language-")
			r.out.WriteString(attr(string(*n.Language)))
			r.out.WriteByte('"')
		}
		r.out.WriteByte('>')
		r.out.WriteString(html.EscapeString(string(n.Text)))
		r.out.WriteString("</code></pre>\n")
	case MathBlock:
		r.out.WriteString("<div class=\"math\">")
		r.out.WriteString(html.EscapeString(string(n.Expression)))
		r.out.WriteString("</div>\n")
	case Anchor:
		r.out.WriteString("<a id=\"")
		r.out.WriteString(attr(string(n.Name)))
		r.out.WriteString("\"></a>\n")
	case Map:
		r.out.WriteString("<figure class=\"map\" data-lat=\"")
		r.out.WriteString(strconv.FormatFloat(n.Lat, 'f', -1, 64))
		r.out.WriteString("\" data-lon=\"")
		r.out.WriteString(strconv.FormatFloat(n.Lon, 'f', -1, 64))
		r.out.WriteByte('"')
		if n.Zoom != nil {
			r.out.WriteString(" data-zoom=\"")
			r.out.WriteString(strconv.Itoa(*n.Zoom))
			r.out.WriteByte('"')
		}
		r.out.WriteString("></figure>\n")
		return r.caption(n.Caption)
	case Image:
		return r.mediaFigure("img", string(n.Src), "", false, n.Caption)
	case Video:
		return r.mediaFigure("video", string(n.Src), string(n.Preview), n.Loop, n.Caption)
	case Audio:
		return r.mediaFigure("audio", string(n.Src), "", false, n.Caption)
	case Collage:
		return r.slides("collage", n.Children, n.Caption)
	case Slideshow:
		return r.slides("slideshow", n.Children, n.Caption)
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
		r.out.WriteString("</summary>\n")
		if err := r.blocks(n.Children); err != nil {
			return err
		}
		r.out.WriteString("</details>\n")
	default:
		return r.unsupported(fmt.Sprintf("%T", node), "unsupported block node")
	}
	return nil
}

func (r *htmlRenderer) list(list List) error {
	ordered := false
	for _, item := range list.Items {
		if item.Order != nil {
			ordered = true
			break
		}
	}
	tag := "ul"
	if ordered {
		tag = "ol"
	}
	fmt.Fprintf(&r.out, "<%s>\n", tag)
	for _, item := range list.Items {
		r.out.WriteString("<li>")
		if item.Checked != nil {
			r.out.WriteString("<input type=\"checkbox\" disabled")
			if *item.Checked {
				r.out.WriteString(" checked")
			}
			r.out.WriteString("> ")
		}
		r.out.WriteByte('\n')
		if err := r.blocks(item.Children); err != nil {
			return err
		}
		r.out.WriteString("</li>\n")
	}
	fmt.Fprintf(&r.out, "</%s>\n", tag)
	return nil
}

func (r *htmlRenderer) mediaFigure(tag, src, preview string, loop bool, caption *Caption) error {
	r.out.WriteString("<figure>")
	switch tag {
	case "img":
		r.out.WriteString("<img src=\"")
		r.out.WriteString(attr(src))
		r.out.WriteString("\" alt=\"\">")
	case "video", "audio":
		r.out.WriteByte('<')
		r.out.WriteString(tag)
		r.out.WriteString(" controls src=\"")
		r.out.WriteString(attr(src))
		if tag == "video" {
			r.out.WriteString("\" poster=\"")
			r.out.WriteString(attr(preview))
		}
		r.out.WriteByte('"')
		if loop {
			r.out.WriteString(" loop")
		}
		r.out.WriteString("></")
		r.out.WriteString(tag)
		r.out.WriteByte('>')
	}
	if err := r.captionInner(caption); err != nil {
		return err
	}
	r.out.WriteString("</figure>\n")
	return nil
}

func (r *htmlRenderer) slides(class string, children []slideshowContent, caption *Caption) error {
	r.out.WriteString("<figure class=\"")
	r.out.WriteString(class)
	r.out.WriteString("\">\n")
	for _, child := range children {
		switch n := child.(type) {
		case Image:
			if err := r.mediaFigure("img", string(n.Src), "", false, n.Caption); err != nil {
				return err
			}
		case Video:
			if err := r.mediaFigure("video", string(n.Src), string(n.Preview), n.Loop, n.Caption); err != nil {
				return err
			}
		default:
			return r.unsupported(fmt.Sprintf("%T", child), "unsupported slideshow node")
		}
	}
	if err := r.captionInner(caption); err != nil {
		return err
	}
	r.out.WriteString("</figure>\n")
	return nil
}

func (r *htmlRenderer) table(table Table) error {
	r.out.WriteString("<table>\n")
	if len(table.Caption) > 0 {
		r.out.WriteString("<caption>")
		if err := r.rich(table.Caption); err != nil {
			return err
		}
		r.out.WriteString("</caption>\n")
	}
	for _, row := range table.Rows {
		r.out.WriteString("<tr>")
		for _, cell := range row.Cells {
			tag := "td"
			if cell.IsHeader {
				tag = "th"
			}
			r.out.WriteByte('<')
			r.out.WriteString(tag)
			if cell.Align != nil {
				r.out.WriteString(" align=\"")
				r.out.WriteString(attr(string(*cell.Align)))
				r.out.WriteByte('"')
			}
			if cell.Colspan != nil {
				r.out.WriteString(" colspan=\"")
				r.out.WriteString(strconv.Itoa(*cell.Colspan))
				r.out.WriteByte('"')
			}
			if cell.Rowspan != nil {
				r.out.WriteString(" rowspan=\"")
				r.out.WriteString(strconv.Itoa(*cell.Rowspan))
				r.out.WriteByte('"')
			}
			r.out.WriteByte('>')
			if err := r.rich(cell.Children); err != nil {
				return err
			}
			r.out.WriteString("</")
			r.out.WriteString(tag)
			r.out.WriteByte('>')
		}
		r.out.WriteString("</tr>\n")
	}
	r.out.WriteString("</table>\n")
	return nil
}

func (r *htmlRenderer) caption(caption *Caption) error {
	if caption == nil {
		return nil
	}
	r.out.WriteString("<figcaption>")
	if err := r.rich(caption.Text); err != nil {
		return err
	}
	if len(caption.Credit) > 0 {
		r.out.WriteString("<cite>")
		if err := r.rich(caption.Credit); err != nil {
			return err
		}
		r.out.WriteString("</cite>")
	}
	r.out.WriteString("</figcaption>\n")
	return nil
}

func (r *htmlRenderer) captionInner(caption *Caption) error {
	if caption == nil {
		return nil
	}
	r.out.WriteString("<figcaption>")
	if err := r.rich(caption.Text); err != nil {
		return err
	}
	if len(caption.Credit) > 0 {
		r.out.WriteString(" <cite>")
		if err := r.rich(caption.Credit); err != nil {
			return err
		}
		r.out.WriteString("</cite>")
	}
	r.out.WriteString("</figcaption>")
	return nil
}

func (r *htmlRenderer) rich(nodes RichText) error {
	for _, node := range nodes {
		if err := r.richNode(node); err != nil {
			return err
		}
	}
	return nil
}

func (r *htmlRenderer) richNode(node any) error {
	switch n := node.(type) {
	case Text:
		parts := strings.Split(string(n), "\n")
		for i, part := range parts {
			if i > 0 {
				r.out.WriteString("<br>\n")
			}
			r.out.WriteString(html.EscapeString(part))
		}
	case Link:
		r.out.WriteString("<a href=\"")
		r.out.WriteString(attr(string(n.Href)))
		r.out.WriteString("\">")
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("</a>")
	case AnchorLink:
		r.out.WriteString("<a href=\"#")
		r.out.WriteString(attr(string(n.AnchorName)))
		r.out.WriteString("\">")
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("</a>")
	case Reference:
		r.out.WriteString("<span id=\"")
		r.out.WriteString(attr(string(n.Name)))
		r.out.WriteString("\">")
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("</span>")
	case ReferenceLink:
		r.out.WriteString("<a href=\"#")
		r.out.WriteString(attr(string(n.ReferenceName)))
		r.out.WriteString("\">")
		if err := r.rich(n.Children); err != nil {
			return err
		}
		r.out.WriteString("</a>")
	case Bold:
		return r.wrap("strong", n.Children)
	case Italic:
		return r.wrap("em", n.Children)
	case Marked:
		return r.wrap("mark", n.Children)
	case Underline:
		return r.wrap("u", n.Children)
	case Strikethrough:
		return r.wrap("s", n.Children)
	case Spoiler:
		return r.wrapClass("span", "spoiler", n.Children)
	case Subscript:
		return r.wrap("sub", n.Children)
	case Superscript:
		return r.wrap("sup", n.Children)
	case DateTime:
		r.out.WriteString("<time data-unix=\"")
		r.out.WriteString(strconv.FormatInt(n.Unix, 10))
		r.out.WriteString("\" data-timezone=\"")
		r.out.WriteString(attr(string(n.Timezone)))
		r.out.WriteString("\">")
		r.out.WriteString(strconv.FormatInt(n.Unix, 10))
		r.out.WriteString("</time>")
	case Code:
		r.out.WriteString("<code>")
		r.out.WriteString(html.EscapeString(string(n.Text)))
		r.out.WriteString("</code>")
	case Math:
		r.out.WriteString("<span class=\"math\">")
		r.out.WriteString(html.EscapeString(string(n.Expression)))
		r.out.WriteString("</span>")
	case Icon:
		r.out.WriteString("<img src=\"")
		r.out.WriteString(attr(string(n.Src)))
		r.out.WriteString("\" alt=\"")
		if n.AlternativeText != nil {
			r.out.WriteString(attr(string(*n.AlternativeText)))
		}
		r.out.WriteString("\">")
	default:
		return r.unsupported(fmt.Sprintf("%T", node), "unsupported rich text node")
	}
	return nil
}

func (r *htmlRenderer) wrap(tag string, children RichText) error {
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

func (r *htmlRenderer) wrapClass(tag, class string, children RichText) error {
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

func (r *htmlRenderer) unsupported(tag string, reason string) error {
	if !r.cfg.soft {
		return unsupportedRenderTag(tag, reason)
	}
	r.out.WriteString(html.EscapeString(r.cfg.placeholder))
	return nil
}

func attr(value string) string {
	return html.EscapeString(value)
}
