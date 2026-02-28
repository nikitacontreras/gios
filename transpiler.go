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
func ContainsString(s []string, e string) bool {
	for _, a := range s { if a == e { return true } }
	return false
}
`
	emptyPolyfill = `package %s
// Empty shim to bypass "cannot find package" errors
`
)

// generatePolyfills creates local shim packages to replace missing standard libraries.
func generatePolyfills(projectDir string) (string, error) {
	polyfillDir := filepath.Join(projectDir, ".gios_polyfills")
	os.MkdirAll(polyfillDir, 0755)

	// Create explicit shims
	fsDir := filepath.Join(polyfillDir, "io_fs")
	os.MkdirAll(fsDir, 0755)
	ioutil.WriteFile(filepath.Join(fsDir, "fs.go"), []byte(ioFsPolyfill), 0644)

	slicesDir := filepath.Join(polyfillDir, "slices")
	os.MkdirAll(slicesDir, 0755)
	ioutil.WriteFile(filepath.Join(slicesDir, "slices.go"), []byte(slicesPolyfill), 0644)

	// Create dynamic empty shims for the rest
	missing := []string{
		"cmp", "crypto_ecdh", "crypto_mlkem", "crypto_pbkdf2", 
		"crypto_tls_fipsonly", "embed", "iter", "log_slog", 
		"maps", "math_rand_v2", "net_netip",
	}

	for _, m := range missing {
		mDir := filepath.Join(polyfillDir, m)
		os.MkdirAll(mDir, 0755)
		pkgName := strings.Split(m, "_")[len(strings.Split(m, "_"))-1] // get last part for package name
		content := fmt.Sprintf(emptyPolyfill, pkgName)
		ioutil.WriteFile(filepath.Join(mDir, m+".go"), []byte(content), 0644)
	}

	return polyfillDir, nil
}

// TranspileLegacy modifies Go 1.18+ source code into Go 1.14 compatible code.
func TranspileLegacy(projectDir string, unsafe bool) error {
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

	// 3. Transpile all .go files in the directory (and subdirectories EXCEPT vendor by default)
	fset := token.NewFileSet()
	var transpiledCount int

	err = filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip the polyfills directory itself so we don't transpile our own shims
			if info.Name() == ".gios_polyfills" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			// Skip vendor directory unless --unsafe was passed
			if !unsafe && info.Name() == "vendor" {
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

		// 4. Map and Rewrite Future Imports
		for _, imp := range node.Imports {
			if imp.Path != nil {
				unquoted, _ := strconv.Unquote(imp.Path.Value)
				
				// Map of standard library paths to our polyfill dir names
				translations := map[string]string{
					"io/fs":               "io_fs",
					"slices":              "slices",
					"cmp":                 "cmp",
					"crypto/ecdh":         "crypto_ecdh",
					"crypto/mlkem":        "crypto_mlkem",
					"crypto/pbkdf2":       "crypto_pbkdf2",
					"crypto/tls/fipsonly": "crypto_tls_fipsonly",
					"embed":               "embed",
					"iter":                "iter",
					"log/slog":            "log_slog",
					"maps":                "maps",
					"math/rand/v2":        "math_rand_v2",
					"net/netip":           "net_netip",
				}

				if polyName, ok := translations[unquoted]; ok {
					imp.Path.Value = strconv.Quote(polyfillBaseImport + "/" + polyName)
					modified = true
				}
			}
		}

		// 5. Mapear e inyectar AST para los keywords "any"
		ast.Inspect(node, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				if ident.Name == "any" {
					ident.Name = "__GIOS_ANY__"
					modified = true
				}
			}
			return true
		})

		// Si el archivo fue tocado, lo regeneramos de AST a código fuente y guardamos
		if modified {
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, node); err == nil {
				finalCode := bytes.ReplaceAll(buf.Bytes(), []byte("__GIOS_ANY__"), []byte("interface{}"))
				if writeErr := ioutil.WriteFile(path, finalCode, info.Mode()); writeErr != nil {
					fmt.Printf("[!] Transpiler failed to write %s: %v\n", path, writeErr)
				} else {
					transpiledCount++
				}
			} else {
				fmt.Printf("[!] Transpiler failed to format %s: %v\n", path, err)
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
