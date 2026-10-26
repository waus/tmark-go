package tmark

import "testing"

func TestMarkdownGFM(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content RichBlocks
		want    string
	}{
		{
			"literal syntax",
			NewRichBlocks(NewParagraph(NewText("1. text\n- item\n+ item\n===\n`code` ~~strike~~ ![image](x) &copy;"))),
			"1\\. text  \n\\- item  \n\\+ item  \n\\=\\=\\=  \n\\`code\\` \\~\\~strike\\~\\~ \\!\\[image\\](x\\) \\&copy;",
		},
		{
			"link destination",
			NewRichBlocks(NewParagraph(NewLink("https://example.com/a (b)?a=1&copy;=2", NewText("link")))),
			"[link](https://example.com/a%20%28b%29?a=1&amp;copy;=2)",
		},
		{
			"code spaces and backticks",
			NewRichBlocks(NewParagraph(NewCode(" code "), NewText(" "), NewCode("`code`"), NewText(" "), NewCode(""))),
			"`  code  ` `` `code` `` <code></code>",
		},
		{
			"formatting whitespace",
			NewRichBlocks(NewParagraph(NewText("a"), NewBold(NewText(" bold ")), NewItalic(NewText(" italic ")), NewStrikethrough(NewText(" old ")))),
			"a **bold**  *italic*  ~~old~~ ",
		},
		{
			"math and highlight",
			NewRichBlocks(NewParagraph(NewMarked(NewText("bright")), NewText(" "), NewMath("x < y")), NewMathBlock("x = y\n```")),
			"<mark>bright</mark> `x < y`\n\n````\nx = y\n```\n````",
		},
		{
			"ordered nesting and line breaks",
			NewRichBlocks(NewList(
				NewListItem(NewParagraph(NewText("first\nsecond")), NewList(NewListItem(NewParagraph(NewText("child"))))).WithOrder(10),
				NewListItem(NewParagraph(NewText("next"))),
			)),
			"10. first  \n    second\n\n    - child\n11. next",
		},
		{
			"tasks with nested blocks",
			NewRichBlocks(NewList(NewListItem(NewParagraph(NewText("task")), NewParagraph(NewText("more"))).WithChecked(true))),
			"- [x] task\n\n  more",
		},
		{
			"separate lists",
			NewRichBlocks(NewList(NewListItem(NewParagraph(NewText("one")))), NewList(NewListItem(NewParagraph(NewText("two"))))),
			"- one\n\n<!-- -->\n\n- two",
		},
		{
			"table pipes and breaks",
			NewRichBlocks(NewTable(
				NewTableRow(NewCell(NewText("a|b")).Header().WithAlign(TableCellAlignCenter)),
				NewTableRow(NewCell(NewCode("x|y"), NewText("\nsecond"))),
			)),
			"| a\\|b |\n| :---: |\n| `x\\|y`<br>second |",
		},
		{
			"headerless ragged table",
			NewRichBlocks(NewTable(NewTableRow(NewCell(NewText("first"))), NewTableRow(NewCell(NewText("second")), NewCell(NewText("extra"))))),
			"|  |  |\n| --- | --- |\n| first |  |\n| second | extra |",
		},
		{
			"merged table cells",
			NewRichBlocks(NewTable(NewTableRow(NewCell(NewText("wide")).Col(2)))),
			"<table>\n<tr><td colspan=\"2\">wide</td></tr>\n</table>",
		},
		{
			"details summary",
			NewRichBlocks(NewDetails(NewParagraph(NewBold(NewText("body")))).WithSummary(NewBold(NewText("title")))),
			"<details>\n<summary><strong>title</strong></summary>\n\n**body**\n\n</details>",
		},
		{
			"quote credit after list",
			NewRichBlocks(NewBlockquote(NewList(NewListItem(NewParagraph(NewText("item"))))).WithCredit(NewText("Author"))),
			"> - item\n> \n> — Author",
		},
		{
			"multiline pull quote",
			NewRichBlocks(NewPullQuote(NewText("one\ntwo"))),
			"> one  \n> two",
		},
		{
			"media captions",
			NewRichBlocks(NewSlideshow(
				NewImage("/a.png").WithCaption(NewCaption(NewText("View "), NewItalic(NewText("nice"))).WithCredit(NewText("by"))),
				NewImage("/b.png").WithCaption(NewCaption().WithCredit(NewText("author"))),
			)),
			"![](/a.png)\n\n_View *nice*_  \n_by_\n\n![](/b.png)\n\n_author_",
		},
		{
			"date timezone",
			NewRichBlocks(NewParagraph(NewDateTime(1625097600, "America/New_York"))),
			"2021\\-06\\-30 20:00:00 EDT",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ToMarkdown(tc.content, false)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("ToMarkdown() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMarkdownFallback(t *testing.T) {
	got, err := ToMarkdown(NewRichBlocks(
		NewParagraph(NewText("before "), Unknown{Raw: []byte("{future;x}")}, NewText(" after")),
		NewMap(0, 0),
		NewParagraph(NewText("next")),
	), false)
	if err != nil {
		t.Fatal(err)
	}
	want := "before `{future;x}` after\n\n```\n{map;#lat{0}#lon{0}}\n```\n\nnext"
	if string(got) != want {
		t.Fatalf("ToMarkdown() = %q, want %q", got, want)
	}
}

func TestMarkdownHTMLLess(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content RichBlocks
		want    string
	}{
		{
			"inline styles",
			NewRichBlocks(NewParagraph(
				NewMarked(NewBold(NewText("marked"))), NewText(" "),
				NewUnderline(NewText("underlined")), NewText(" "),
				NewSpoiler(NewItalic(NewText("spoiler"))), NewText(" "),
				NewSubscript(NewText("sub")), NewText(" "), NewSuperscript(NewText("sup")),
			)),
			"**marked** underlined *spoiler* sub sup",
		},
		{
			"anchor and reference labels",
			NewRichBlocks(NewParagraph(
				NewReference("ref", NewBold(NewText("reference"))), NewText(" "),
				NewReferenceLink("ref", NewText("reference link")), NewText(" "),
				NewAnchorLink("anchor", NewText("anchor link")),
			), NewAnchor("anchor")),
			"**reference** reference link anchor link",
		},
		{
			"nested details",
			NewRichBlocks(NewDetails(
				NewParagraph(NewUnderline(NewText("visible"))),
				NewDetails(NewParagraph(NewText("body"))).WithSummary(NewMarked(NewText("inner"))),
			).WithSummary(NewText("outer"))),
			"**outer**\n\nvisible\n\n**inner**\n\nbody",
		},
		{
			"table spans and breaks",
			NewRichBlocks(NewTable(
				NewTableRow(NewCell(NewText("header")).Header().Col(2)),
				NewTableRow(NewCell(NewText("a|b\nnext"), NewSubscript(NewText("2"))).Row(2)),
			)),
			"| header |\n| --- |\n| a\\|b next2 |",
		},
		{
			"empty code and math",
			NewRichBlocks(NewParagraph(NewCode(""), NewMath(""), NewCode("x"))),
			"`x`",
		},
		{
			"adjacent lists",
			NewRichBlocks(NewList(NewListItem(NewParagraph(NewText("one")))), NewList(NewListItem(NewParagraph(NewText("two"))))),
			"- one\n\n---\n\n- two",
		},
		{
			"nested list and caption",
			NewRichBlocks(NewList(NewListItem(
				NewParagraph(NewSpoiler(NewText("task"))),
				NewImage("/image.png").WithCaption(NewCaption(NewMarked(NewText("caption"))).WithCredit(NewUnderline(NewText("author")))),
			).WithChecked(false))),
			"- [ ] task\n\n  ![](/image.png)\n\n  _caption_  \n  _author_",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ToMarkdown(tc.content, true)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("ToMarkdown() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMarkdownHTMLLessFallback(t *testing.T) {
	content := NewRichBlocks(NewDetails(NewMap(0, 0)).WithSummary(NewText("Map")))
	got, err := ToMarkdown(content, true)
	if err != nil {
		t.Fatal(err)
	}
	want := "**Map**\n\n```\n{map;#lat{0}#lon{0}}\n```"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
