package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	stdhtml "html"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2/styles"
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

type tocItem struct {
	Level int
	ID    string
	Text  string
}

type tocNode struct {
	Item     tocItem
	Children []*tocNode
}

var pageTmpl = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg: #ffffff;
      --fg: #24292f;
      --muted: #57606a;
      --border: #d0d7de;
      --panel: #f6f8fa;
      --link: #0969da;
      --code-bg: #f6f8fa;
      --toc-width: 280px;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #0d1117;
        --fg: #c9d1d9;
        --muted: #8b949e;
        --border: #30363d;
        --panel: #161b22;
        --link: #58a6ff;
        --code-bg: #161b22;
      }
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--fg);
      font-family: -apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji";
      line-height: 1.6;
    }
    a { color: var(--link); text-decoration: none; }
    a:hover { text-decoration: underline; }
    .layout {
      display: grid;
      grid-template-columns: minmax(220px, var(--toc-width)) minmax(0, 1fr);
      gap: 24px;
      max-width: 1440px;
      margin: 0 auto;
      padding: 24px;
      align-items: start;
    }
    .toc {
      position: sticky;
      top: 24px;
      max-height: calc(100vh - 48px);
      overflow: auto;
      padding: 16px;
      border: 1px solid var(--border);
      border-radius: 12px;
      background: var(--panel);
      font-size: 14px;
    }
    .toc h2 {
      margin: 0 0 12px;
      font-size: 14px;
      text-transform: uppercase;
      letter-spacing: .08em;
      color: var(--muted);
    }
    .toc ul {
      list-style: none;
      padding-left: 16px;
      margin: 0;
    }
    .toc > ul { padding-left: 0; }
    .toc li { margin: 4px 0; }
    .toc a { color: var(--fg); }
    .toc a:hover { color: var(--link); }
    .markdown-body {
      min-width: 0;
      max-width: 100%;
    }
    .markdown-body > :first-child { margin-top: 0; }
    .markdown-body h1,
    .markdown-body h2,
    .markdown-body h3,
    .markdown-body h4,
    .markdown-body h5,
    .markdown-body h6 {
      position: relative;
      line-height: 1.25;
      margin: 24px 0 16px;
      scroll-margin-top: 24px;
    }
    .markdown-body h1 .heading-anchor,
    .markdown-body h2 .heading-anchor,
    .markdown-body h3 .heading-anchor,
    .markdown-body h4 .heading-anchor,
    .markdown-body h5 .heading-anchor,
    .markdown-body h6 .heading-anchor {
      position: absolute;
      left: -24px;
      top: 50%;
      transform: translateY(-50%);
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 20px;
      height: 20px;
      color: var(--muted);
      text-decoration: none;
      opacity: 0;
      transition: opacity .15s ease-in-out;
    }
    .markdown-body h1:hover .heading-anchor,
    .markdown-body h2:hover .heading-anchor,
    .markdown-body h3:hover .heading-anchor,
    .markdown-body h4:hover .heading-anchor,
    .markdown-body h5:hover .heading-anchor,
    .markdown-body h6:hover .heading-anchor,
    .markdown-body .heading-anchor:focus {
      opacity: 1;
    }
    .markdown-body .heading-anchor svg {
      width: 16px;
      height: 16px;
      fill: currentColor;
      flex: none;
    }
    .markdown-body h1 { font-size: 2em; }
    .markdown-body h2 { font-size: 1.5em; }
    .markdown-body h3 { font-size: 1.25em; }
    .markdown-body h4 { font-size: 1em; }
    .markdown-body h5 { font-size: .875em; }
    .markdown-body h6 { font-size: .85em; color: var(--muted); }
    .markdown-body p, .markdown-body ul, .markdown-body ol, .markdown-body blockquote,
    .markdown-body pre, .markdown-body table, .markdown-body details {
      margin-top: 0;
      margin-bottom: 16px;
    }
    .markdown-body blockquote {
      padding: 0 16px;
      color: var(--muted);
      border-left: .25em solid var(--border);
    }
    .markdown-body code {
      font-family: ui-monospace,SFMono-Regular,SF Mono,Menlo,Consolas,Liberation Mono,monospace;
      font-size: .95em;
    }
    .markdown-body :not(pre) > code {
      padding: .2em .4em;
      background: var(--panel);
      border-radius: 6px;
    }
    .markdown-body pre {
      padding: 16px;
      overflow: auto;
      background: var(--code-bg);
      border: 1px solid var(--border);
      border-radius: 12px;
    }
    .markdown-body pre code {
      padding: 0;
      background: transparent;
      border: 0;
      display: block;
      white-space: pre;
    }
    .markdown-body table {
      display: block;
      width: max-content;
      max-width: 100%;
      overflow: auto;
      border-collapse: collapse;
    }
    .markdown-body th,
    .markdown-body td {
      padding: 6px 13px;
      border: 1px solid var(--border);
    }
    .markdown-body th { background: var(--panel); }
    .markdown-body img { max-width: 100%; }
    .markdown-body hr {
      border: 0;
      border-top: 1px solid var(--border);
      margin: 24px 0;
    }
    .markdown-body .task-list-item {
      list-style-type: none;
    }
    .markdown-body .task-list-item input {
      margin: 0 .4em .25em -1.4em;
      vertical-align: middle;
    }
    .markdown-body .footnotes {
      margin-top: 24px;
      padding-top: 16px;
      border-top: 1px solid var(--border);
      font-size: .95em;
      color: var(--muted);
    }
    .markdown-body .footnotes ol {
      padding-left: 1.5em;
    }
    .markdown-body .footnotes li + li {
      margin-top: 8px;
    }
    .markdown-body .footnote-ref,
    .markdown-body .footnote-backref {
      text-decoration: none;
    }
    .markdown-body .footnote-ref {
      vertical-align: super;
      font-size: .8em;
    }
    .markdown-body .footnote-backref {
      margin-left: .25em;
      color: var(--muted);
    }
    .markdown-body .mermaid {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 16px;
      overflow: auto;
    }
    @media (max-width: 900px) {
      .layout { grid-template-columns: 1fr; }
      .toc { position: static; max-height: none; }
    }
  </style>
