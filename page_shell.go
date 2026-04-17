package main

import (
	"bytes"
	"html/template"
)

var pageShellTmpl = template.Must(template.New("shell").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
{{.CSS}}
  </style>
</head>
<body>
{{if .Header}}{{.Header}}{{end}}
{{.Content}}
{{.Script}}
{{if .HasMermaid}}
  <script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs';
    mermaid.initialize({ startOnLoad: true, theme: matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'default' });
  </script>
{{end}}
</body>
</html>`))

type pageShellData struct {
	Title      string
	CSS        template.CSS
	Header     template.HTML
	Content    template.HTML
	Script     template.HTML
	HasMermaid bool
}

func renderPageHTML(title string, css template.CSS, header, content, script template.HTML, hasMermaid bool) (string, error) {
	var buf bytes.Buffer
	data := pageShellData{
		Title:      title,
		CSS:        css,
		Header:     header,
		Content:    content,
		Script:     script,
		HasMermaid: hasMermaid,
	}
	if err := pageShellTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderMarkdownPageContent(tocHTML, bodyHTML string) template.HTML {
	return template.HTML(`<div class="layout">` +
		`<nav class="toc" aria-label="Table of contents"><h2>Contents</h2>` + tocHTML + `</nav>` +
		`<main class="markdown-body">` + bodyHTML + `</main>` +
		`</div>`)
}
