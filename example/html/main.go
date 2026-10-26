package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/waus/tmark-go"
)

var page = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
body { max-width: 850px; margin: 2rem auto; padding: 0 1rem; font: 17px/1.6 system-ui, sans-serif; color: #222; }
nav { display: flex; flex-wrap: wrap; gap: .5rem 1rem; padding-bottom: 1rem; border-bottom: 1px solid #ddd; }
nav a { white-space: nowrap; }
figure { margin: 1.5rem 0; }
figure img, video { max-width: 100%; height: auto; }
pre { overflow-x: auto; padding: 1rem; background: #f5f5f5; }
table { border-collapse: collapse; display: block; overflow-x: auto; }
td, th { padding: .3rem .6rem; }
.tmark-bordered td, .tmark-bordered th { border: 1px solid #ddd; }
.tmark-striped tr:nth-child(even) { background: #f5f5f5; }
li:has(> input[type="checkbox"]) { display: grid; grid-template-columns: 1em minmax(0, 1fr); gap: .25em .5em; margin: .5em 0; list-style: none; }
li > input[type="checkbox"] { align-self: start; margin: .45em 0 0; }
li:has(> input[type="checkbox"]) > :not(input) { grid-column: 2; }
li:has(> input[type="checkbox"]) > p { margin: 0; }
blockquote { border-left: 3px solid #ddd; margin-left: 0; padding-left: 1rem; }
details > :not(summary) { margin: 0 0 0 .5rem; padding: .5rem 0 .5rem 1rem; border-left: 1px solid #ddd; }
.collage { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr)); gap: .5rem; }
.collage > figure, .slideshow > figure { min-width: 0; margin: 0; }
.collage > figure > img, .collage > figure > video { display: block; width: 100%; aspect-ratio: 4 / 3; object-fit: cover; }
.collage > figcaption { grid-column: 1 / -1; }
.slideshow { display: grid; grid-auto-columns: 100%; gap: .5rem 1rem; overflow-x: auto; scroll-snap-type: x mandatory; }
.slideshow > figure { grid-row: 1; scroll-snap-align: start; }
.slideshow > figure > img, .slideshow > figure > video { display: block; width: 100%; aspect-ratio: 16 / 9; object-fit: contain; }
.slideshow > figcaption { grid-row: 2; grid-column: 1; position: sticky; left: 0; }
.spoiler { background: #222; color: #222; }
.spoiler:hover { color: white; }
.tmark-spoiler > img, .tmark-spoiler > video { filter: blur(16px); }
.tmark-spoiler:hover > img, .tmark-spoiler:hover > video, .tmark-spoiler:focus-within > img, .tmark-spoiler:focus-within > video { filter: none; }
</style>
<nav><a href="index.html">All examples</a>{{range .Pages}}<a href="{{.File}}">{{.Title}}</a>{{end}}</nav>
<main><h1>{{.Title}}</h1>{{.Body}}</main>
</html>
`))

type pageData struct {
	Title string
	Body  template.HTML
	Pages []renderedExample
}

type renderedExample struct {
	File  string
	Title string
	Body  template.HTML
}

func main() {
	input := flag.String("input", "../tmark/examples", "directory containing .tmark examples")
	output := flag.String("output", "example/html/pages", "directory for HTML pages")
	flag.Parse()
	if err := generate(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(filepath.Join(*output, "index.html"))
}

func generate(input, output string) error {
	entries, err := os.ReadDir(input)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}

	var rendered []renderedExample
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
		body, err := tmark.ToHTML(document.Content)
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Name(), err)
		}
		rendered = append(rendered, renderedExample{
			File:  strings.TrimSuffix(entry.Name(), ".tmark") + ".html",
			Title: string(document.Title),
			Body:  template.HTML(body),
		})
	}
	if len(rendered) == 0 {
		return fmt.Errorf("no .tmark examples in %s", input)
	}
	for _, ex := range rendered {
		if err := writePage(filepath.Join(output, ex.File), pageData{Title: ex.Title, Body: ex.Body, Pages: rendered}); err != nil {
			return err
		}
	}
	return writePage(filepath.Join(output, "index.html"), pageData{Title: "Tmark examples", Pages: rendered})
}

func writePage(path string, data pageData) error {
	var buf bytes.Buffer
	if err := page.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
