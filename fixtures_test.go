package tmark

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixtureValidRoundTrip(t *testing.T) {
	for _, path := range fixtureFiles(t, "../fixtures/valid", ".tmark") {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data := fixtureData(t, path)

			var got Document
			if err := DefaultUnmarshaler.Soft().Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			out, err := Marshal(got)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if !bytes.Equal(out, data) {
				t.Fatalf("round trip mismatch\n got: %q\nwant: %q", out, data)
			}
		})
	}
}

func TestFixtureValidNodeRoundTrip(t *testing.T) {
	for _, path := range fixtureFiles(t, "../fixtures/valid", ".fixture") {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data := fixtureData(t, path)

			var got any
			if err := Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			out, err := Marshal(got)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if !bytes.Equal(out, data) {
				t.Fatalf("round trip mismatch\n got: %q\nwant: %q", out, data)
			}
		})
	}
}

func TestFixtureInvalidRejected(t *testing.T) {
	for _, path := range fixtureFiles(t, "../fixtures/invalid", ".tmark") {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data := fixtureData(t, path)

			var got Document
			if err := Unmarshal(data, &got); err == nil {
				t.Fatal("Unmarshal() error = nil, want error")
			}
		})
	}
}

func fixtureFiles(t *testing.T, dir, suffix string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	var files []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), suffix) {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	if len(files) == 0 {
		t.Fatalf("no %s fixtures in %s", suffix, dir)
	}
	return files
}

func fixtureData(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
