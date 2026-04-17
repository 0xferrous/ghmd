package main

import (
	"html/template"
	"strings"
)

func interactionScript(includeCodeTheme, includeFragment bool) template.HTML {
	var b strings.Builder
	b.WriteString(`<script>
`)
	b.WriteString(`const themeKey = 'ghmd-theme';
`)
	if includeCodeTheme {
		b.WriteString(`const codeThemeKey = 'ghmd-code-theme';
`)
	}
	b.WriteString(`function systemTheme() {
  return matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}
`)
	b.WriteString(`function loadTheme() {
  try {
    const saved = localStorage.getItem(themeKey);
    if (saved === 'light' || saved === 'dark') return saved;
  } catch (_) {}
  return systemTheme();
}
`)
	if includeCodeTheme {
		b.WriteString(`function loadCodeTheme() {
  try {
    const saved = localStorage.getItem(codeThemeKey);
    if (saved) return saved;
  } catch (_) {}
  return '';
}
`)
	}
	b.WriteString(`function applyTheme(theme) {
  document.documentElement.dataset.theme = theme;
  const btn = document.getElementById('theme-toggle');
  if (btn) {
    const next = theme === 'dark' ? 'light' : 'dark';
    btn.textContent = next === 'dark' ? 'Dark' : 'Light';
    btn.setAttribute('aria-label', 'Switch to ' + next + ' theme');
  }
}
`)
	b.WriteString(`function setTheme(theme) {
  try { localStorage.setItem(themeKey, theme); } catch (_) {}
  applyTheme(theme);
}
`)
	if includeCodeTheme {
		b.WriteString(`function syncCodeTheme() {
  const select = document.getElementById('code-theme-select');
  if (!select) return;
  const u = new URL(window.location.href);
  const urlTheme = u.searchParams.get('code-theme') || '';
  const stored = loadCodeTheme();
  if (urlTheme) {
    try { localStorage.setItem(codeThemeKey, urlTheme); } catch (_) {}
    select.value = urlTheme;
    return;
  }
  if (stored) {
    select.value = stored;
    u.searchParams.set('code-theme', stored);
    setTimeout(() => window.location.replace(u.toString()), 0);
    return;
  }
  select.value = select.value || '';
}
`)
	}
	if includeFragment {
		b.WriteString(`function scrollToFragment() {
  if (!location.hash) return;
  const id = decodeURIComponent(location.hash.slice(1));
  const el = document.getElementById(id);
  if (!el) return;
  el.scrollIntoView({ block: 'start' });
  if (history.replaceState) {
    history.replaceState(null, '', location.hash);
  }
}
`)
	}
	b.WriteString(`window.addEventListener('load', () => {
  applyTheme(loadTheme());
  const btn = document.getElementById('theme-toggle');
  if (btn) {
    btn.addEventListener('click', () => {
      const current = document.documentElement.dataset.theme || loadTheme();
      setTheme(current === 'dark' ? 'light' : 'dark');
    });
  }
`)
	if includeCodeTheme {
		b.WriteString(`  const codeThemeSelect = document.getElementById('code-theme-select');
  if (codeThemeSelect) {
    syncCodeTheme();
    codeThemeSelect.addEventListener('change', () => {
      const u = new URL(window.location.href);
      u.searchParams.set('code-theme', codeThemeSelect.value);
      try { localStorage.setItem(codeThemeKey, codeThemeSelect.value); } catch (_) {}
      setTimeout(() => { window.location.href = u.toString(); }, 0);
    });
  }
`)
	}
	if includeFragment {
		b.WriteString(`  requestAnimationFrame(scrollToFragment);
  setTimeout(scrollToFragment, 50);
  setTimeout(scrollToFragment, 250);
`)
	}
	b.WriteString(`});
`)
	if includeFragment {
		b.WriteString(`window.addEventListener('hashchange', scrollToFragment);
`)
	}
	b.WriteString(`</script>`)
	return template.HTML(b.String())
}

func baseThemeCSS() template.CSS {
	return template.CSS(`
    :root {
      color-scheme: light dark;
      --bg: #ffffff;
      --fg: #24292f;
      --muted: #57606a;
      --border: #d0d7de;
      --panel: #f6f8fa;
      --link: #0969da;
      --link-hover: #0550ae;
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
        --link-hover: #79c0ff;
        --code-bg: #161b22;
      }
    }
    html[data-theme="light"] {
      color-scheme: light;
      --bg: #ffffff;
      --fg: #24292f;
      --muted: #57606a;
      --border: #d0d7de;
      --panel: #f6f8fa;
      --link: #0969da;
      --link-hover: #0550ae;
      --code-bg: #f6f8fa;
    }
    html[data-theme="dark"] {
      color-scheme: dark;
      --bg: #0d1117;
      --fg: #c9d1d9;
      --muted: #8b949e;
      --border: #30363d;
      --panel: #161b22;
      --link: #58a6ff;
      --link-hover: #79c0ff;
      --code-bg: #161b22;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--fg);
      line-height: 1.6;
    }
    a { color: var(--link); text-decoration: none; }
    a:hover { text-decoration: underline; }
`)
}

func baseThemeCSSString() string { return string(baseThemeCSS()) }

func docPageCSS() template.CSS {
	return template.CSS(`
    .doc-header {
      position: sticky;
      top: 0;
      z-index: 20;
      max-width: 1440px;
      margin: 0 auto;
      padding: 24px 24px 12px;
      background: color-mix(in srgb, var(--bg) 92%, transparent);
      backdrop-filter: blur(8px);
      border-bottom: 1px solid var(--border);
    }
    .doc-header .titlebar {
      display: flex;
      font-family: monospace;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      flex-wrap: wrap;
    }
    .doc-header h1 {
      margin: 0;
      font-size: .9rem;
      line-height: 1.25;
      word-break: break-word;
      font-family: monospace;
      font-weight: 400;
    }
    .doc-header .leaf { color: var(--fg); }
    .doc-header .breadcrumb a { color: var(--link); }
    .doc-header .breadcrumb a:hover { color: var(--link-hover); }
    .doc-header .breadcrumb .sep { color: var(--muted); }
    .doc-header .controls {
      display: flex;
      align-items: center;
      gap: 8px;
      flex-wrap: wrap;
    }
    .doc-header .code-theme {
      display: flex;
      align-items: center;
      gap: 6px;
      color: var(--muted);
      font-size: 12px;
    }
    .doc-header .code-theme span {
      text-transform: uppercase;
      letter-spacing: .04em;
    }
    .markdown-body code,
    .markdown-body pre,
    .markdown-body kbd,
    .markdown-body samp { font-family: monospace; }
    .doc-header .code-theme-select {
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
    .doc-header .code-theme-select:hover { background: var(--border); }
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
    .markdown-body { min-width: 0; max-width: 100%; }
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
    .markdown-body .heading-anchor:focus { opacity: 1; }
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
    .markdown-body code { font-family: monospace; font-size: .95em; }
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
    .markdown-body .task-list-item { list-style-type: none; }
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
    .markdown-body .footnotes ol { padding-left: 1.5em; }
    .markdown-body .footnotes li + li { margin-top: 8px; }
    .markdown-body .footnote-ref,
    .markdown-body .footnote-backref { text-decoration: none; }
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
`)
}

func dirPageCSS() template.CSS {
	return template.CSS(`
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
    tbody td.modified { text-align: right; }
    thead th.modified a {
      justify-content: flex-end;
      width: 100%;
    }
    thead th.active { color: var(--fg); }
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
      tbody td.modified { width: 9rem; }
    }
`)
}
