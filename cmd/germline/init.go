package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/justinstimatze/germline/internal/project"
)

//go:embed all:templates
var templates embed.FS

// Adoption is meant to be one command and then a real decision. init writes
// the artifacts and nothing else: no config file, no generated code, no
// framework in the import graph. What it cannot write is the first boundary
// entry and the first decision, which are the two things that make a project
// governed rather than merely scaffolded, so it says so and stops.

func cmdInit(e *env, args []string) error {
	fset := flags(e, "init")
	force := fset.Bool("force", false, "overwrite files that already exist")
	p, err := e.project(fset, args)
	if err != nil {
		return err
	}

	written, skipped, err := writeTemplates(p.Root, *force)
	if err != nil {
		return err
	}
	for _, f := range written {
		fmt.Fprintf(e.stdout, "wrote %s\n", f)
	}
	for _, f := range skipped {
		fmt.Fprintf(e.stdout, "kept  %s (already there)\n", f)
	}

	fmt.Fprintln(e.stdout, "\nTwo things this cannot write for you:")
	fmt.Fprintf(e.stdout, "  · the first decision, in %s\n", project.DecisionsDir+"/")
	fmt.Fprintln(e.stdout, "  · the first boundary entry: one thing users could observe that is")
	fmt.Fprintln(e.stdout, "    not promised, and why. That is a human's decision, and a close only")
	fmt.Fprintln(e.stdout, "    cites an entry once a human has committed it. It is cheapest now;")
	fmt.Fprintln(e.stdout, "    a boundary that waits until there are dependents never gets written.")
	fmt.Fprintln(e.stdout, "\nThen: germline check")
	return nil
}

// writeTemplates copies the embedded artifacts into root, never overwriting
// unless asked. Adopting germline in a repository that already has a
// BOUNDARY.md must not lose it.
func writeTemplates(root string, force bool) (written, skipped []string, err error) {
	err = fs.WalkDir(templates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(path, "templates/")
		dst := filepath.Join(root, rel)
		if _, statErr := os.Stat(dst); statErr == nil && !force {
			skipped = append(skipped, rel)
			return nil
		}
		body, err := templates.ReadFile(path)
		if err != nil {
			return err
		}
		if rel == project.ManifestFile {
			body = []byte(strings.Replace(string(body), `"estimated_at": ""`,
				fmt.Sprintf(`"estimated_at": %q`, time.Now().UTC().Format(time.DateOnly)), 1))
		}
		if mkErr := os.MkdirAll(filepath.Dir(dst), 0o750); mkErr != nil {
			return mkErr
		}
		if wErr := os.WriteFile(dst, body, 0o600); wErr != nil {
			return wErr
		}
		written = append(written, rel)
		return nil
	})
	return written, skipped, err
}
