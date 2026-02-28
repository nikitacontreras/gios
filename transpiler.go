package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// TranspileLegacy modifies Go 1.18+ source code into Go 1.14 compatible code.
// Currently it implements a Proof of Concept:
// - Downgrades go.mod to go 1.14
// - Replaces the "any" keyword with "interface{}"
func TranspileLegacy(projectDir string) error {
	fmt.Println("[gios] [Transpiler] Starting automated backport to Go 1.14...")

	// 1. Downgrade go.mod
	modPath := filepath.Join(projectDir, "go.mod")
	if _, err := os.Stat(modPath); err == nil {
		content, _ := ioutil.ReadFile(modPath)
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "go ") {
				lines[i] = "go 1.14"
			} else if strings.HasPrefix(line, "toolchain ") {
				lines[i] = "// " + line // comment out toolchain
			}
		}
		ioutil.WriteFile(modPath, []byte(strings.Join(lines, "\n")), 0644)
		fmt.Println("[gios] [Transpiler] Downgraded go.mod to Go 1.14")
	}

	// 2. Transpile all .go files in the directory (and subdirectories)
	fset := token.NewFileSet()
	var transpiledCount int

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Parse the file
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files we can't parse (could be due to newer syntax we haven't handled yet)
		}

		modified := false

		// Inspect the AST and modify it
		ast.Inspect(node, func(n ast.Node) bool {
			// Find identifiers named "any" and replace them with "interface{}"
			if ident, ok := n.(*ast.Ident); ok {
				if ident.Name == "any" {
					// In Go AST, "any" is just an identifier if it's used as a type.
					// We can rename it to "interface{}" as a text hack, but properly
					// we'd replace the node. For now, renaming the Ident works for the printer!
					ident.Name = "interface{}"
					modified = true
				}
			}
			return true
		})

		if modified {
			// Write the modified AST back to a buffer
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, node); err != nil {
				return nil // Skip on format error
			}
			
			// Replace the original file
			ioutil.WriteFile(path, buf.Bytes(), info.Mode())
			transpiledCount++
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("[gios] [Transpiler] Successfully backported %d Go files.\n", transpiledCount)
	return nil
}
