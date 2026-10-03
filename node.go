package tmark

type Text string

func NewText(text string) Text { return Text(text) }

func (Text) IsRichText() {}

type RichText []richText

type RichBlocks []blockContent

type Caption struct {
	Credit RichText `tmark:"named"`
	Text   RichText `tmark:"unnamed"`
}

func NewCaption(text ...richText) Caption { return Caption{Text: text} }

func (c Caption) WithCredit(children ...richText) Caption {
	c.Credit = children
	return c
}

type Link struct {
	rich
	Href     Text     `tmark:"named"`
	Children RichText `tmark:"unnamed"`
}

func NewLink(href Text, children ...richText) Link {
	return Link{Href: href, Children: children}
}

type AnchorLink struct {
	rich
	AnchorName Text     `tmark:"named,name"`
	Children   RichText `tmark:"unnamed"`
}

func NewAnchorLink(anchorName Text, children ...richText) AnchorLink {
	return AnchorLink{AnchorName: anchorName, Children: children}
}

type Reference struct {
	rich
	Name     Text     `tmark:"named"`
	Children RichText `tmark:"unnamed"`
}

func NewReference(name Text, children ...richText) Reference {
	return Reference{Name: name, Children: children}
}

type ReferenceLink struct {
	rich
	ReferenceName Text     `tmark:"named,name"`
	Children      RichText `tmark:"unnamed"`
}

func NewReferenceLink(referenceName Text, children ...richText) ReferenceLink {
	return ReferenceLink{ReferenceName: referenceName, Children: children}
}

type Bold struct {
	rich
	Children RichText `tmark:"unnamed"`
}

func NewBold(children ...richText) Bold { return Bold{Children: children} }

type Italic struct {
	rich
	Children RichText `tmark:"unnamed"`
}

func NewItalic(children ...richText) Italic { return Italic{Children: children} }

type Marked struct {
	rich
	Children RichText `tmark:"unnamed"`
}

func NewMarked(children ...richText) Marked { return Marked{Children: children} }

type Underline struct {
	rich
	Children RichText `tmark:"unnamed"`
}

func NewUnderline(children ...richText) Underline { return Underline{Children: children} }

type Strikethrough struct {
	rich
	Children RichText `tmark:"unnamed"`
}

func NewStrikethrough(children ...richText) Strikethrough { return Strikethrough{Children: children} }

type Spoiler struct {
	rich
	Children RichText `tmark:"unnamed"`
}

func NewSpoiler(children ...richText) Spoiler { return Spoiler{Children: children} }

type Subscript struct {
	rich
	Children RichText `tmark:"unnamed"`
}

func NewSubscript(children ...richText) Subscript { return Subscript{Children: children} }

type Superscript struct {
	rich
	Children RichText `tmark:"unnamed"`
}

func NewSuperscript(children ...richText) Superscript { return Superscript{Children: children} }

type DateTime struct {
	rich
	Timezone Text  `tmark:"named"`
	Unix     int64 `tmark:"unnamed"`
}

func NewDateTime(unix int64, timezone Text) DateTime {
	return DateTime{Unix: unix, Timezone: timezone}
}

type Code struct {
	rich
	Text Text `tmark:"unnamed"`
}

func NewCode(text Text) Code { return Code{Text: text} }

type Math struct {
	rich
	Expression Text `tmark:"unnamed"`
}

func NewMath(expression Text) Math { return Math{Expression: expression} }

type Icon struct {
	rich
	AlternativeText *Text `tmark:"named,alt"`
	Src             Text  `tmark:"unnamed"`
}

func NewIcon(src Text) Icon { return Icon{Src: src} }
func (i Icon) WithAlternativeText(alternativeText Text) Icon {
	i.AlternativeText = &alternativeText
	return i
}

type Paragraph struct {
	block
	Children RichText `tmark:"unnamed"`
}

func NewParagraph(children ...richText) Paragraph { return Paragraph{Children: children} }

