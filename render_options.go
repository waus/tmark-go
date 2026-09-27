package tmark

import "fmt"

const defaultRenderPlaceholder = "[unsupported]"

type RenderOption func(*renderConfig)

type renderConfig struct {
	soft        bool
	placeholder string
}

func newRenderConfig(options []RenderOption) renderConfig {
	cfg := renderConfig{placeholder: defaultRenderPlaceholder}
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func WithSoftConversion() RenderOption {
	return func(cfg *renderConfig) {
		cfg.soft = true
	}
}

func WithPlaceholderText(text string) RenderOption {
	return func(cfg *renderConfig) {
		cfg.placeholder = text
	}
}

type RenderError struct {
	Tag    string
	Reason string
}

func (e *RenderError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("tmark: cannot render node %q", e.Tag)
	}
	return fmt.Sprintf("tmark: cannot render node %q: %s", e.Tag, e.Reason)
}

func unsupportedRenderTag(tag string, reason string) error {
	return &RenderError{Tag: tag, Reason: reason}
}
