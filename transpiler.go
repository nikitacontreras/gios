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

func generatePolyfills() (string, error) {
	polyfillDir := filepath.Join(giosDir, "polyfills")
	os.MkdirAll(polyfillDir, 0755)

	modContent := "module gios/polyfills\n\ngo 1.14\n"
	ioutil.WriteFile(filepath.Join(polyfillDir, "go.mod"), []byte(modContent), 0644)

	fsDir := filepath.Join(polyfillDir, "io_fs")
	os.MkdirAll(fsDir, 0755)
	ioutil.WriteFile(filepath.Join(fsDir, "fs.go"), []byte(ioFsPolyfill), 0644)

	slicesDir := filepath.Join(polyfillDir, "slices")
	os.MkdirAll(slicesDir, 0755)
	ioutil.WriteFile(filepath.Join(slicesDir, "slices.go"), []byte(slicesPolyfill), 0644)

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

func TranspileLegacy(projectDir string, unsafe bool) error {
	fmt.Println("[gios] [Transpiler] Starting automated backport to Go 1.14...")

	os.RemoveAll(filepath.Join(projectDir, ".gios_polyfills"))

	modPath := filepath.Join(projectDir, "go.mod")
	var moduleName string
	if _, err := os.Stat(modPath); err == nil {
		content, _ := ioutil.ReadFile(modPath)
		lines := strings.Split(string(content), "\n")
		
		polyfillDir := filepath.Join(giosDir, "polyfills")
		replaceRule := fmt.Sprintf("replace gios/polyfills => %s", polyfillDir)
		foundReplace := false

		for i, line := range lines {
			if strings.HasPrefix(line, "module ") {
				moduleName = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			}
			if strings.HasPrefix(line, "go ") {
				lines[i] = "go 1.14" // Bruteforce the downgrade
			} else if strings.HasPrefix(line, "toolchain ") {
				lines[i] = "// " + line // comment out toolchain
			}
			if strings.HasPrefix(line, "replace gios/polyfills") {
				lines[i] = replaceRule
				foundReplace = true
			}
		}
		
		if !foundReplace {
			lines = append(lines, "", "require gios/polyfills v0.0.0", replaceRule)
		}

		ioutil.WriteFile(modPath, []byte(strings.Join(lines, "\n")), 0644)
		fmt.Println("[gios] [Transpiler] Downgraded go.mod and injected polyfill replace rules")
	}

	if moduleName == "" {
		fmt.Println("[!] [Transpiler] Warning: Could not detect module name in go.mod. Standard imports might fail.")
		moduleName = "example"
	}

	fmt.Println("[gios] [Transpiler] Ensuring global polyfills for future standard libraries...")
	_, err := generatePolyfills()
	if err != nil {
		return err
	}
	polyfillBaseImport := "gios/polyfills"

	fset := token.NewFileSet()
	var transpiledCount int

	err = filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".gios_polyfills" || info.Name() == ".git" {
				return filepath.SkipDir
			}

			if !unsafe && info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err == nil {
			strContent := string(content)
			if strings.Contains(strContent, "//go:build ") && !strings.Contains(strContent, "// +build ") {
				lines := strings.Split(strContent, "\n")
				for i, line := range lines {
					if strings.HasPrefix(line, "//go:build ") {
						tag := strings.TrimPrefix(line, "//go:build ")
						lines[i] = line + "\n// +build " + tag
						break
					}
				}
				ioutil.WriteFile(path, []byte(strings.Join(lines, "\n")), info.Mode())
			}
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
			// Helper to replace 'any' in a type expression
			replaceIfAny := func(e *ast.Expr) {
				if e == nil {
					return
				}
				if ident, ok := (*e).(*ast.Ident); ok && ident.Name == "any" {
					ident.Name = "__GIOS_ANY__"
					modified = true
				}
			}

			switch x := n.(type) {
			case *ast.Field:
				replaceIfAny(&x.Type)
			case *ast.ValueSpec:
				replaceIfAny(&x.Type)
			case *ast.TypeSpec:
				replaceIfAny(&x.Type)
			case *ast.TypeAssertExpr:
				replaceIfAny(&x.Type)
			case *ast.ArrayType:
				replaceIfAny(&x.Elt)
			case *ast.MapType:
				replaceIfAny(&x.Key)
				replaceIfAny(&x.Value)
			case *ast.ChanType:
				replaceIfAny(&x.Value)
			case *ast.StarExpr:
				replaceIfAny(&x.X)
			case *ast.CompositeLit:
				replaceIfAny(&x.Type)
			case *ast.Ellipsis:
				replaceIfAny(&x.Elt)
			case *ast.IndexExpr: // Handle generic types like Map[string, any]
				replaceIfAny(&x.Index)
			}
			return true
		})

		// 6. Convertir __GIOS_ANY__ a interface{}
		if modified {
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, node); err == nil {
				finalCode := buf.Bytes()
				finalCode = bytes.ReplaceAll(finalCode, []byte("__GIOS_ANY__"), []byte("interface{}"))
				if writeErr := ioutil.WriteFile(path, finalCode, info.Mode()); writeErr != nil {
					fmt.Printf("[!] Transpiler failed to write %s: %v\n", path, writeErr)
				} else {
					transpiledCount++
				}
			} else {
				fmt.Printf("[!] Transpiler failed to format %s: %v\n", path, err)
			}
			return nil
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("[gios] [Transpiler] Walked through files complete.\n"); fmt.Printf("[gios] [Transpiler] Successfully backported %d Go files and injected polyfills.\n", transpiledCount)
	return nil
}
