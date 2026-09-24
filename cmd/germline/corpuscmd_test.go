package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/justinstimatze/germline/internal/project"
)

// TestLawNamesDoesNotAcceptAHelperFunctionName is the regression case a
// resection audit found: a helper function like allLaws, named beside the
// laws it assembles but never itself a law, must not validate as one. The
// scanner used to treat every identifier under laws/ as a law name, which
// let `germline close -as law -law allLaws ...` succeed against a name
// nobody registered.
func TestLawNamesDoesNotAcceptAHelperFunctionName(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeLawsFile(t, dir, `package laws

import "github.com/justinstimatze/germline/pkg/law"

func allLaws() []law.Law {
	return []law.Law{
		{Name: "QueryRoundTrips", Statement: "x", Prop: func(int) bool { return true }},
	}
}
`)
	p := &project.Project{Root: dir}
	names := lawNames(p)
	if slices.Contains(names, "allLaws") {
		t.Errorf("lawNames(%v) contains the helper function name allLaws, want only the law it returns", names)
	}
	if !slices.Contains(names, "QueryRoundTrips") {
		t.Errorf("lawNames(%v) = %v, want it to contain the registered law QueryRoundTrips", names, names)
	}
}

// TestLawNamesFindsConstructorCalls covers the five families that take a
// single name and the Lens family, which registers three names under
// suffixes rather than the bare name it was called with.
func TestLawNamesFindsConstructorCalls(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeLawsFile(t, dir, `package laws

import "github.com/justinstimatze/germline/pkg/law"

func register() {
	law.Register(law.Roundtrip("ParseRendersBack", nil, nil))
	law.Register(law.Lens("Workers",
		func(c int) int { return c },
		func(c, v int) int { return v },
	)...)
}
`)
	p := &project.Project{Root: dir}
	names := lawNames(p)
	for _, want := range []string{"ParseRendersBack", "WorkersGetPut", "WorkersPutGet", "WorkersPutPut"} {
		if !slices.Contains(names, want) {
			t.Errorf("lawNames(%v) = %v, want it to contain %s", names, names, want)
		}
	}
	if slices.Contains(names, "Workers") {
		t.Errorf("lawNames(%v) contains the bare name Workers, but Lens never registers one under it", names)
	}
}

func writeLawsFile(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, "laws")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "laws.go"), []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}