type Header struct {
	block
	Size     int      `tmark:"named,s"`
	Children RichText `tmark:"unnamed"`
}

func NewHeader(size int, children ...richText) Header {
	return Header{Size: size, Children: children}
}

type Preformatted struct {
	block
	Language *Text `tmark:"named"`
	Text     Text  `tmark:"unnamed"`
}

func NewPreformatted(text Text) Preformatted { return Preformatted{Text: text} }
func (p Preformatted) WithLanguage(language Text) Preformatted {
	p.Language = &language
	return p
}

type MathBlock struct {
	block
	Expression Text `tmark:"unnamed"`
}

func NewMathBlock(expression Text) MathBlock { return MathBlock{Expression: expression} }

type Anchor struct {
	block
	Name Text `tmark:"unnamed"`
}

func NewAnchor(name Text) Anchor { return Anchor{Name: name} }

type Divider struct {
	block
}

func NewDivider() Divider { return Divider{} }

type Blockquote struct {
	block
	Credit RichText   `tmark:"named"`
	Blocks RichBlocks `tmark:"unnamed"`
}

func NewBlockquote(blocks ...blockContent) Blockquote {
	return Blockquote{Blocks: blocks}
}
func (q Blockquote) WithCredit(children ...richText) Blockquote {
	q.Credit = children
	return q
}

type PullQuote struct {
	block
	Credit RichText `tmark:"named"`
	Text   RichText `tmark:"unnamed"`
}

func NewPullQuote(text ...richText) PullQuote { return PullQuote{Text: text} }
func (a PullQuote) WithCredit(children ...richText) PullQuote {
	a.Credit = children
	return a
}

type Collage struct {
	block
	Caption  *Caption           `tmark:"named"`
	Children []slideshowContent `tmark:"unnamed"`
}

func NewCollage(children ...slideshowContent) Collage {
	return Collage{Children: children}
}
func (c Collage) WithCaption(caption Caption) Collage {
	c.Caption = &caption
	return c
}

type List struct {
	block
	Items []ListItem `tmark:"unnamed"`
}

func NewList(items ...ListItem) List { return List{Items: items} }

type ListItem struct {
	Type     *Text      `tmark:"named"` //For ordered lists, the type of the item label; must be one of “a” for lowercase letters, “A” for uppercase letters, “i” for lowercase Roman numerals, “I” for uppercase Roman numerals, or “1” for decimal numbers
	Order    *int       `tmark:"named"`
	Checked  *bool      `tmark:"named"`
	Children RichBlocks `tmark:"unnamed"`
}

func NewListItem(children ...blockContent) ListItem { return ListItem{Children: children} }

func (li ListItem) WithType(typ Text) ListItem {
	li.Type = &typ
	return li
}

func (li ListItem) WithOrder(order int) ListItem {
	li.Order = &order
	return li
}

func (li ListItem) WithChecked(checked bool) ListItem {
	li.Checked = &checked
	return li
}

type Map struct {
	block
	Lat     float64  `tmark:"named"`
	Lon     float64  `tmark:"named"`
	Zoom    *int     `tmark:"named"`
	Caption *Caption `tmark:"named"`
}

func NewMap(lat, lon float64) Map { return Map{Lat: lat, Lon: lon} }
func (m Map) WithZoom(zoom int) Map {
	m.Zoom = &zoom
	return m
}
func (m Map) WithCaption(caption Caption) Map {
	m.Caption = &caption
	return m
}

type Image struct {
	block
	slideshow
	Caption    *Caption `tmark:"named"`
	HasSpoiler bool     `tmark:"named,has_spoiler"`
	Src        Text     `tmark:"unnamed"`
}

func NewImage(src Text) Image { return Image{Src: src} }

func (i Image) WithCaption(caption Caption) Image {
	i.Caption = &caption
	return i
}
func (i Image) WithSpoiler() Image {
	i.HasSpoiler = true
	return i
}

