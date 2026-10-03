package tmark

import (
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshalRoundTrip(t *testing.T) {
	falseValue := false
	tests := []any{
		NewHeader(3, T("x")),
		P(T("plain ; "), B(T("bold #{}"))),
		P(T("\n")),
		NewLink("https://example.com", T("x")),
		NewCaption(T("x")),
		NewImage("/file/x.jpg").WithCaption(NewCaption(T("caption")).WithCredit(T("credit"))).WithSpoiler(),
		NewVideo("/video.mp4", "/preview.jpg"),
		NewVideo("/video.mp4", "/preview.jpg").WithLoop(),
		NewDateTime(1, "UTC"),
		NewMap(52.52, -0.125),
		NewIcon("/file/x.png").WithAlternativeText("x"),
		NewList(NewListItem(P(T("x"))).WithChecked(falseValue)),
		NewTable(NewTableRow(
			NewCell(T("A")).Header().WithAlign(TableCellAlignLeft),
			NewCell(T("B")),
		)),
		NewTable(NewTableRow(NewCell(T("td;x")))),
		Document{AttachedMedia: []AttachedMedia{{Hash: "h", Content: NewRichBlocks(P(T("x")))}}, Content: NewRichBlocks(P(T("y")))},
		Document{URL: "u", Title: "t", Content: NewRichBlocks(P(T("x")))},
		NewRichBlocks(P(T("x"))),
		NewPreformatted("a\nb").WithLanguage("go"),
		NewPreformatted("\n").WithLanguage("go"),
	}

	for _, in := range tests {
		t.Run(reflect.TypeOf(in).Name(), func(t *testing.T) {
			data, err := Marshal(in)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			out := reflect.New(reflect.TypeOf(in))
			if err := Unmarshal(data, out.Interface()); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			got, err := Marshal(out.Elem().Interface())
			if err != nil {
				t.Fatalf("Marshal(unmarshaled) error = %v", err)
			}
			if string(got) != string(data) {
				t.Fatalf("round trip = %q, want %q", got, data)
			}
		})
	}
}

func TestUnmarshalRejectsInvalidSerialization(t *testing.T) {
	cases := []string{
		"{list;\n{li;{p;x}}\n}",
		"{table;\n{p;x}\n}",
		"{h;#s{0}x}",
		"{h;#s{1.0}x}",
		"{map;#lat{1.0}#lon{2}}",
		"{list;\n{#order{1.5}{p;x}}\n}",
		"{h;#s{2}x#s{3}}",
		"{a;#href{x}text#href{y}}",
		"{p;bad\\q}",
		"{p;x\ry}",
		"{p;#BAD{x}}",
		"{video;/video.mp4}",
	}
	for _, data := range cases {
		t.Run(data, func(t *testing.T) {
			var got any
			if err := Unmarshal([]byte(data), &got); err == nil {
				t.Fatalf("accepted %q", data)
			}
		})
	}
}

func TestUnmarshalHeaderSize(t *testing.T) {
	var got Header
	if err := Unmarshal([]byte("{h;#s{2}x}"), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.Size != 2 {
		t.Fatalf("Header.Size = %d, want 2", got.Size)
	}
}

func TestUnmarshalRejectsUnknownField(t *testing.T) {
	var got Paragraph
	if err := Unmarshal([]byte("{p;#bad{x}x}"), &got); err == nil {
		t.Fatal("Unmarshal() error = nil, want unknown field error")
	}
}

func TestUnmarshalRejectsUnknownTagByDefault(t *testing.T) {
	var got RichText
	if err := Unmarshal([]byte("{x-unknown;value}"), &got); err == nil {
		t.Fatal("Unmarshal() error = nil, want unknown tag error")
	}
}

func TestUnmarshalSoftUnknownInline(t *testing.T) {
	data := []byte("{x-unknown;#attr{value}before {b;bold}{nested;after}}")
	var got RichText
	if err := DefaultUnmarshaler.Soft().Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	unknown, ok := got[0].(Unknown)
	if !ok {
		t.Fatalf("got[0] = %T, want Unknown", got[0])
	}
	if string(unknown.Raw) != string(data) {
		t.Fatalf("Unknown.Raw = %q, want %q", unknown.Raw, data)
	}
	out, err := Marshal(got)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(out) != string(data) {
		t.Fatalf("Marshal() = %q, want %q", out, data)
	}
}

func TestMarshalUnknownValidatesRaw(t *testing.T) {
	if _, err := Marshal(Unknown{Raw: []byte("{x;")}); err == nil {
		t.Fatal("Marshal(Unknown) error = nil, want validation error")
	}
}

func TestUnmarshalSoftUnknownBlock(t *testing.T) {
	data := []byte("{x-block;#meta{{p;text}}body}")
	var got RichBlocks
	if err := DefaultUnmarshaler.Soft().Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if _, ok := got[0].(Unknown); !ok {
		t.Fatalf("got[0] = %T, want Unknown", got[0])
	}
	out, err := Marshal(got)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(out) != string(data) {
		t.Fatalf("Marshal() = %q, want %q", out, data)
	}
}

func TestUnmarshalMaxDepth(t *testing.T) {
	var b strings.Builder
	for i := 0; i < maxDepth+1; i++ {
		b.WriteString("{b;")
	}
	b.WriteString("x")
	for i := 0; i < maxDepth+1; i++ {
		b.WriteByte('}')
	}

	var got RichText
	if err := Unmarshal([]byte(b.String()), &got); err == nil {
		t.Fatal("Unmarshal(deep node) error = nil, want max depth error")
	}
}
