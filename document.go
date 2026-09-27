package tmark

// Document wraps markup content with generic document metadata.
type Document struct {
	URL           Text            `tmark:"named"`
	Title         Text            `tmark:"named"`
	Description   *Text           `tmark:"named"`
	AuthorName    *Text           `tmark:"named,author_name"`
	AuthorURL     *Text           `tmark:"named,author_url"`
	ImageURL      *Text           `tmark:"named,image_url"`
	AttachedMedia []AttachedMedia `tmark:"named,attached_media"`
	Content       RichBlocks      `tmark:"unnamed"`
}

type AttachedMedia struct {
	Hash    Text       `tmark:"named"`
	Content RichBlocks `tmark:"unnamed"`
}
