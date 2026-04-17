package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	stdhtml "html"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

func renderMarkdown(src []byte, theme, baseDir string) (string, []tocItem, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			highlighting.NewHighlighting(
				highlighting.WithStyle(theme),
				highlighting.WithWrapperRenderer(mermaidWrapperRenderer),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			renderer.WithNodeRenderers(
				util.Prioritized(&githubLikeRenderer{baseDir: baseDir}, 100),
			),
		),
	)

	doc := md.Parser().Parse(text.NewReader(src))
	headings := collectHeadingsFromDoc(doc, src)

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, src, doc); err != nil {
		return "", nil, err
	}

	return buf.String(), headings, nil
}

type githubLikeRenderer struct {
	baseDir string
}

func (r *githubLikeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, renderHeadingWithAnchor)
	reg.Register(ast.KindImage, r.renderImage)
}

func renderHeadingWithAnchor(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	h := node.(*ast.Heading)
	if entering {
		_, _ = w.WriteString("<h")
		_ = w.WriteByte("0123456"[h.Level])
		if h.Attributes() != nil {
			html.RenderAttributes(w, h, html.HeadingAttributeFilter)
		}
		_ = w.WriteByte('>')
		if idVal, ok := h.AttributeString("id"); ok {
			var id string
			switch v := idVal.(type) {
			case string:
				id = v
			case []byte:
				id = string(v)
			default:
				id = fmt.Sprint(v)
			}
			_, _ = w.WriteString(`<a class="heading-anchor" href="#`)
			_, _ = w.WriteString(stdhtml.EscapeString(id))
			_, _ = w.WriteString(`" aria-label="Permalink to this heading" title="Permalink to this heading"><svg viewBox="0 0 16 16" aria-hidden="true" focusable="false"><path d="M7.775 3.275a.75.75 0 0 0-1.06-1.06L4.22 4.69a3.25 3.25 0 1 0 4.596 4.596l1.414-1.414a.75.75 0 0 0-1.06-1.06L7.756 8.227a1.75 1.75 0 1 1-2.475-2.475l2.494-2.477Zm.45 9.45a.75.75 0 0 0 1.06 1.06l2.495-2.495a3.25 3.25 0 0 0-4.596-4.596L5.77 8.108a.75.75 0 1 0 1.06 1.06l1.414-1.414a1.75 1.75 0 1 1 2.475 2.475l-2.494 2.496Z"/></svg></a>`)
		}
		return ast.WalkContinue, nil
	}

	_, _ = w.WriteString("</h")
	_ = w.WriteByte("0123456"[h.Level])
	_, _ = w.WriteString(">\n")
	return ast.WalkContinue, nil
}

func (r *githubLikeRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Image)
	dest := util.URLEscape(n.Destination, true)
	if inline, ok := r.inlineImageSource(string(dest)); ok {
		dest = []byte(inline)
	}

	_, _ = w.WriteString(`<img src="`)
	_, _ = w.Write(util.EscapeHTML(dest))
	_, _ = w.WriteString(`" alt="`)
	_, _ = w.WriteString(stdhtml.EscapeString(string(n.Text(source))))
	_ = w.WriteByte('"')
	if n.Title != nil {
		_, _ = w.WriteString(` title="`)
		_, _ = w.Write(n.Title)
		_ = w.WriteByte('"')
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, n, html.ImageAttributeFilter)
	}
	_, _ = w.WriteString(">")
	return ast.WalkSkipChildren, nil
}

func (r *githubLikeRenderer) inlineImageSource(dest string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(dest))
	if lower == "" {
		return "", false
	}
	if strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "//") || strings.Contains(lower, "://") {
		return "", false
	}

	path := dest
	if unescaped, err := url.PathUnescape(dest); err == nil {
		path = unescaped
	}
	if idx := strings.IndexAny(path, "?#"); idx >= 0 {
		path = path[:idx]
	}
	if path == "" {
		return "", false
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(r.baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return toDataURI(path, data), true
}

func toDataURI(path string, data []byte) string {
	mimeType := strings.TrimSpace(mime.TypeByExtension(strings.ToLower(filepath.Ext(path))))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	if strings.HasSuffix(strings.ToLower(path), ".svg") {
		mimeType = "image/svg+xml"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func mermaidWrapperRenderer(w util.BufWriter, ctx highlighting.CodeBlockContext, entering bool) {
	if ctx.Highlighted() {
		return
	}

	if lang, ok := ctx.Language(); ok && strings.EqualFold(string(lang), "mermaid") {
		if entering {
			_, _ = w.WriteString(`<pre class="mermaid">`)
		} else {
			_, _ = w.WriteString("</pre>\n")
		}
		return
	}

	if entering {
		_, _ = w.WriteString("<pre><code")
		if lang, ok := ctx.Language(); ok && len(lang) > 0 {
			_, _ = w.WriteString(` class="language-`)
			_, _ = w.Write(util.EscapeHTML(lang))
			_, _ = w.WriteString(`"`)
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</code></pre>\n")
	}
}

func collectHeadingsFromDoc(doc ast.Node, source []byte) []tocItem {
	var items []tocItem
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		idVal, ok := h.AttributeString("id")
		if !ok {
			return ast.WalkContinue, nil
		}
		var id string
		switch v := idVal.(type) {
		case string:
			id = v
		case []byte:
			id = string(v)
		default:
			id = fmt.Sprint(v)
		}
		text := strings.TrimSpace(string(h.Text(source)))
		if text == "" {
			return ast.WalkContinue, nil
		}
		items = append(items, tocItem{Level: h.Level, ID: id, Text: text})
		return ast.WalkContinue, nil
	})
	return items
}

func filterHeadings(items []tocItem, minLevel, maxLevel int) []tocItem {
	if minLevel < 1 {
		minLevel = 1
	}
	if maxLevel > 6 {
		maxLevel = 6
	}
	if minLevel > maxLevel {
		minLevel, maxLevel = maxLevel, minLevel
	}
	filtered := make([]tocItem, 0, len(items))
	for _, item := range items {
		if item.Level >= minLevel && item.Level <= maxLevel {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func renderTOC(items []tocItem) string {
	if len(items) == 0 {
		return ""
	}

	roots := buildTOCTree(items)
	var b strings.Builder
	b.WriteString("<ul>")
	for _, n := range roots {
		renderTOCNode(&b, n)
	}
	b.WriteString("</ul>")
	return b.String()
}

func buildTOCTree(items []tocItem) []*tocNode {
	var roots []*tocNode
	var stack []*tocNode

	for _, item := range items {
		node := &tocNode{Item: item}
		for len(stack) > 0 && stack[len(stack)-1].Item.Level >= item.Level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, node)
		}
		stack = append(stack, node)
	}

	return roots
}

func renderTOCNode(b *strings.Builder, n *tocNode) {
	b.WriteString("<li><a href=\"#")
	b.WriteString(stdhtml.EscapeString(n.Item.ID))
	b.WriteString("\">")
	b.WriteString(stdhtml.EscapeString(n.Item.Text))
	b.WriteString("</a>")
	if len(n.Children) > 0 {
		b.WriteString("<ul>")
		for _, child := range n.Children {
			renderTOCNode(b, child)
		}
		b.WriteString("</ul>")
	}
	b.WriteString("</li>")
}
