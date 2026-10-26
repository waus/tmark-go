package tmark

import (
	"strings"
	"testing"
)

func TestToHTML(t *testing.T) {
	input := NewRichBlocks(
		NewHeader(3, NewText("Title")),
		NewParagraph(NewText("Hello "), NewBold(NewText("world")), NewText(" "), NewCode(NewText("x"))),
	)
	output, err := ToHTML(input)
	if err != nil {
		t.Fatalf("to html: %v", err)
	}
	want := "<h3>Title</h3>\n<p>Hello <strong>world</strong> <code>x</code></p>\n"
	if string(output) != want {
		t.Fatalf("html mismatch\nwant: %q\n got: %q", want, output)
	}
}

func TestVideoHTMLPoster(t *testing.T) {
	got, err := ToHTML(NewRichBlocks(NewVideo("/video.mp4", "/preview.jpg")))
	if err != nil {
		t.Fatalf("ToHTML() error = %v", err)
	}
	want := "<figure><video controls src=\"/video.mp4\" poster=\"/preview.jpg\"></video></figure>\n"
	if string(got) != want {
		t.Fatalf("ToHTML() = %q, want %q", got, want)
	}
}

func TestVideoHTMLLoop(t *testing.T) {
	got, err := ToHTML(NewRichBlocks(NewVideo("/video.mp4", "/preview.jpg").WithLoop()))
	if err != nil {
		t.Fatalf("ToHTML() error = %v", err)
	}
	want := "<figure><video controls src=\"/video.mp4\" poster=\"/preview.jpg\" loop></video></figure>\n"
	if string(got) != want {
		t.Fatalf("ToHTML() = %q, want %q", got, want)
	}
}

func TestToMarkdown(t *testing.T) {
	input := NewRichBlocks(
		NewHeader(3, NewText("Title")),
		NewParagraph(NewText("Hello "), NewBold(NewText("world")), NewText(" "), NewCode(NewText("x"))),
		NewList(NewListItem(NewParagraph(NewText("one"))), NewListItem(NewParagraph(NewText("two")))),
	)
	output, err := ToMarkdown(input)
	if err != nil {
		t.Fatalf("to markdown: %v", err)
	}
	want := "### Title\n\nHello **world** `x`\n\n- one\n- two"
	if string(output) != want {
		t.Fatalf("markdown mismatch\nwant: %q\n got: %q", want, output)
	}
}

func TestMarkdownUnsupportedStrictAndSoft(t *testing.T) {
	content := NewRichBlocks(NewMap(1, 2))
	if _, err := ToMarkdown(content); err == nil {
		t.Fatal("ToMarkdown map strict succeeded, want error")
	}
	if _, err := ToMarkdown(content, WithSoftConversion()); err != nil {
		t.Fatalf("ToMarkdown map soft: %v", err)
	}
}

