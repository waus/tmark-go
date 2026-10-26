package tmark

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTerminalLayout(t *testing.T) {
	content := NewRichBlocks(
		NewHeader(2, NewText("Title")),
		NewBlockquote(NewParagraph(NewText("one\ntwo"))).WithCredit(NewText("Author")),
		NewList(NewListItem(NewParagraph(NewText("task")), NewList(NewListItem(NewParagraph(NewText("child"))))).WithOrder(10).WithChecked(true), NewListItem(NewParagraph(NewText("next")))),
		NewPreformatted("a\n\tb").WithLanguage("go"),
		NewDetails(NewParagraph(NewText("body"))).WithSummary(NewText("More")),
		NewTable(NewTableRow(NewCell(NewText("Name")).Header().Col(2)), NewTableRow(NewCell(NewText("a\nb")).Row(2))).WithCaption(NewText("Data")),
	)
	got, err := ToTerminal(content, true, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "## Title\n\n│ one\n│ two\n│ \n│ — Author\n\n10. [x] task\n        \n        • child\n11. next\n\n┌─ code go\n│ a\n│     b\n└─\n\n▾ More\n  body\n\nData\n┌─ row 1\n│ 1 [cols=2]: Name\n└─\n┌─ row 2\n│ 1 [rows=2]: a\n│    b\n└─"
	if string(got) != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestTerminalNestedStyles(t *testing.T) {
	content := NewRichBlocks(NewParagraph(NewBold(NewText("a"), NewItalic(NewText("b")), NewText("c")), NewText("d")))
	got, err := ToTerminal(content, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "\x1b[1ma\x1b[3mb\x1b[0m\x1b[1mc\x1b[0md"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	plain, err := ToTerminal(content, true, false)
	if err != nil || string(plain) != "abcd" {
		t.Fatalf("plain = %q, %v", plain, err)
	}
}

func TestTerminalTableLabels(t *testing.T) {
	got, err := ToTerminal(NewRichBlocks(NewTable(
		NewTableRow(NewCell(NewBold(NewText("Full\nname"))).Header(), NewCell(NewText("")).Header()),
		NewTableRow(NewCell(NewText("Alice")), NewCell(NewText("42"))),
	)), false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "┌─ row 1\n│ Full name: Alice\n│ 2: 42\n└─"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTerminalSpoilers(t *testing.T) {
	content := NewRichBlocks(
		NewParagraph(NewText("before "), NewSpoiler(NewBold(NewText("secret text"))), NewText(" after")),
		NewCollage(NewImage("secret.png").WithSpoiler().WithCaption(NewCaption(NewText("secret caption")))),
		NewSlideshow(NewVideo("secret.mp4", "secret-preview.png").WithSpoiler()),
		NewTable(NewTableRow(NewCell(NewSpoiler(NewText("secret heading"))).Header()), NewTableRow(NewCell(NewText("value")))),
	)
	got, err := ToTerminal(content, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "secret") || strings.Count(string(got), "spoiler") != 4 {
		t.Fatalf("spoiler leaked: %s", got)
	}
	got, err = ToTerminal(content, true, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"secret text", "secret.png", "secret caption", "secret.mp4", "secret heading"} {
		if !strings.Contains(string(got), value) {
			t.Errorf("missing %q: %s", value, got)
		}
	}
}

func TestTerminalRejectsInputControls(t *testing.T) {
	attack := Text("safe\x1b]52;c;payload\a\x1b[2J\r\b\u009b31m")
	content := NewRichBlocks(
		NewHeader(1, attack), NewParagraph(attack, NewLink(attack, attack), NewIcon(attack).WithAlternativeText(attack)),
		NewPreformatted(attack).WithLanguage(attack), NewImage(attack).WithCaption(NewCaption(attack).WithCredit(attack)),
		NewTable(NewTableRow(NewCell(attack))),
		NewDetails(NewParagraph(attack)).WithSummary(attack),
		Unknown{Raw: []byte("{future;safe\x1b]52;c;payload\a\u009b31m}")},
	)
	got, err := ToTerminal(content, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(string(got), "\x1b\a\r\b\u009b") {
		t.Fatalf("unsafe controls: %q", got)
	}
}

func TestTerminalFallback(t *testing.T) {
	for _, content := range []RichBlocks{
		NewRichBlocks(Unknown{Raw: []byte("{future;x}")}),
		NewRichBlocks(NewTable(NewTableRow(NewCell(Unknown{Raw: []byte("{future;x}")})))),
		NewRichBlocks(NewTable(NewTableRow(NewCell(Unknown{Raw: []byte("{future;x}")}).Header()), NewTableRow(NewCell(NewText("value"))))),
	} {
		got, err := ToTerminal(content, false, false)
		if err != nil || !strings.Contains(string(got), "{future;x}") {
			t.Fatalf("fallback = %q, %v", got, err)
		}
	}
	got, err := ToTerminal(NewRichBlocks(NewParagraph(NewDateTime(0, "Unknown/Zone"))), false, false)
	if err != nil || string(got) != "`{datetime;#timezone{Unknown/Zone}0}`" {
		t.Fatalf("date fallback = %q, %v", got, err)
	}
}

func TestTerminalExamples(t *testing.T) {
	paths, err := filepath.Glob("../tmark/examples/*.tmark")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no example documents")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc Document
			if err := DefaultUnmarshaler.Soft().Unmarshal(data, &doc); err != nil {
				t.Fatal(err)
			}
			for _, plainText := range []bool{false, true} {
				for _, revealSpoilers := range []bool{false, true} {
					if _, err := ToTerminal(doc.Content, plainText, revealSpoilers); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
}

func TestTerminalLinksMathAndDates(t *testing.T) {
	got, err := ToTerminal(NewRichBlocks(NewParagraph(
		NewLink("https://example.com", NewText("site")), NewText(" "), NewAnchorLink("a", NewText("jump")), NewText(" "),
		NewReference("r", NewText("ref")), NewReferenceLink("r", NewText("back")),
		NewSubscript(NewText("2")), NewSuperscript(NewText("3")), NewMath("x+y"), NewCode("x"), NewDateTime(0, "UTC"),
	), NewMap(1, 2).WithCaption(NewCaption(NewText("place")))), true, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "site <https://example.com> jump [#a] refback [ref: r]_(2)^(3)⟦x+y⟧`x`1970-01-01 00:00:00 UTC\n\n[map] 1.000000, 2.000000\nhttps://www.openstreetmap.org/?mlat=1&mlon=2\nplace"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
