package main

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/kdeps/kartographer/codegraph"
)

// changesCommand prints the symbols that the working tree changes against base
// touch, and the symbols those can affect.
func changesCommand(out io.Writer, dir, base string, depth, limit int, noCache bool) error {
	diff, err := gitDiff(dir, base)
	if err != nil {
		return err
	}
	ix, err := load(dir, noCache)
	if err != nil {
		return err
	}
	printChanges(out, ix, diff, depth, limit)
	return nil
}

// gitDiff is the unified diff of dir against base, with paths relative to dir,
// so it matches a module that is below the repository root.
func gitDiff(dir, base string) (string, error) {
	raw, err := exec.Command("git", "-C", dir, "diff", "--unified=0", "--relative", base).Output() //nolint:gosec // base is the user's revision
	if err != nil {
		return "", fmt.Errorf("git diff %s: %w", base, err)
	}
	return string(raw), nil
}

func printChanges(out io.Writer, ix *codegraph.Index, diff string, depth, limit int) {
	fmt.Fprintln(out, "changed:")
	changed := ix.Changed(diff)
	if len(changed) == 0 {
		fmt.Fprintln(out, "  none")
	}
	for _, s := range changed {
		fmt.Fprintf(out, "  %s  %s:%d\n", s.Name, s.File, s.Line)
	}
	fmt.Fprintln(out, "impact:")
	for _, s := range ix.Impact(diff, depth, limit) {
		fmt.Fprintf(out, "  %s  %s:%d\n", s.Name, s.File, s.Line)
	}
}