type Video struct {
	block
	slideshow
	Caption    *Caption `tmark:"named"`
	HasSpoiler bool     `tmark:"named,has_spoiler"`
	Loop       bool     `tmark:"named"`
	Preview    Text     `tmark:"named"`
	Src        Text     `tmark:"unnamed"`
}

func NewVideo(src, preview Text) Video { return Video{Src: src, Preview: preview} }

func (v Video) WithCaption(caption Caption) Video {
	v.Caption = &caption
	return v
}
func (v Video) WithSpoiler() Video {
	v.HasSpoiler = true
	return v
}
func (v Video) WithLoop() Video {
	v.Loop = true
	return v
}

type Audio struct {
	block
	Caption *Caption `tmark:"named"`
	Src     Text     `tmark:"unnamed"`
}

func NewAudio(src Text) Audio { return Audio{Src: src} }

func (a Audio) WithCaption(caption Caption) Audio {
	a.Caption = &caption
	return a
}

type Slideshow struct {
	block
	Caption  *Caption           `tmark:"named"`
	Children []slideshowContent `tmark:"unnamed"`
}

func NewSlideshow(children ...slideshowContent) Slideshow {
	return Slideshow{Children: children}
}
func (sl Slideshow) WithCaption(caption Caption) Slideshow {
	sl.Caption = &caption
	return sl
}

type Table struct {
	block
	Caption  RichText   `tmark:"named"`
	Bordered bool       `tmark:"named"`
	Striped  bool       `tmark:"named"`
	Rows     []TableRow `tmark:"unnamed"`
}

func NewTable(rows ...TableRow) Table { return Table{Rows: rows} }
func (t Table) WithCaption(children ...richText) Table {
	t.Caption = children
	return t
}
func (t Table) WithBordered() Table {
	t.Bordered = true
	return t
}
func (t Table) WithStriped() Table {
	t.Striped = true
	return t
}

type TableCellAlign string

const (
	TableCellAlignLeft   TableCellAlign = "left"
	TableCellAlignCenter TableCellAlign = "center"
	TableCellAlignRight  TableCellAlign = "right"
)

type TableCellValign string

const (
	TableCellValignTop    TableCellValign = "top"
	TableCellValignMiddle TableCellValign = "middle"
	TableCellValignBottom TableCellValign = "bottom"
)

type TableRow struct {
	block
	Cells []Cell `tmark:"unnamed"`
}

func NewTableRow(cells ...Cell) TableRow { return TableRow{Cells: cells} }

type Cell struct {
	block
	IsHeader bool             `tmark:"named,header"`
	Colspan  *int             `tmark:"named"`
	Rowspan  *int             `tmark:"named"`
	Align    *TableCellAlign  `tmark:"named"`
	Valign   *TableCellValign `tmark:"named"`
	Children RichText         `tmark:"unnamed"`
}

func NewCell(children ...richText) Cell { return Cell{Children: children} }

func (c Cell) Header() Cell {
	c.IsHeader = true
	return c
}

func (c Cell) Col(col int) Cell {
	c.Colspan = &col
	return c
}

func (c Cell) Row(row int) Cell {
	c.Rowspan = &row
	return c
}

func (c Cell) WithAlign(align TableCellAlign) Cell {
	c.Align = &align
	return c
}

func (c Cell) WithValign(valign TableCellValign) Cell {
	c.Valign = &valign
	return c
}

type Details struct {
	block
	Summary  RichText   `tmark:"named"`
	IsOpen   bool       `tmark:"named,open"`
	Children RichBlocks `tmark:"unnamed"`
}

func NewDetails(children ...blockContent) Details { return Details{Children: children} }

func (d Details) WithSummary(children ...richText) Details {
	d.Summary = children
	return d
}
func (d Details) Open() Details {
	d.IsOpen = true
	return d
}

func NewRichBlocks(children ...blockContent) RichBlocks { return children }
