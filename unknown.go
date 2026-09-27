package tmark

// Unknown preserves a valid node with a tag unknown to this package.
type Unknown struct {
	block
	rich

	Raw []byte
}
