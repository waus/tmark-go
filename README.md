# tmark-go

Go package with a typed `tmark` model. More about `tmark` you can read on https://tmark.waus.app

```go
import "github.com/waus/tmark-go"
```

## Build Markup In Go

Documents can be built with constructors:

```go
description := tmark.Text("Example page")
document := tmark.Document{
    URL:         "https://example.com/example",
    Title:       "Example",
    Description: &description,
    Content: tmark.NewRichBlocks(
        tmark.NewHeader(3, tmark.NewText("Title")),
        tmark.NewParagraph(
            tmark.NewText("Hello "),
            tmark.NewBold(tmark.NewText("world")),
            tmark.NewText(". See "),
            tmark.NewLink("https://example.com", tmark.NewText("a link")),
            tmark.NewText("."),
        ),
        tmark.NewImage("/file/example.jpg").WithCaption(
            tmark.NewCaption(tmark.NewText("Caption")).
                WithCredit(tmark.NewText("Photo credit")),
        ),
        tmark.NewParagraph(tmark.NewText("Footer text")),
    ),
}
```

Allowed nesting is typed through internal marker interfaces, so invalid nesting
such as `NewParagraph(NewImage(...))` or
`NewList(NewParagraph(...))` does not compile.

Text must be passed explicitly with `NewText("...")`. Other constructors accept
typed children only, for example `NewBold(NewText("world"))` or
`NewHeader(3, NewText("Title"))`.

`Document.Content`, `ListItem`, and `Details` share the same common child model.
Footer-style text and link sections are represented as regular blocks.

## Rendering

`tmark` document content can be rendered directly to HTML, Markdown, or terminal text.

```go
html, err := tmark.ToHTML(document.Content)
markdown, err := tmark.ToMarkdown(document.Content, false)
terminal, err := tmark.ToTerminal(document.Content, false, false)
```

HTML renders unsupported nodes as escaped tmark in inline `<code>` or block
`<pre><code>` elements.

Markdown targets GitHub Flavored Markdown (GFM), including task lists, tables,
and strikethrough. Highlighting uses raw HTML `<mark>`; math uses code spans or
fenced code blocks because math syntax is not part of GFM. Tables without a
header get an empty header; merged cells use an HTML table. Dates use the same
timezone-aware display format as HTML. Collages and slideshows become sequential
media links. List label styles, table borders, striping, and spoilers require
renderer-specific styling and are not preserved by GFM alone.

Spoiler images and videos use `<figure class="tmark-spoiler" tabindex="0">` in
HTML and in Markdown with HTML allowed. The example stylesheet blurs the media
until hover or keyboard focus. With `htmlLess=true`, they use ordinary Markdown
images or video links without spoiler styling.

`ToMarkdown(content, htmlLess)` accepts an explicit boolean. Pass `true` to emit
Markdown without raw HTML tags or comments:

```go
markdown, err := tmark.ToMarkdown(document.Content, true)
```

In this mode, details expand into a bold summary and visible content; HTML-only
inline styles keep their text. Anchors are omitted, and anchor/reference links
keep their labels. Table spans are ignored and cell line breaks become spaces.
Adjacent lists are separated by a thematic break, and empty inline code is
omitted. Literal HTML in text or code remains escaped text or code content.
Unknown or unsupported nodes always render as serialized tmark in inline code
or code blocks. This also applies to dates with unknown timezones. Rendering only
returns an error when fallback serialization fails, for example for malformed
`Unknown.Raw`. Soft mode belongs to the parser only.

## Generate Example Pages

Run from the `tmark-go` directory:

```sh
go run ./example/html
go run ./example/markdown
go run ./example/markdown -html-less
```

The generators read `../tmark/examples` and write to `example/html/pages` or
`example/markdown/pages`, including an index. With `-html-less`, Markdown pages
go to `example/markdown/pages-htmlless`. Both accept `-input` and `-output`.

## Terminal Export

`ToTerminal(content, plainText, revealSpoilers)` renders ANSI text; the booleans disable styling and reveal spoilers.
CLI example: `go run ./example/cli ../tmark/examples/all_widgets.tmark` prints to stdout.
