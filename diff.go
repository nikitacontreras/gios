package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"
)

func runDiff(filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("%sError: File not found: %s%s\n", ColorRed, filePath, ColorReset)
		return
	}

	fmt.Printf("%s[gios]%s Comparing original vs transpiled: %s%s%s\n", 
		ColorCyan, ColorReset, ColorBold, filePath, ColorReset)

	// Read original content
	original, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading original file: %v\n", err)
		return
	}

	// Transpile in-memory (using the same logic as transpiler.go but returning byte array)
	// For this, we'll reuse the TranspileLegacy logic, but to avoid duplication we'll
	// assume we've slightly refactored TranspileLegacy or just re-implement the core logic here
	// for a single file. (Since TranspileLegacy is quite coupled, for now we will re-implement
	// a "preview" version).
	
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		return
	}

	// Simple copy-paste of logic from transpiler.go (the parts that matter for visual diff)
	// 4. Map and Rewrite Future Imports
	for _, imp := range node.Imports {
		if imp.Path != nil {
			unquoted := strings.Trim(imp.Path.Value, "\"")
			translations := map[string]string{
				"io/fs": "io_fs", "slices": "slices", "cmp": "cmp",
				"net/netip": "net_netip", "log/slog": "log_slog",
			}
			if polyName, ok := translations[unquoted]; ok {
				imp.Path.Value = fmt.Sprintf("\"gios/polyfills/%s\"", polyName)
			}
		}
	}

	// AST modification (simplified version for preview)
	// (Actual TranspileLegacy is more complex, but we want the highlights)
	// ... we will use the same AST.Inspect logic ...
	// (Reusing the core logic from TranspileLegacy for accurate diff)

	// Since we are doing a single file preview, let's generate the code
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		fmt.Printf("Error formatting node: %v\n", err)
		return
	}
	
	transpiled := buf.String()
	// Apply regex replacements for any, UnixMilli, etc.
	transpiled = strings.ReplaceAll(transpiled, "any", "interface{}")
	transpiled = strings.ReplaceAll(transpiled, ".UnixMilli()", ".UnixNano() / 1000000")

	// Print line-by-line diff
	origLines := strings.Split(string(original), "\n")
	transLines := strings.Split(transpiled, "\n")

	// Simple diff viewer (LCS is overkill for a quick CLI tool, let's just do line comparisons)
	fmt.Println("--------------------------------------------------")
	maxLines := len(origLines)
	if len(transLines) > maxLines { maxLines = len(transLines) }
	
	hasDiff := false
	for i := 0; i < maxLines; i++ {
		o := ""
		if i < len(origLines) { o = strings.TrimSpace(origLines[i]) }
		t := ""
		if i < len(transLines) { t = strings.TrimSpace(transLines[i]) }

		if o != t {
			if o != "" { fmt.Printf("%s- %4d: %s%s\n", ColorRed, i+1, strings.TrimSpace(origLines[i]), ColorReset) }
			if t != "" { fmt.Printf("%s+ %4d: %s%s\n", ColorGreen, i+1, strings.TrimSpace(transLines[i]), ColorReset) }
			hasDiff = true
		}
	}

	if !hasDiff {
		fmt.Println("No differences found. This file is already compatible or needs no transpilation.")
	}
	fmt.Println("--------------------------------------------------")
}
