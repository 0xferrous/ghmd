package main

import (
	"fmt"
	"github.com/alecthomas/chroma/v2/styles"
	stdhtml "html"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type sortState struct {
	Key  string
	Desc bool
}

func preprocessServerArgs(args []string) ([]string, bool) {
	out := make([]string, 0, len(args))
	serverMode := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-server" || arg == "--server":
			serverMode = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				out = append(out, "-server="+args[i+1])
				i++
			} else {
				out = append(out, "-server=")
			}
		case strings.HasPrefix(arg, "-server="):
			serverMode = true
			out = append(out, arg)
		case strings.HasPrefix(arg, "--server="):
			serverMode = true
			out = append(out, "-server="+strings.TrimPrefix(arg, "--server="))
		default:
			out = append(out, arg)
		}
	}
	return out, serverMode
}

func resolveCodeTheme(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	if styles.Get(raw) == nil {
		return fallback
	}
	return raw
}

func serveHTTP(root, host string, port int, theme string) error {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absRoot)
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveHTTPPath(w, r, absRoot, theme)
	})

	fmt.Fprintf(os.Stderr, "serving %s at http://%s/\n", absRoot, addr)
	return http.ListenAndServe(addr, mux)
}

func serveHTTPPath(w http.ResponseWriter, r *http.Request, root, theme string) {
	codeTheme := resolveCodeTheme(r.URL.Query().Get("code-theme"), theme)

	fullPath, relPath, err := resolveHTTPPath(root, r.URL.Path)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if info.IsDir() {
		if !strings.HasSuffix(r.URL.Path, "/") {
			u := *r.URL
			u.Path = r.URL.Path + "/"
			http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
			return
		}
		handleDirectoryIndex(w, r, fullPath, relPath, codeTheme)
		return
	}

	if strings.EqualFold(filepath.Ext(fullPath), ".md") {
		handleMarkdownFile(w, fullPath, relPath, codeTheme)
		return
	}

	http.ServeFile(w, r, fullPath)
}

func resolveHTTPPath(root, reqPath string) (string, string, error) {
	clean := path.Clean("/" + reqPath)
	if clean == "/" {
		return root, "", nil
	}

	rel := strings.TrimPrefix(clean, "/")
	full := filepath.Join(root, filepath.FromSlash(rel))
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", "", err
	}

	relToRoot, err := filepath.Rel(root, fullAbs)
	if err != nil {
		return "", "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("path outside root")
	}

	return fullAbs, rel, nil
}

func handleDirectoryIndex(w http.ResponseWriter, r *http.Request, fullPath, relPath, codeTheme string) {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sortState := parseSortState(r.URL.Query().Get("sort"))
	sortEntries(entries, sortState)

	items := make([]dirEntry, 0, len(entries)+1)
	dirsCount := 0
	mdCount := 0
	filesCount := 0
	sortQuery := sortState.QueryValue()

	if relPath != "" {
		parentRel := parentRelPath(relPath)
		items = append(items, dirEntry{
			Name:     "..",
			Href:     buildPathHref(parentRel, "", true, sortQuery, codeTheme),
			Link:     true,
			Kind:     "up",
			Modified: "",
		})
	}

	for _, entry := range entries {
		name := entry.Name()
		item := dirEntry{Name: name}
		if info, err := entry.Info(); err == nil {
			item.Modified = info.ModTime().Format(time.DateTime)
		}
		if entry.IsDir() {
			item.Href = buildPathHref(relPath, name, true, sortQuery, codeTheme)
			item.Link = true
			item.Kind = "dir"
			dirsCount++
		} else if strings.EqualFold(filepath.Ext(name), ".md") {
			item.Href = buildPathHref(relPath, name, false, sortQuery, codeTheme)
			item.Link = true
			item.Kind = "md"
			mdCount++
		} else {
			item.Kind = "file"
			filesCount++
		}
		items = append(items, item)
	}

	title := "Index of /"
	if relPath != "" {
		title = "Index of /" + relPath + "/"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, renderDirectoryIndex(title, relPath, sortState, codeTheme, dirsCount, mdCount, filesCount, items))
}

func sortEntries(entries []os.DirEntry, state sortState) {
	sort.SliceStable(entries, func(i, j int) bool {
		a := entries[i]
		b := entries[j]
		switch state.Key {
		case "modified":
			ai := entryModifiedString(a)
			bj := entryModifiedString(b)
			if ai == bj {
				return compareName(a.Name(), b.Name(), state.Desc)
			}
			if state.Desc {
				return ai > bj
			}
			return ai < bj
		default:
			return compareName(a.Name(), b.Name(), state.Desc)
		}
	})
}

