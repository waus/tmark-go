package tmark

import "testing"

func TestMarshal(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"header", NewHeader(3, T("x")), "{h;#s{3}x}"},
		{"paragraph", P(T("x")), "{p;x}"},
		{"link", NewLink("https://example.com", T("x")), "{a;#href{https://example.com}x}"},
		{"pull quote", NewPullQuote(T("x")), "{as;x}"},
		{"list", NewList(NewListItem(P(T("x"))).WithChecked(false)), "{list;\n{#checked{f}{p;x}}\n}"},
		{"table", NewTable(NewTableRow(NewCell(T("x")))), "{table;\n{{x}}\n}"},
		{"collage", NewCollage(NewImage("a"), NewImage("b")), "{collage;\n{img;a}\n{img;b}\n}"},
		{"slideshow", NewSlideshow(NewImage("a"), NewVideo("b")), "{slideshow;\n{img;a}\n{video;b}\n}"},
		{"bool", NewImage("/file/x.jpg").WithSpoiler(), "{img;#has_spoiler{t}/file/x.jpg}"},
		{"caption", NewImage("/file/x.jpg").WithCaption(NewCaption(T("x"))), "{img;#caption{x}/file/x.jpg}"},
		{"caption credit", NewImage("x").WithCaption(NewCaption(T("song")).WithCredit(T("author"))), "{img;#caption{#credit{author}song}x}"},
		{"attached array", Document{AttachedMedia: []AttachedMedia{{Hash: "h", Content: NewRichBlocks(P(T("x")))}}, Content: NewRichBlocks(P(T("y")))}, "{document;\n#attached_media{{\n#hash{h}\n{p;x}\n}}\n{p;y}\n}"},
		{"datetime", NewDateTime(1, "UTC"), "{datetime;#timezone{UTC}1}"},
		{"map", NewMap(52.52, -0.125), "{map;#lat{52.52}#lon{-0.125}}"},
		{"icon", NewIcon("/file/x.png").WithAlternativeText("x"), "{icon;#alt{x}/file/x.png}"},
		{"anchor", NewAnchor("x"), "{anchor;x}"},
		{"divider", NewDivider(), "{hr;}"},
		{"blocks", NewRichBlocks(P(T("x"))), "{p;x}"},
		{"document", Document{URL: "u", Title: "t", Content: NewRichBlocks(P(T("x")))}, "{document;\n#url{u}\n#title{t}\n{p;x}\n}"},
		{"attached media", AttachedMedia{Hash: "h", Content: NewRichBlocks(P(T("x")))}, "{attached;\n#hash{h}\n{p;x}\n}"},
		{"semicolon text", P(T(";")), "{p;;}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.in)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Marshal() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarshalCaptionAsNode(t *testing.T) {
	got, err := Marshal(NewCaption(T("x")))
	if err != nil || string(got) != "{caption;x}" {
		t.Fatalf("Marshal(Caption) = %q, %v", got, err)
	}
}

func TestNewHeaderPreservesSizeForValidation(t *testing.T) {
	if _, err := Marshal(NewHeader(7, T("x"))); err == nil {
		t.Fatal("Marshal(NewHeader(7)) error = nil, want invalid size error")
	}
}

func TestMarshalMaxDepth(t *testing.T) {
	var node richText = T("x")
	for i := 0; i < maxDepth+1; i++ {
		node = NewBold(node)
	}
	if _, err := Marshal(node); err == nil {
		t.Fatal("Marshal(deep node) error = nil, want max depth error")
	}
}
