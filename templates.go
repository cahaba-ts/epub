package epub

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

//go:embed tmpl
var tmpl embed.FS

var overrides = make(map[string][]byte)

// RetrieveTemplate returns either the default template
// or an overridden template for the filename
func RetrieveTemplate(filename string) ([]byte, error) {
	if body, ok := overrides[filename]; ok {
		return body, nil
	}
	b, err := tmpl.ReadFile("tmpl/" + filename)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Can't find file in embed",
		)
	}
	return b, nil
}

// OverrideTemplate will set a new template for the filename.
// Valid filenames are book.opf, chapter.xhtml, container.xml,
// cover.xhtml, main.css, toc.ncx, and toc.xhtml.
func OverrideTemplate(filename string, content []byte) {
	overrides[filename] = content
}

func CompileTemplate(filename string) (*template.Template, error) {
	b, err := RetrieveTemplate(filename)
	if err != nil {
		return nil, err
	}
	t, err := template.New("cahaba").
		Funcs(template.FuncMap{
			"clean": func(s, cutset string) string {
				return strings.TrimPrefix(s, cutset)
			},
		}).Parse(string(b))
	if err != nil {
		return nil, errors.Wrap(
			err,
			fmt.Sprintf("Compile Template (%s): ", filename),
		)
	}
	return t, nil
}