func entryModifiedString(entry os.DirEntry) string {
	info, err := entry.Info()
	if err != nil {
		return ""
	}
	return info.ModTime().Format(time.DateTime)
}

func compareName(a, b string, desc bool) bool {
	aLower := strings.ToLower(a)
	bLower := strings.ToLower(b)
	if aLower == bLower {
		if desc {
			return a > b
		}
		return a < b
	}
	if desc {
		return aLower > bLower
	}
	return aLower < bLower
}

func parseSortState(raw string) sortState {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return sortState{Key: "name"}
	}
	desc := strings.HasPrefix(raw, "-")
	if desc {
		raw = strings.TrimPrefix(raw, "-")
	}
	switch raw {
	case "modified":
		return sortState{Key: "modified", Desc: desc}
	default:
		return sortState{Key: "name", Desc: desc}
	}
}

func (s sortState) QueryValue() string {
	key := s.Key
	if key == "" {
		key = "name"
	}
	if s.Desc {
		return "-" + key
	}
	return key
}

func (s sortState) Toggle(key string) sortState {
	if s.Key == key {
		return sortState{Key: key, Desc: !s.Desc}
	}
	return sortState{Key: key, Desc: false}
}

func sortIndicator(state sortState, key string) string {
	if state.Key != key {
		return ""
	}
	if state.Desc {
		return "↓"
	}
	return "↑"
}

func dirEntryKind(entry os.DirEntry) int {
	if entry.IsDir() {
		return 0
	}
	if strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
		return 1
	}
	return 2
}

type dirEntry struct {
	Name     string
	Href     string
	Link     bool
	Kind     string
	Modified string
}

