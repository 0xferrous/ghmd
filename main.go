package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2/styles"
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

var pageTmpl = template.Must(template.New("page").Funcs(template.FuncMap{"baseThemeCSS": baseThemeCSS, "docPageCSS": docPageCSS}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
{{docPageCSS}}
  </style>
</head>
<body>
  {{if .Header}}
  <header class="doc-header">{{.Header}}</header>
  {{end}}
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
  {{.Script}}
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
	Header     template.HTML
	Script     template.HTML
	TOC        template.HTML
	Body       template.HTML
	HasMermaid bool
}

func main() {
	args, serverMode := preprocessServerArgs(os.Args[1:])
	os.Args = append([]string{os.Args[0]}, args...)

	inPath := flag.String("i", "", "input markdown file (default: stdin)")
	outPath := flag.String("o", "", "output html file (default: stdout)")
	openInBrowser := flag.Bool("open", false, "open the generated HTML in your browser")
	titleOverride := flag.String("title", "", "override HTML title")
	theme := flag.String("theme", "github", "syntax highlighting theme (see available themes in help)")
	serverRoot := flag.String("server", "", "start HTTP server; optional root dir path")
	serverHost := flag.String("host", "127.0.0.1", "server host")
	serverPort := flag.Int("port", 8080, "server port")
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

	if serverMode {
		if err := serveHTTP(*serverRoot, *serverHost, *serverPort, *theme); err != nil {
			fatal(err)
		}
		return
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

	content := renderMarkdownPageContent(tocHTML, bodyHTML)
	htmlOut, err := renderPageHTML(title, docPageCSS(), template.HTML(""), content, interactionScript(true, true), strings.Contains(bodyHTML, `class="mermaid"`))
	if err != nil {
		fatal(err)
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

	if _, err := io.WriteString(out, htmlOut); err != nil {
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

func availableThemes() []string {
	themes := styles.Names()
	sort.Strings(themes)
	return themes
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
