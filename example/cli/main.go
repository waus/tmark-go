package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/waus/tmark-go"
)

func main() {
	plain := flag.Bool("plain", false, "disable ANSI styles")
	reveal := flag.Bool("reveal-spoilers", false, "show spoiler text and media")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: cli [-plain] [-reveal-spoilers] path/to/example.tmark")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *plain, *reveal); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string, plain, reveal bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var document tmark.Document
	if err := tmark.DefaultUnmarshaler.Soft().Unmarshal(data, &document); err != nil {
		return err
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return err
	}
	plain = plain || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || info.Mode()&os.ModeCharDevice == 0
	content := document.Content
	if document.Title != "" {
		content = append(tmark.NewRichBlocks(tmark.NewHeader(1, document.Title)), content...)
	}
	output, err := tmark.ToTerminal(content, plain, reveal)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(output, '\n'))
	return err
}
