package messagecleanup

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestScheduledAppCodeUsesMessageCleanupContract(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	appDir := filepath.Dir(filepath.Dir(currentFile))
	allowed := map[string]bool{
		"delivery/compensation.go":         true, // Deletes a message sent in the same call path.
		"messagecleanup/messagecleanup.go": true, // Owns the age-aware deletion contract.
	}

	err := filepath.WalkDir(appDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(appDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if allowed[rel] {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "DeleteMessage" {
				t.Errorf("%s calls DeleteMessage directly; scheduled app cleanup must use messagecleanup", rel)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