func TestMarkdownTableCellUnsupportedStrictAndSoft(t *testing.T) {
	content := NewRichBlocks(NewTable(NewTableRow(NewCell(Unknown{Raw: []byte("{x;v}")}))))
	if _, err := ToMarkdown(content); err == nil {
		t.Fatal("ToMarkdown table cell strict succeeded, want error")
	}
	got, err := ToMarkdown(content, WithSoftConversion())
	if err != nil {
		t.Fatalf("ToMarkdown table cell soft: %v", err)
	}
	want := "|  |\n| --- |\n| \\[unsupported\\] |"
	if string(got) != want {
		t.Fatalf("markdown mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestHTMLListNumbering(t *testing.T) {
	content := NewRichBlocks(NewList(
		NewListItem(NewParagraph(NewText("ten"))).WithOrder(10),
		NewListItem(NewParagraph(NewText("twenty-seven"))).WithOrder(27).WithType("A"),
		NewListItem(NewList(NewListItem(NewParagraph(NewText("nested"))).WithOrder(4).WithType("i"))),
	))
	got, err := ToHTML(content)
	if err != nil {
		t.Fatal(err)
	}
	want := "<ol>\n<li value=\"10\">\n<p>ten</p>\n</li>\n<li value=\"27\" type=\"A\">\n<p>twenty-seven</p>\n</li>\n<li>\n<ol>\n<li value=\"4\" type=\"i\">\n<p>nested</p>\n</li>\n</ol>\n</li>\n</ol>\n"
	if string(got) != want {
		t.Fatalf("ToHTML() = %q, want %q", got, want)
	}
}

func TestHTMLIcon(t *testing.T) {
	got, err := ToHTML(NewRichBlocks(NewParagraph(NewIcon("/icon.png").WithAlternativeText("<logo>"))))
	if err != nil {
		t.Fatal(err)
	}
	want := "<p><img src=\"/icon.png\" alt=\"&lt;logo&gt;\" style=\"height: 1em; width: auto; vertical-align: middle;\"></p>\n"
	if string(got) != want {
		t.Fatalf("ToHTML() = %q, want %q", got, want)
	}
}

func TestHTMLFallback(t *testing.T) {
	content := NewRichBlocks(
		NewParagraph(NewText("before "), NewMath("a<b"), NewText(" "), Unknown{Raw: []byte("{future;<img src=x>}")}, NewText(" after")),
		NewMathBlock("a<b\nc>d"),
		NewMap(1, 2).WithZoom(3).WithCaption(NewCaption(NewText("Map"))),
		Unknown{Raw: []byte("{future;\n<script>alert(1)</script>\n}")},
	)
	want := "<p>before <code>{math;a&lt;b}</code> <code>{future;&lt;img src=x&gt;}</code> after</p>\n" +
		"<pre><code>{math-block;\na&lt;b\nc&gt;d\n}</code></pre>\n" +
		"<pre><code>{map;#lat{1}#lon{2}#zoom{3}#caption{Map}}</code></pre>\n" +
		"<pre><code>{future;\n&lt;script&gt;alert(1)&lt;/script&gt;\n}</code></pre>\n"
	got, err := ToHTML(content)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() = %q, want %q", got, want)
	}
}

func TestHTMLFallbackRejectsInvalidNode(t *testing.T) {
	if _, err := ToHTML(NewRichBlocks(Unknown{Raw: []byte("{future;")})); err == nil {
		t.Fatal("ToHTML() accepted malformed fallback node")
	}
}

func TestHTMLDateTime(t *testing.T) {
	for _, tc := range []struct {
		name string
		node DateTime
		want string
	}{
		{"UTC", NewDateTime(0, "UTC"), "<p><time data-unix=\"0\" data-timezone=\"UTC\">1970-01-01 00:00:00 UTC</time></p>\n"},
		{"winter", NewDateTime(0, "America/New_York"), "<p><time data-unix=\"0\" data-timezone=\"America/New_York\">1969-12-31 19:00:00 EST</time></p>\n"},
		{"summer", NewDateTime(1625097600, "America/New_York"), "<p><time data-unix=\"1625097600\" data-timezone=\"America/New_York\">2021-06-30 20:00:00 EDT</time></p>\n"},
		{"negative", NewDateTime(-1, "UTC"), "<p><time data-unix=\"-1\" data-timezone=\"UTC\">1969-12-31 23:59:59 UTC</time></p>\n"},
		{"unknown timezone", NewDateTime(0, "Unknown/Zone"), "<p><code>{datetime;#timezone{Unknown/Zone}0}</code></p>\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ToHTML(NewRichBlocks(NewParagraph(tc.node)))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("ToHTML() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSpoilerMediaRendering(t *testing.T) {
	for _, tc := range []struct {
		name     string
		content  RichBlocks
		html     string
		markdown string
	}{
		{
			"image with caption",
			NewRichBlocks(NewImage("/image.png").WithSpoiler().WithCaption(NewCaption(NewText("Caption")))),
			"<figure class=\"tmark-spoiler\" tabindex=\"0\"><img src=\"/image.png\" alt=\"\"><figcaption>Caption</figcaption></figure>\n",
			"![](/image.png)\n\n_Caption_",
		},
		{
			"looped video",
			NewRichBlocks(NewVideo("/video.mp4", "/poster.jpg").WithSpoiler().WithLoop()),
			"<figure class=\"tmark-spoiler\" tabindex=\"0\"><video controls src=\"/video.mp4\" poster=\"/poster.jpg\" loop></video></figure>\n",
			"[video](/video.mp4)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ToHTML(tc.content)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.html {
				t.Fatalf("ToHTML() = %q, want %q", got, tc.html)
			}
			got, err = ToMarkdown(tc.content)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != strings.TrimSuffix(tc.html, "\n") {
				t.Fatalf("ToMarkdown() did not preserve spoiler HTML: %q", got)
			}
			got, err = ToMarkdown(tc.content, WithHTMLLess())
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.markdown {
				t.Fatalf("HTML-less ToMarkdown() = %q, want %q", got, tc.markdown)
			}
		})
	}
}

func TestSpoilersInGalleries(t *testing.T) {
	content := NewRichBlocks(
		NewCollage(NewImage("/image.png").WithSpoiler()),
		NewSlideshow(NewVideo("/video.mp4", "/poster.jpg").WithSpoiler()),
	)
	for name, render := range map[string]func(RichBlocks) ([]byte, error){
		"HTML":     ToHTML,
		"Markdown": func(content RichBlocks) ([]byte, error) { return ToMarkdown(content) },
	} {
		got, err := render(content)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(got), `class="tmark-spoiler"`) != 2 {
			t.Fatalf("%s lost gallery spoiler flags: %s", name, got)
		}
	}
}
