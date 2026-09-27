package tmark

func T(text string) Text            { return NewText(text) }
func B(children ...richText) Bold   { return NewBold(children...) }
func I(children ...richText) Italic { return NewItalic(children...) }

func P(children ...richText) Paragraph { return NewParagraph(children...) }
