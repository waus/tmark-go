package tmark

const maxDepth = 16

type blockContent interface {
	IsBlockContent()
}

type richText interface {
	IsRichText()
}

type slideshowContent interface {
	IsSlideshowContent()
}

type block struct{}

func (block) IsBlockContent() {}

type rich struct{}

func (rich) IsRichText() {}

type slideshow struct{}

func (slideshow) IsSlideshowContent() {}
