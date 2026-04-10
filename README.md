# ghmd

`ghmd` is a small Go CLI for rendering Markdown to HTML using [Goldmark](https://github.com/yuin/goldmark).

It aims to be close to GitHub-flavored Markdown rendering and includes:

- GitHub-flavored Markdown features
- syntax highlighting for fenced code blocks
- selectable highlighting themes
- table of contents generation
- clickable heading anchors
- Mermaid diagram support
- footnotes
- raw HTML support

## Usage

Render a Markdown file to HTML:

```bash
go run . -i input.md -o output.html
```

Or read from stdin and write to stdout:

```bash
cat input.md | go run . > output.html
```

## Flags

- `-i` — input Markdown file, defaults to stdin
- `-o` — output HTML file, defaults to stdout
- `-title` — override the HTML page title
- `-theme` — syntax highlighting theme
- `-toc-min-level` — minimum heading level included in the TOC
- `-toc-max-level` — maximum heading level included in the TOC

## Theme options

Use `-theme` to choose the code highlighting theme.

Available values are the Chroma styles exposed by Goldmark highlighting, including:

`abap`, `algol`, `algol_nu`, `arduino`, `autumn`, `average`, `base16-snazzy`, `borland`, `bw`, `colorful`, `doom-one`, `doom-one2`, `dracula`, `emacs`, `friendly`, `fruity`, `github`, `gruvbox`, `hr_high_contrast`, `hrdark`, `igor`, `lovelace`, `manni`, `monokai`, `monokailight`, `murphy`, `native`, `nord`, `onesenterprise`, `paraiso-dark`, `paraiso-light`, `pastie`, `perldoc`, `pygments`, `rainbow_dash`, `rrt`, `solarized-dark`, `solarized-dark256`, `solarized-light`, `swapoff`, `tango`, `trac`, `vim`, `vs`, `vulcan`, `witchhazel`, `xcode`, `xcode-dark`

## Example

Markdown input:

````md
# Hello

Some `inline code`.

## Section

```go
fmt.Println("hi")
```

```mermaid
graph TD
  A --> B
```

A footnote reference.[^1]

[^1]: Footnote text.
````

## Notes

- Heading anchors are inserted automatically.
- The table of contents includes headings from level 2 through 6 by default.
- Mermaid diagrams are rendered client-side using the Mermaid CDN.
- Raw HTML is allowed in the rendered output.

## Development

Format and test the project:

```bash
gofmt -w main.go
go test ./...
```

## License

No license has been specified yet.
