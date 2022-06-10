package epub

import (
	"bytes"
	"strings"
	"sync"

	"github.com/cahaba-ts/epub/shortcode"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	lock    = &sync.Mutex{}
	current *Book
)

type ShortcodeHandler func(*Book, string, map[string]any, string) (string, error)

func RegisterShortcode(name string, handler ShortcodeHandler) {
	shortcode.RegisterHandler(name, shortcode.HandleFunc(
		func(name string, attrs map[string]any, body string) (string, error) {
			return handler(current, name, attrs, body)
		},
	))
}

// AddMDExtension adds another extension to goldmark. Note that
// the Table, Strikethrough, and Definition List extensions are
// already added.
func (e *Book) AddMDExtension(ext goldmark.Extender) {
	e.exts = append(e.exts, ext)
}

func (e *Book) renderMarkdown(content string) ([]string, error) {
	if e.md == nil {
		e.md = goldmark.New(
			goldmark.WithExtensions(e.exts...),
			goldmark.WithExtensions(shortcode.Extension),
			goldmark.WithRendererOptions(
				html.WithHardWraps(),
				html.WithXHTML(),
			),
		)
	}
	lock.Lock()
	current = e
	buf := &bytes.Buffer{}
	err := e.md.Convert([]byte(content), buf)
	lock.Unlock()
	if err != nil {
		return nil, err
	}

	// must break into multiple sections
	if !strings.Contains(buf.String(), "<!-- PAGE BREAK -->") {
		return []string{content}, nil
	}

	parts := strings.Split(content, "<!-- PAGE BREAK -->")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		parts[i] = strings.TrimSuffix(parts[i], "<p>")
		parts[i] = strings.TrimPrefix(parts[i], "</p>")
	}
	return parts, nil
}
