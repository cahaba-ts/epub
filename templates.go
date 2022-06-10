package epub

var overrides = make(map[string][]byte)

// RetrieveTemplate returns either the default template
// or an overridden template for the filename
func RetrieveTemplate(filename string) ([]byte, error) {
	if body, ok := overrides[filename]; ok {
		return body, nil
	}
	return tmpl.ReadFile(filename)
}

// OverrideTemplate will set a new template for the filename.
// Valid filenames are book.opf, chapter.xhtml, container.xml,
// cover.xhtml, main.css, toc.ncx, and toc.xhtml.
func OverrideTemplate(filename string, content []byte) {
	overrides[filename] = content
}
