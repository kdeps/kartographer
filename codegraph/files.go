package codegraph

import (
	"context"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
)

// GoFiles lists the Go files of the module at root that the graph reads. Git
// decides when it can: the files git tracks, or would track, honouring
// .gitignore. Otherwise every .go file outside the folders the rules skip.
// The paths are absolute when root is.
func GoFiles(ctx context.Context, root string) ([]string, error) {
	if !configured {
		return nil, ErrNotConfigured
	}
	if files := gitGoFiles(ctx, root); files != nil {
		return files, nil
	}
	return walkGoFiles(root), nil
}

// gitGoFiles lists root's .go files that git tracks or would track, or nil
// when root is not in a git work tree, git is missing, or it lists none.
func gitGoFiles(ctx context.Context, root string) []string {
	out, err := exec.CommandContext(ctx, "git", "-C", root, "ls-files",
		"--cached", "--others", "--exclude-standard", "-z", "--", "*.go").Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f == "" || skippedPath(f) {
			continue
		}
		files = append(files, filepath.Join(root, filepath.FromSlash(f)))
	}
	return files
}

// walkGoFiles lists root's .go files outside the skipped folders.
func walkGoFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil //nolint:nilerr // unreadable paths are skipped
		case d.IsDir():
			if p != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
		case strings.HasSuffix(d.Name(), ".go"):
			files = append(files, p)
		}
		return nil
	})
	return files
}

// skippedPath reports whether a slash-separated path lies in a skipped folder.
func skippedPath(rel string) bool {
	parts := strings.Split(rel, "/")
	for _, dir := range parts[:len(parts)-1] {
		if skipDir(dir) {
			return true
		}
	}
	return false
}

// skipDir reports whether the rules skip a folder with this name.
func skipDir(name string) bool {
	if rules.Paths.SkipHidden && strings.HasPrefix(name, ".") {
		return true
	}
	for _, s := range rules.Paths.SkipDirs {
		if name == s {
			return true
		}
	}
	return false
}
