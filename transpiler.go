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
	"strconv"
	"strings"
)

// The polyfills string constants (Shims for missing Go 1.18+ libraries)
const (
	ioFsPolyfill = `package fs
import "os"
type FileInfo = os.FileInfo
type FileMode = os.FileMode
type PathError = os.PathError
`
	slicesPolyfill = `package slices
// This is a minimal shim for the new slices package.
// Production transpilers would contain the full generic-monomorphized algorithms.
func ContainsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
`
)

// generatePolyfills creates local shim packages to replace missing standard libraries.
func generatePolyfills(projectDir string) (string, error) {
	polyfillDir := filepath.Join(projectDir, ".gios_polyfills")
	os.MkdirAll(polyfillDir, 0755)

	// Create io/fs shim
	fsDir := filepath.Join(polyfillDir, "io_fs")
	os.MkdirAll(fsDir, 0755)
	ioutil.WriteFile(filepath.Join(fsDir, "fs.go"), []byte(ioFsPolyfill), 0644)

	// Create slices shim
	slicesDir := filepath.Join(polyfillDir, "slices")
	os.MkdirAll(slicesDir, 0755)
	ioutil.WriteFile(filepath.Join(slicesDir, "slices.go"), []byte(slicesPolyfill), 0644)

	return polyfillDir, nil
}

// TranspileLegacy modifies Go 1.18+ source code into Go 1.14 compatible code.
func TranspileLegacy(projectDir string) error {
	fmt.Println("[gios] [Transpiler] Starting automated backport to Go 1.14...")

	// 1. Downgrade go.mod and get Module Name
	modPath := filepath.Join(projectDir, "go.mod")
	var moduleName string
	if _, err := os.Stat(modPath); err == nil {
		content, _ := ioutil.ReadFile(modPath)
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "module ") {
				moduleName = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			}
			if strings.HasPrefix(line, "go ") {
				lines[i] = "go 1.14" // Bruteforce the downgrade
			} else if strings.HasPrefix(line, "toolchain ") {
				lines[i] = "// " + line // comment out toolchain
			}
		}
		ioutil.WriteFile(modPath, []byte(strings.Join(lines, "\n")), 0644)
		fmt.Println("[gios] [Transpiler] Downgraded go.mod to Go 1.14")
	}

	if moduleName == "" {
		fmt.Println("[!] [Transpiler] Warning: Could not detect module name in go.mod. Polyfill imports might fail.")
		moduleName = "github.com/unknown/module"
	}

	// 2. Generate Polyfills inside the target project
	fmt.Println("[gios] [Transpiler] Generating polyfills for future standard libraries...")
	_, err := generatePolyfills(projectDir)
	if err != nil {
		return err
	}
	polyfillBaseImport := moduleName + "/.gios_polyfills"

	// 3. Transpile all .go files in the directory (and subdirectories EXCEPT vendor)
	fset := token.NewFileSet()
	var transpiledCount int

	err = filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip the polyfills directory itself so we don't transpile our own shims
			// We also skip vendor to avoid destroying third party libs blindly (for now)
			if info.Name() == ".gios_polyfills" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Parse the file
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil 
		}

		modified := false

		// 4. Mapear y Reescribir los IMPORTS del Futuro
		for _, imp := range node.Imports {
			if imp.Path != nil {
				unquoted, _ := strconv.Unquote(imp.Path.Value)
				switch unquoted {
				case "io/fs":
					imp.Path.Value = strconv.Quote(polyfillBaseImport + "/io_fs")
					modified = true
				case "slices":
					imp.Path.Value = strconv.Quote(polyfillBaseImport + "/slices")
					modified = true
				}
			}
		}

		// 5. Mapear e inyectar AST para los keywords "any"
		ast.Inspect(node, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				if ident.Name == "any" {
					ident.Name = "interface{}"
					modified = true
				}
			}
			return true
		})

		// Si el archivo fue tocado, lo regeneramos de AST a código fuente y guardamos
		if modified {
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, node); err == nil {
				ioutil.WriteFile(path, buf.Bytes(), info.Mode())
				transpiledCount++
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("[gios] [Transpiler] Successfully backported %d Go files and injected polyfills.\n", transpiledCount)
	return nil
}