</head>
<body>
  <div class="layout">
    {{if .TOC}}
    <nav class="toc" aria-label="Table of contents">
      <h2>Contents</h2>
      {{.TOC}}
    </nav>
    {{end}}
    <main class="markdown-body">
      {{.Body}}
    </main>
  </div>
  <script>
    function scrollToFragment() {
      if (!location.hash) return;
      const id = decodeURIComponent(location.hash.slice(1));
      const el = document.getElementById(id);
      if (!el) return;
      el.scrollIntoView({ block: 'start' });
      if (history.replaceState) {
        history.replaceState(null, '', location.hash);
      }
    }
    window.addEventListener('load', () => {
      requestAnimationFrame(scrollToFragment);
      setTimeout(scrollToFragment, 50);
      setTimeout(scrollToFragment, 250);
    });
    window.addEventListener('hashchange', scrollToFragment);
  </script>
  {{if .HasMermaid}}
  <script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs';
    mermaid.initialize({ startOnLoad: true, theme: matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'default' });
  </script>
  {{end}}
</body>
</html>`))

type pageData struct {
	Title      string
	TOC        template.HTML
	Body       template.HTML
	HasMermaid bool
}

func main() {
	inPath := flag.String("i", "", "input markdown file (default: stdin)")
	outPath := flag.String("o", "", "output html file (default: stdout)")
	openInBrowser := flag.Bool("open", false, "open the generated HTML in your browser")
	titleOverride := flag.String("title", "", "override HTML title")
	theme := flag.String("theme", "github", "syntax highlighting theme (see available themes in help)")
	tocMinLevel := flag.Int("toc-min-level", 2, "minimum heading level to include in the table of contents")
	tocMaxLevel := flag.Int("toc-max-level", 6, "maximum heading level to include in the table of contents")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage: %s [flags]\n\n", os.Args[0])
		fmt.Fprintln(out, "Flags:")
		flag.PrintDefaults()
		fmt.Fprintln(out, "\nAvailable -theme values:")
		for _, name := range availableThemes() {
			fmt.Fprintln(out, "  -", name)
		}
	}
	flag.Parse()

	if styles.Get(*theme) == nil {
		fatal(fmt.Errorf("unknown theme %q", *theme))
	}

	src, name, err := readInput(*inPath)
	if err != nil {
		fatal(err)
	}

	baseDir := filepath.Dir(name)
	if name == "stdin" {
		if wd, err := os.Getwd(); err == nil {
			baseDir = wd
		}
	}
	if baseDir == "" {
		baseDir = "."
	}

	bodyHTML, headings, err := renderMarkdown(src, *theme, baseDir)
	if err != nil {
		fatal(err)
	}

	items := filterHeadings(headings, *tocMinLevel, *tocMaxLevel)
	tocHTML := renderTOC(items)

	title := strings.TrimSpace(*titleOverride)
	if title == "" {
		if len(items) > 0 {
			title = items[0].Text
		} else {
			title = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
			if title == "" || title == "." || title == string(filepath.Separator) || title == "stdin" {
				title = "Document"
			}
		}
	}

	data := pageData{
		Title:      title,
		TOC:        template.HTML(tocHTML),
		Body:       template.HTML(bodyHTML),
		HasMermaid: strings.Contains(bodyHTML, `class="mermaid"`),
	}

	outputPath := *outPath
	var out io.Writer = os.Stdout
	var outFile *os.File
	if outputPath != "" {
		f, err := os.Create(outputPath)
		if err != nil {
			fatal(err)
		}
		outFile = f
		out = f
	} else if *openInBrowser {
		f, err := os.CreateTemp("", "ghmd-*.html")
		if err != nil {
			fatal(err)
		}
		outputPath = f.Name()
		outFile = f
		out = f
	}

	if err := pageTmpl.Execute(out, data); err != nil {
		fatal(err)
	}

	if outFile != nil {
		if err := outFile.Close(); err != nil {
			fatal(err)
		}
	}

	if *openInBrowser {
		if outputPath == "" {
			fatal(fmt.Errorf("cannot open output when writing to stdout; use -o or let ghmd create a temp file"))
		}
		if err := openBrowser(outputPath); err != nil {
			fatal(err)
		}
	}
}

func readInput(path string) ([]byte, string, error) {
	if path == "" {
		b, err := io.ReadAll(os.Stdin)
		return b, "stdin", err
	}
	b, err := os.ReadFile(path)
	return b, path, err
}

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

func openBrowser(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", absPath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", absPath)
	default:
		cmd = exec.Command("xdg-open", absPath)
	}
	return cmd.Run()
}

func availableThemes() []string {
	themes := styles.Names()
	sort.Strings(themes)
	return themes
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
