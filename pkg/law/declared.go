package law

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Declared reads the law names a directory declares.
//
// This exists because the obvious thing was wrong. The laws metric used to
// count Default.Laws(), the registry inside whatever binary was doing the
// counting, which for a checker looking at somebody else's project is always
// empty — and it was empty for germline's own repository too, so the number
// had never measured anything since the day it was written. A project's laws
// are Go source in its laws directory, and the only way a checker running
// outside that project can count them is to read it.
//
// Two shapes count as a declaration: a composite literal with a Name field
// holding a string, which is how a Law value is written out, and a call on the
// law package whose first argument is a string, which is how the families are
// used. A family that yields several laws from one name — Lens is the one that
// does — counts once, because one name is what a reader has to find.
func Declared(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, nil //nolint:nilerr // a project with no laws directory has no laws
	}
	seen := map[string]bool{}
	fset := token.NewFileSet()
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil //nolint:nilerr // an unreadable file is not a reason to refuse a count
		}
		f, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return nil //nolint:nilerr // a file that does not parse declares nothing
		}
		for _, name := range namesIn(f) {
			seen[name] = true
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// namesIn collects the law names one file declares.
func namesIn(f *ast.File) []string {
	var names []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.KeyValueExpr:
			key, ok := node.Key.(*ast.Ident)
			if !ok || key.Name != "Name" {
				return true
			}
			if s, isString := stringLit(node.Value); isString {
				names = append(names, s)
			}
		case *ast.CallExpr:
			sel, ok := node.Fun.(*ast.SelectorExpr)
			if !ok || len(node.Args) == 0 {
				return true
			}
			pkg, isIdent := sel.X.(*ast.Ident)
			if !isIdent || pkg.Name != "law" {
				return true
			}
			if s, isString := stringLit(node.Args[0]); isString {
				names = append(names, s)
			}
		}
		return true
	})
	return names
}

// stringLit unquotes a string literal, and reports false for anything else.
func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil || s == "" {
		return "", false
	}
	return s, true
}
