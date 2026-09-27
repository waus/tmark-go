package tmark

import "testing"

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
	want := "| [unsupported] |\n| --- |"
	if string(got) != want {
		t.Fatalf("markdown mismatch\nwant: %q\n got: %q", want, got)
	}
}
