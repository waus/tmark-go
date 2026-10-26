package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/waus/tmark-go"
)

func main() {
	input := flag.String("input", "../tmark/examples", "directory containing .tmark examples")
	output := flag.String("output", "", "output directory (default: example/markdown/pages or pages-htmlless)")
	htmlLess := flag.Bool("html-less", false, "render Markdown without raw HTML")
	flag.Parse()
	if *output == "" {
		*output = "example/markdown/pages"
		if *htmlLess {
			*output += "-htmlless"
		}
	}
	options := []tmark.RenderOption{tmark.WithSoftConversion()}
	if *htmlLess {
		options = append(options, tmark.WithHTMLLess())
	}
	if err := generate(*input, *output, options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(filepath.Join(*output, "index.md"))
}

func generate(input, output string, options []tmark.RenderOption) error {
	entries, err := os.ReadDir(input)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	index := tmark.NewList()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmark") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(input, entry.Name()))
		if err != nil {
			return err
		}
		var document tmark.Document
		if err := tmark.DefaultUnmarshaler.Soft().Unmarshal(data, &document); err != nil {
			return fmt.Errorf("%s: %w", entry.Name(), err)
		}
		file := strings.TrimSuffix(entry.Name(), ".tmark") + ".md"
		content := append(tmark.NewRichBlocks(tmark.NewHeader(1, document.Title)), document.Content...)
		if err := writePage(filepath.Join(output, file), content, options); err != nil {
			return fmt.Errorf("%s: %w", entry.Name(), err)
		}
		index.Items = append(index.Items, tmark.NewListItem(tmark.NewParagraph(tmark.NewLink(tmark.Text(file), document.Title))))
	}
	if len(index.Items) == 0 {
		return fmt.Errorf("no .tmark examples in %s", input)
	}
	return writePage(filepath.Join(output, "index.md"), tmark.NewRichBlocks(tmark.NewHeader(1, tmark.NewText("Tmark examples")), index), options)
}

func writePage(path string, content tmark.RichBlocks, options []tmark.RenderOption) error {
	body, err := tmark.ToMarkdown(content, options...)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}