func renderDirectoryIndex(title, relPath string, sortState sortState, codeTheme string, dirsCount, mdCount, filesCount int, items []dirEntry) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>")
	b.WriteString(stdhtml.EscapeString(title))
	b.WriteString(`</title><style>
`)
	b.WriteString(baseThemeCSSString())
	b.WriteString(`
	.titlebar {
	  position: sticky;
	  top: 0;
	  z-index: 20;
	  display: flex;
	  font-family: monospace;
	  align-items: center;
	  justify-content: space-between;
	  gap: 12px;
	  flex-wrap: wrap;
	  margin: 0 0 .25rem;
	  padding: .75rem 0 .5rem;
	  background: color-mix(in srgb, var(--bg) 92%, transparent);
	  backdrop-filter: blur(8px);
	  border-bottom: 1px solid var(--border);
	}
	.controls {
	  display: flex;
	  align-items: center;
	  gap: 8px;
	  flex-wrap: wrap;
	}
	.code-theme {
	  display: flex;
	  align-items: center;
	  gap: 6px;
	  color: var(--muted);
	  font-size: 12px;
	}
	.code-theme span {
	  text-transform: uppercase;
	  letter-spacing: .04em;
	}
	.code-theme-select {
	  appearance: none;
	  border: 1px solid var(--border);
	  background: var(--panel);
	  color: var(--fg);
	  border-radius: 999px;
	  padding: 6px 10px;
	  font: inherit;
	  font-size: 12px;
	  line-height: 1;
	  max-width: 18rem;
	  cursor: pointer;
	}
	.code-theme-select:hover { background: var(--border); }
	table, thead th, tbody td, .breadcrumb, .theme-toggle, .code-theme, .code-theme-select {
	  font-family: monospace;
	}
	.meta { margin: 0 0 .5rem; color: var(--muted); }
	.meta small { font-size: inherit; }
	.breadcrumb a { color: var(--link); }
	.breadcrumb a:hover { color: var(--link-hover); }
	.breadcrumb .sep { color: var(--muted); }
	.theme-toggle {
	  appearance: none;
	  border: 1px solid var(--border);
	  background: var(--panel);
	  color: var(--fg);
	  border-radius: 999px;
	  padding: 6px 10px;
	  font: inherit;
	  font-size: 12px;
	  line-height: 1;
	  cursor: pointer;
	  transition: background .15s ease, border-color .15s ease, color .15s ease;
	}
	.theme-toggle:hover { background: var(--border); }
	.table-wrap { overflow-x: auto; }
	table {
	  width: 100%;
	  table-layout: fixed;
	  border-spacing: 0;
	  border-collapse: collapse;
	  border: 1px solid var(--border);
	  border-radius: 8px;
	  overflow: hidden;
	  background: var(--panel);
	}
	thead th {
	  text-align: left;
	  padding: .4rem .6rem;
	  color: var(--muted);
	  font-weight: 700;
	  font-size: .72rem;
	  text-transform: uppercase;
	  letter-spacing: .04em;
	  border-bottom: 1px solid var(--border);
	}
	thead th.sortable a {
	  display: inline-flex;
	  align-items: center;
	  gap: .35rem;
	  color: inherit;
	}
	thead th.modified,
	tbody td.modified {
	  text-align: right;
	}
	thead th.modified a {
	  justify-content: flex-end;
	  width: 100%;
	}
	thead th.active {
	  color: var(--fg);
	}
	thead .sort-arrow {
	  font-size: .85em;
	  line-height: 1;
	  color: var(--link);
	}
	tbody td {
	  padding: .35rem .6rem;
	  vertical-align: top;
	  white-space: nowrap;
	  border-bottom: 1px solid rgba(127, 127, 127, .16);
	}
	tbody tr:last-child td { border-bottom: 0; }
	tbody tr:hover td { background-color: rgba(127, 127, 127, .12); }
	tbody td.name { width: 100%; }
	tbody td.modified { width: 12rem; color: var(--muted); }
	a { color: var(--link); text-decoration: none; }
	a:hover { text-decoration: underline; }
	.dim { color: var(--muted); }
	footer { padding-top: .5rem; color: var(--muted); }
	@media (max-width: 40rem) {
	  body { padding: .75rem; }
	  tbody td.modified { width: 9rem; }
	}
	</style></head><body><main><div class="titlebar"><h1>Index of <span class="breadcrumb">`)
	b.WriteString(renderBreadcrumb(relPath, sortState.QueryValue(), codeTheme))
	b.WriteString(`</span></h1>`)
	b.WriteString(renderServerControls(codeTheme))
	b.WriteString(`</div><p class="meta"><small>directories: `)
	b.WriteString(strconv.Itoa(dirsCount))
	b.WriteString(`, markdown: `)
	b.WriteString(strconv.Itoa(mdCount))
	b.WriteString(`, files: `)
	b.WriteString(strconv.Itoa(filesCount))
	b.WriteString(`</small></p><div class="table-wrap"><table><thead><tr>`)
	b.WriteString(`<th class="sortable `)
	if sortState.Key == "name" {
		b.WriteString(`active`)
	}
	b.WriteString(`"><a href="`)
	b.WriteString(stdhtml.EscapeString(buildSortHref(relPath, sortState.Toggle("name").QueryValue(), codeTheme)))
	b.WriteString(`">Name`)
	if arrow := sortIndicator(sortState, "name"); arrow != "" {
		b.WriteString(` <span class="sort-arrow">`)
		b.WriteString(arrow)
		b.WriteString(`</span>`)
	}
	b.WriteString(`</a></th>`)

	b.WriteString(`<th class="sortable modified `)
	if sortState.Key == "modified" {
		b.WriteString(`active`)
	}
	b.WriteString(`"><a href="`)
	b.WriteString(stdhtml.EscapeString(buildSortHref(relPath, sortState.Toggle("modified").QueryValue(), codeTheme)))
	b.WriteString(`">Modified`)
	if arrow := sortIndicator(sortState, "modified"); arrow != "" {
		b.WriteString(` <span class="sort-arrow">`)
		b.WriteString(arrow)
		b.WriteString(`</span>`)
	}
	b.WriteString(`</a></th>`)
	b.WriteString(`</tr></thead><tbody>`)
	for _, item := range items {
		b.WriteString(`<tr><td class="name">`)
		if item.Link {
			b.WriteString(`<a href="`)
			b.WriteString(stdhtml.EscapeString(item.Href))
			b.WriteString(`">`)
			b.WriteString(stdhtml.EscapeString(item.Name))
			if item.Kind == "dir" {
				b.WriteString("/")
			}
			b.WriteString(`</a>`)
		} else {
			b.WriteString(`<span class="dim">`)
			b.WriteString(stdhtml.EscapeString(item.Name))
			b.WriteString(`</span>`)
		}
		b.WriteString(`</td><td class="modified">`)
		b.WriteString(stdhtml.EscapeString(item.Modified))
		b.WriteString(`</td></tr>`)
	}
	b.WriteString(`</tbody></table></div><footer><small><a href="https://github.com/0xferrous/ghmd">ghmd</a> directory listing</small></footer></main>
  ` + string(interactionScript(true, false)) + `</body></html>`)
	return b.String()
}

func buildSortHref(relPath, sortQuery, codeTheme string) string {
	return buildPathHref(relPath, "", true, sortQuery, codeTheme)
}

