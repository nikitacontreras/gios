package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ANSI colors
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
)

type Finding struct {
	File   string
	Line   int
	Factor string
	Code   string
}

type RiskFactor struct {
	Name        string
	Risk        string
	Description string
	Occurrences int
	Findings    []Finding
}

// safeRepeat returns s repeated n times, or empty if n < 0
func safeRepeat(s string, n int) string {
	if n < 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

// printRow prints a table row with fixed widths, handling colors manually to ensure alignment
func printRow(factor, risk, found, icon, status string) {
	// Column Widths (matching the header exactly)
	wFactor := 29
	wRisk := 10
	wFound := 9
	wStatus := 8

	fName := factor
	if len(fName) > wFactor {
		fName = fName[:wFactor]
	}
	fStr := fName + safeRepeat(" ", wFactor-len(fName))

	rColor := ColorReset
	switch risk {
	case "Critical", "High":
		rColor = ColorRed
	case "Medium":
		rColor = ColorYellow
	case "Low":
		rColor = ColorCyan
	case "None":
		rColor = ColorGreen
	}
	rStr := rColor + risk + ColorReset + safeRepeat(" ", wRisk-len(risk))

	foStr := found + safeRepeat(" ", wFound-len(found))

	sColor := ColorReset
	switch status {
	case "DANGER":
		sColor = ColorRed
	case "WARNING":
		sColor = ColorYellow
	case "AUTOFIX":
		sColor = ColorCyan
	case "NATIVE", "OK":
		sColor = ColorGreen
	}
	stStr := sColor + status + ColorReset + safeRepeat(" ", wStatus-len(status))

	fmt.Printf("│ %s │ %s │ %s │ %s %s │\n", fStr, rStr, foStr, icon, stStr)
}

func analyzeProject() {
	// Check if gios project
	if _, err := os.Stat("gios.json"); os.IsNotExist(err) {
		fmt.Println(ColorRed + "Error: This directory is not a Gios project (gios.json not found)." + ColorReset)
		fmt.Println("Please run 'gios init' or navigate to a project directory.")
		return
	}

	conf := loadConfig()
	isLegacy := conf.Arch == "armv7"
	targetName := "Modern (arm64)"
	if isLegacy {
		targetName = "Legacy (armv7)"
	}

	verbose := false
	for _, arg := range os.Args {
		if arg == "--verbose" || arg == "-v" {
			verbose = true
			break
		}
	}

	cwd, _ := os.Getwd()
	fmt.Printf("\n%s[gios]%s Analyzing project for %s%s%s compatibility: %s%s%s\n", 
		ColorCyan, ColorReset, ColorBold, targetName, ColorReset, ColorBold, filepath.Base(cwd), ColorReset)

	riskFactors := map[string]*RiskFactor{
		"Generics":  {Name: "Go Generics (1.18+)", Risk: "High", Occurrences: 0},
		"Any":       {Name: "Any Keyword (1.18+)", Risk: "Low", Occurrences: 0},
		"IO-Ref":    {Name: "Modern IO/OS (1.16+)", Risk: "Medium", Occurrences: 0},
		"UnixMilli": {Name: "Time UnixMilli (1.17+)", Risk: "High", Occurrences: 0},
		"Unsafe":    {Name: "Unsafe Slice/String (1.17+)", Risk: "Critical", Occurrences: 0},
		"ModernLib": {Name: "Modern Std Libraries (1.21+)", Risk: "Critical", Occurrences: 0},
	}

	if !isLegacy {
		for k := range riskFactors {
			riskFactors[k].Risk = "None"
		}
	}

	// Regexes
	reGenerics := regexp.MustCompile(`\[[a-zA-Z0-9_* ,.\[\]]+\]`)
	reAny := regexp.MustCompile(`\bany\b`)
	reIORef := regexp.MustCompile(`\b(io|os)\.(ReadAll|Discard|NopCloser|ReadFile|WriteFile|ReadDir)\b`)
	reUnixMilli := regexp.MustCompile(`\bUnixMilli\b`)
	reUnsafe := regexp.MustCompile(`unsafe\.(Slice|StringData|String)\b`)
	reModernLib := regexp.MustCompile(`"(slices|maps|cmp|net/netip)"`)

	totalFiles := 0
	filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".go") || strings.Contains(path, ".gios") {
			return nil
		}
		totalFiles++
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		row := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			row++
			line := scanner.Text()
			
			check := func(re *regexp.Regexp, factorKey string) {
				if re.MatchString(line) && !strings.Contains(line, "//") {
					// Extra check for generics vs array [64]byte
					if factorKey == "Generics" {
						if strings.Contains(line, "[") && regexp.MustCompile(`\[[0-9]+\]`).MatchString(line) {
							return
						}
					}
					riskFactors[factorKey].Occurrences++
					if verbose {
						rel, _ := filepath.Rel(cwd, path)
						riskFactors[factorKey].Findings = append(riskFactors[factorKey].Findings, Finding{
							File:   rel,
							Line:   row,
							Factor: factorKey,
							Code:   strings.TrimSpace(line),
						})
					}
				}
			}

			check(reGenerics, "Generics")
			check(reAny, "Any")
			check(reIORef, "IO-Ref")
			check(reUnixMilli, "UnixMilli")
			check(reUnsafe, "Unsafe")
			check(reModernLib, "ModernLib")
		}
		return nil
	})

	fmt.Printf("%s[gios]%s Scanned %d Go files.\n\n", ColorCyan, ColorReset, totalFiles)
	
	// Borders
	wFactor := 29
	wRisk := 10
	wFound := 9
	wStatus := 12

	fmt.Println("┌" + strings.Repeat("─", wFactor+2) + "┬" + strings.Repeat("─", wRisk+2) + "┬" + strings.Repeat("─", wFound+2) + "┬" + strings.Repeat("─", wStatus+2) + "┐")
	fmt.Printf("│ %-29s │ %-10s │ %-9s │ %-12s │\n", "COMPATIBILITY FACTOR", "RISK", "FOUND", "STATUS")
	fmt.Println("├" + strings.Repeat("─", wFactor+2) + "┼" + strings.Repeat("─", wRisk+2) + "┼" + strings.Repeat("─", wFound+2) + "┼" + strings.Repeat("─", wStatus+2) + "┤")

	problems := 0
	keys := []string{"Generics", "Any", "IO-Ref", "UnixMilli", "Unsafe", "ModernLib"}
	for _, k := range keys {
		f := riskFactors[k]
		icon := "✅"
		status := "OK"
		if f.Occurrences > 0 && isLegacy {
			if f.Risk == "Critical" || f.Risk == "High" {
				icon = "❌"
				status = "DANGER"
				problems++
			} else if f.Risk == "Medium" {
				icon = "⚠️"
				status = "WARNING"
			} else {
				icon = "🪄"
				status = "AUTOFIX"
			}
		} else if f.Occurrences > 0 && !isLegacy {
			icon = "🚀"
			status = "NATIVE"
		}
		printRow(f.Name, f.Risk, fmt.Sprintf("%d", f.Occurrences), icon, status)
	}
	fmt.Println("└" + strings.Repeat("─", wFactor+2) + "┴" + strings.Repeat("─", wRisk+2) + "┴" + strings.Repeat("─", wFound+2) + "┴" + strings.Repeat("─", wStatus+2) + "┘")

	if verbose {
		fmt.Println("\n" + ColorBold + "--- DETAILED FINDINGS ---" + ColorReset)
		for _, k := range keys {
			f := riskFactors[k]
			for _, find := range f.Findings {
				fmt.Printf("%s%s:%d%s [%s] -> %s\n", ColorCyan, find.File, find.Line, ColorReset, f.Name, ColorYellow+find.Code+ColorReset)
			}
		}
	}

	if isLegacy && problems > 0 {
		fmt.Printf("\n%s[!] Conclusion:%s Project has significant legacy compatibility risks.\n", ColorRed, ColorReset)
		if !verbose {
			fmt.Println("    Use 'gios analyze --verbose' to see exactly where the issues are.")
		}
		fmt.Printf("    Use %s'gios run --unsafe'%s to attempt automated backporting.\n", ColorBold, ColorReset)
	} else if !isLegacy {
		fmt.Printf("\n%s[+] Conclusion:%s Modern target detected. All features natively supported!\n", ColorGreen, ColorReset)
	} else {
		fmt.Printf("\n%s[+] Conclusion:%s Project looks compatible with legacy iOS!\n", ColorGreen, ColorReset)
	}
}
