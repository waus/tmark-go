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

`tmark` document content can be rendered directly to HTML or Markdown.

```go
html, err := tmark.ToHTML(document.Content)
markdown, err := tmark.ToMarkdown(document.Content)
```

Soft conversion replaces unsupported nodes with placeholders or close markdown
approximations:

```go
html, err := tmark.ToHTML(document.Content, tmark.WithSoftConversion())
markdown, err := tmark.ToMarkdown(document.Content, tmark.WithSoftConversion())
```

`WithPlaceholderText` changes the default `[unsupported]` placeholder text.