func buildPathHref(relPath, name string, isDir bool, sortQuery, codeTheme string) string {
	segments := make([]string, 0, 8)
	for _, part := range strings.Split(relPath, "/") {
		if part == "" {
			continue
		}
		segments = append(segments, url.PathEscape(part))
	}
	if name != "" {
		segments = append(segments, url.PathEscape(name))
	}

	href := "/"
	if len(segments) > 0 {
		href += strings.Join(segments, "/")
		if isDir {
			href += "/"
		}
	}
	if len(segments) == 0 && isDir {
		href = "/"
	}
	params := url.Values{}
	if sortQuery != "" {
		params.Set("sort", sortQuery)
	}
	if codeTheme != "" {
		params.Set("code-theme", codeTheme)
	}
	if len(params) > 0 {
		href += "?" + params.Encode()
	}
	return href
}

func renderBreadcrumb(relPath, sortQuery, codeTheme string) string {
	if relPath == "" {
		return `<a href="` + stdhtml.EscapeString(buildPathHref("", "", true, sortQuery, codeTheme)) + `">/</a>`
	}

	parts := strings.Split(relPath, "/")
	var b strings.Builder
	b.WriteString(`<a href="`)
	b.WriteString(stdhtml.EscapeString(buildPathHref("", "", true, sortQuery, codeTheme)))
	b.WriteString(`">#</a>`)
	prefix := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if prefix == "" {
			prefix = part
		} else {
			prefix = prefix + "/" + part
		}
		b.WriteString(`<span class="sep">/</span><a href="`)
		b.WriteString(stdhtml.EscapeString(buildPathHref(prefix, "", true, sortQuery, codeTheme)))
		b.WriteString(`">`)
		b.WriteString(stdhtml.EscapeString(part))
		b.WriteString(`</a>`)
	}
	return b.String()
}

func renderFileHeader(relPath, codeTheme string) string {
	if relPath == "" {
		return ""
	}
	parent := path.Dir(relPath)
	if parent == "." {
		parent = ""
	}
	file := path.Base(relPath)
	var b strings.Builder
	b.WriteString(`<div class="titlebar"><h1><span class="breadcrumb">`)
	if parent == "" {
		b.WriteString(`<a href="/">#</a>`)
	} else {
		b.WriteString(renderBreadcrumb(parent, "", codeTheme))
	}
	if parent != "" {
		b.WriteString(`<span class="sep">/</span>`)
	}
	b.WriteString(`<span class="leaf">`)
	b.WriteString(stdhtml.EscapeString(file))
	b.WriteString(`</span></span></h1>`)
	b.WriteString(renderServerControls(codeTheme))
	b.WriteString(`</div>`)
	return b.String()
}

func renderServerControls(codeTheme string) string {
	var b strings.Builder
	b.WriteString(`<div class="controls">`)
	b.WriteString(`<button class="theme-toggle" id="theme-toggle" type="button">Theme</button>`)
	b.WriteString(`<label class="code-theme"><span>Code</span><select class="code-theme-select" id="code-theme-select" aria-label="Code theme">`)
	for _, name := range availableThemes() {
		b.WriteString(`<option value="`)
		b.WriteString(stdhtml.EscapeString(name))
		b.WriteString(`"`)
		if name == codeTheme {
			b.WriteString(` selected`)
		}
		b.WriteString(`>`)
		b.WriteString(stdhtml.EscapeString(name))
		b.WriteString(`</option>`)
	}
	b.WriteString(`</select></label></div>`)
	return b.String()
}

func parentRelPath(relPath string) string {
	if relPath == "" {
		return ""
	}
	parent := path.Dir(relPath)
	if parent == "." {
		return ""
	}
	return parent
}

func handleMarkdownFile(w http.ResponseWriter, fullPath, relPath, codeTheme string) {
	src, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseDir := filepath.Dir(fullPath)
	bodyHTML, headings, err := renderMarkdown(src, codeTheme, baseDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := filterHeadings(headings, 2, 6)
	title := strings.TrimSpace(strings.TrimSuffix(filepath.Base(fullPath), filepath.Ext(fullPath)))
	if title == "" {
		title = "Document"
	}
	content := renderMarkdownPageContent(renderTOC(items), bodyHTML)
	htmlOut, err := renderPageHTML(title, docPageCSS(), template.HTML(renderFileHeader(relPath, codeTheme)), content, interactionScript(true, false), strings.Contains(bodyHTML, `class="mermaid"`))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := io.WriteString(w, htmlOut); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
