package shortcode

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	html5 "golang.org/x/net/html"
)

func RegisterHandler(name string, handler ShortcodeHandler) {
	shortcodeHandlers[name] = handler
}

var shortcodeHandlers = map[string]ShortcodeHandler{}

type ShortcodeHandler interface {
	Handle(name string, attributes map[string]any, body string) (string, error)
}

func HandleFunc(fn func(string, map[string]any, string) (string, error)) ShortcodeHandler {
	return handlerFunc(fn)
}

type handlerFunc func(string, map[string]any, string) (string, error)

func (h handlerFunc) Handle(name string, attributes map[string]any, body string) (string, error) {
	return h(name, attributes, body)
}

func newShortcodeAST(name string, attributes []*ast.Attribute) *shortcodeAST {
	sa := &shortcodeAST{
		Name: name,
	}
	for _, a := range attributes {
		sa.SetAttributeString(string(a.Name), a.Value)
	}
	return sa
}

type shortcodeAST struct {
	ast.BaseInline
	Name string
}

func (s *shortcodeAST) Dump(source []byte, level int) {
	ast.DumpHelper(s, source, level, nil, nil)
}

func (s *shortcodeAST) Kind() ast.NodeKind {
	return KindShortcode
}

var (
	KindShortcode          = ast.NewNodeKind("Shortcode")
	shortcodeRegex         = regexp.MustCompile(`({{<)(.+)(>}})`)
	defaultShortcodeParser = &shortcodeParser{}
)

func newShortcodeParser() parser.InlineParser {
	return defaultShortcodeParser
}

type shortcodeParser struct{}

func (p *shortcodeParser) Trigger() []byte {
	return []byte{'{', '{', '<'}
}
func (p *shortcodeParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	m := shortcodeRegex.FindSubmatchIndex(line)
	if m == nil {
		return nil
	}
	value := bytes.TrimSpace(line[m[3]:m[5]])
	function, attributes, hasAttributes := bytes.Cut(value, []byte{' '})
	attrs := []*ast.Attribute{}
	if hasAttributes {
		nodes, _ := html5.ParseFragment(
			strings.NewReader("<div "+string(attributes)+">"),
			nil,
		)
		for _, n := range nodes[0].LastChild.FirstChild.Attr {
			attrs = append(attrs, &ast.Attribute{Name: []byte(n.Key), Value: n.Val})
		}
	}

	block.Advance(m[1])
	return newShortcodeAST(string(function), attrs)

}
func (p *shortcodeParser) CloseBlock(parent ast.Node, pc parser.Context) {
	// nothing?
}

type shortcodeHTMLRenderer struct {
	html.Config
}

func newShortcodeHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &shortcodeHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, o := range opts {
		o.SetHTMLOption(&r.Config)
	}
	return r
}
func (r *shortcodeHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindShortcode, r.renderShortcode)
}

func (r *shortcodeHTMLRenderer) renderShortcode(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*shortcodeAST)
	fn := shortcodeHandlers[n.Name]
	if fn == nil {
		w.WriteString("<!-- unknown shortcode: " + n.Name + " -->")
		return ast.WalkContinue, nil
	}

	attrs := make(map[string]any)
	if node.Attributes() != nil {
		for _, a := range node.Attributes() {
			attrs[string(a.Name)] = a.Value
		}
	}
	resp, err := fn.Handle(n.Name, attrs, string(source))
	if err != nil {
		return ast.WalkStop, err
	}
	w.WriteString(resp)
	return ast.WalkContinue, nil
}

var Extension = &shortcode{}

type shortcode struct{}

func (s *shortcode) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(newShortcodeParser(), 0),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(newShortcodeHTMLRenderer(), 500),
	))
}
