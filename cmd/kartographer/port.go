package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kdeps/kartographer/codegraph"
)

// hookBegin and hookEnd mark the block this tool owns inside a git hook, so
// install and uninstall touch only that block and keep whatever else a hook has.
const (
	hookBegin = "# >>> kartographer >>>"
	hookEnd   = "# <<< kartographer <<<"
)

// hookNames are the git hooks that rebuild the index: after a commit and after
// a branch switch.
//
//nolint:gochecknoglobals // fixed list
var hookNames = []string{"post-commit", "post-checkout"}

// hookBlock rebuilds the index when KARTOGRAPHER_RULES names the rules file.
// Without it the hook does nothing, so the rules are never guessed.
const hookBlock = hookBegin + `
if [ -n "$KARTOGRAPHER_RULES" ]; then
  kartographer -rules "$KARTOGRAPHER_RULES" update "$(git rev-parse --show-toplevel)" >/dev/null 2>&1 || true
fi
` + hookEnd

// hookCommand installs, removes or reports the git hooks of the repository at dir.
func hookCommand(out io.Writer, dir, sub string) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return fmt.Errorf("%s is not a git repository", dir)
	}
	hooks := filepath.Join(gitDir, "hooks")
	switch sub {
	case "install":
		if err := os.MkdirAll(hooks, 0o755); err != nil {
			return err
		}
		for _, name := range hookNames {
			if err := writeHook(filepath.Join(hooks, name), true); err != nil {
				return err
			}
		}
		_, _ = fmt.Fprintf(out, "installed %s in %s\n", strings.Join(hookNames, ", "), hooks)
	case "uninstall":
		for _, name := range hookNames {
			if err := writeHook(filepath.Join(hooks, name), false); err != nil {
				return err
			}
		}
		_, _ = fmt.Fprintln(out, "uninstalled")
	case "status":
		for _, name := range hookNames {
			raw, _ := os.ReadFile(filepath.Join(hooks, name))
			state := "not installed"
			if bytes.Contains(raw, []byte(hookBegin)) {
				state = "installed"
			}
			_, _ = fmt.Fprintf(out, "%s: %s\n", name, state)
		}
	default:
		return fmt.Errorf("%w: hook needs install, uninstall or status", errUsage)
	}
	return nil
}

// writeHook adds the block to a hook (install) or removes it (!install). A hook
// left with nothing but its shebang is deleted.
func writeHook(path string, install bool) error {
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := stripBlock(string(raw))
	if install {
		if strings.TrimSpace(text) == "" {
			text = "#!/bin/sh\n"
		}
		text = strings.TrimRight(text, "\n") + "\n" + hookBlock + "\n"
		return os.WriteFile(path, []byte(text), 0o755) //nolint:gosec // git hooks must be executable
	}
	if strings.TrimSpace(text) == "" || strings.TrimSpace(text) == "#!/bin/sh" {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return os.Remove(path)
	}
	return os.WriteFile(path, []byte(text), 0o755) //nolint:gosec // git hooks must be executable
}

// stripBlock removes the tool's block from a hook's text.
func stripBlock(text string) string {
	start := strings.Index(text, hookBegin)
	if start < 0 {
		return text
	}
	end := strings.Index(text, hookEnd)
	if end < start {
		return text
	}
	end += len(hookEnd)
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return text[:start] + text[end:]
}

// watchCommand indexes dir, then rebuilds whenever a file under it changes. It
// checks every interval and stops after rounds checks (0 means never).
func watchCommand(out io.Writer, dir string, interval time.Duration, rounds int, noCache bool) error {
	last := ""
	for i := 0; rounds <= 0 || i < rounds; i++ {
		fp, err := codegraph.Fingerprint(dir)
		if err != nil {
			return err
		}
		if fp != last {
			ix, err := load(dir, noCache)
			if err != nil {
				_, _ = fmt.Fprintf(out, "rebuild failed: %v\n", err)
			} else {
				symbols, edges := ix.Stats()
				_, _ = fmt.Fprintf(out, "indexed %s: %d symbols, %d call edges\n", ix.Module, symbols, edges)
			}
			last = fp
		}
		if rounds > 0 && i == rounds-1 {
			break
		}
		time.Sleep(interval)
	}
	return nil
}

// exportCommand writes the index in a format. Single-file formats go to stdout
// or to path; obsidian and wiki write a directory named by path.
func exportCommand(out io.Writer, format, path, dir string, noCache bool) error {
	ix, err := load(dir, noCache)
	if err != nil {
		return err
	}
	switch format {
	case "obsidian", "wiki":
		if path == "" {
			return fmt.Errorf("%w: export %s needs -o DIR", errUsage, format)
		}
		if format == "obsidian" {
			err = codegraph.WriteObsidian(path, ix)
		} else {
			err = codegraph.WriteWiki(path, ix)
		}
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "wrote %s\n", path)
		return nil
	case "graphml", "html", "svg", "neo4j", "falkordb":
	default:
		return fmt.Errorf("%w: export supports graphml, html, svg, obsidian, wiki, neo4j, falkordb", errUsage)
	}
	write := map[string]func(io.Writer, *codegraph.Index) error{
		"graphml":  codegraph.WriteGraphML,
		"html":     codegraph.WriteHTML,
		"svg":      codegraph.WriteSVG,
		"neo4j":    codegraph.WriteCypher,
		"falkordb": codegraph.WriteCypher,
	}[format]
	if path == "" {
		return write(out, ix)
	}
	f, err := os.Create(path) //nolint:gosec // the output path is the user's choice
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if err := write(f, ix); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "wrote %s\n", path)
	return nil
}

// addCommand fetches an http or https URL into dir as a markdown file with the
// source and the optional author and contributor in its front matter.
func addCommand(out io.Writer, url, dir, author, contributor string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("%w: add needs an http or https URL", errUsage)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url) //nolint:noctx // a one-off CLI fetch
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAddBytes))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var doc bytes.Buffer
	doc.WriteString("---\n")
	_, _ = fmt.Fprintf(&doc, "source: %q\n", url)
	_, _ = fmt.Fprintf(&doc, "fetched: %q\n", time.Now().UTC().Format(time.RFC3339))
	if author != "" {
		_, _ = fmt.Fprintf(&doc, "author: %q\n", author)
	}
	if contributor != "" {
		_, _ = fmt.Fprintf(&doc, "contributor: %q\n", contributor)
	}
	doc.WriteString("---\n\n")
	doc.Write(body)
	name := filepath.Join(dir, addFileName(url))
	if err := os.WriteFile(name, doc.Bytes(), 0o600); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "saved %s\n", name)
	return nil
}

// maxAddBytes caps one fetched page.
const maxAddBytes = 16 << 20

// addFileName derives a file name from a URL: its host and path, slugged.
func addFileName(url string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	s = strings.Trim(strings.NewReplacer("/", "_", "?", "_", "&", "_", "=", "_", ":", "_").Replace(s), "_")
	if s == "" {
		s = "page"
	}
	return s + ".md"
}
