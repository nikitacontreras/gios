package diagnostic

import (
"github.com/nikitastrike/gios/pkg/config"
"github.com/nikitastrike/gios/pkg/utils"
)


import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func RunDoctor() {
	fmt.Printf("\n%s[gios]%s Running Gios Environment Diagnostic...\n", utils.ColorCyan, utils.ColorReset)
	fmt.Println("==========================================")

	errors := 0
	warnings := 0

	// 1. Go Version Check
	fmt.Print("Checking Go version... ")
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		fmt.Printf("%sFAILED%s (Is Go installed?)\n", utils.ColorRed, utils.ColorReset)
		errors++
	} else {
		ver := string(out)
		fmt.Printf("%sOK%s (%s)", utils.ColorGreen, utils.ColorReset, strings.TrimSpace(ver))
		if !strings.Contains(ver, "go1.2") {
			fmt.Printf(" %s[!] RECOMMENDATION: Use a modern Go (1.21+) for the transpiler to work best.%s", utils.ColorYellow, utils.ColorReset)
			warnings++
		}
		fmt.Println()
	}

	// 2. OS Check
	fmt.Printf("Checking OS... %s (%s/%s)\n", runtime.GOOS, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS != "darwin" {
		fmt.Printf("%s[!] WARNING: Gios is designed primarily for macOS. Some functions might be limited.%s\n", utils.ColorYellow, utils.ColorReset)
		warnings++
	}

	// 3. SDK Check (Check both Theos and Gios internal ones)
	fmt.Print("Checking iOS SDKs... ")
	home, _ := os.UserHomeDir()
	
	// Potential SDK paths
	searchPaths := []string{
		filepath.Join(config.GiosDir, "sdks"),
		filepath.Join(home, ".theos", "sdks"),
		"/opt/theos/sdks",
	}

	sdkCount := 0
	foundIn := ""
	for _, p := range searchPaths {
		entries, _ := os.ReadDir(p)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".sdk") {
				sdkCount++
				if foundIn == "" {
					foundIn = p
				}
			}
		}
	}

	if sdkCount == 0 {
		fmt.Printf("%sNONE FOUND%s\n", utils.ColorRed, utils.ColorReset)
		fmt.Println("    [!] Missing iOS SDKs for cross-compilation.")
		fmt.Printf("    [!] Use %s'gios sdk add'%s to download one automatically.\n", utils.ColorBold, utils.ColorReset)
		errors++
	} else {
		fmt.Printf("%sOK%s (%d SDKs found, primarily in %s)\n", utils.ColorGreen, utils.ColorReset, sdkCount, foundIn)
	}

	// 4. Code Signing (codesign & ldid)
	fmt.Print("Checking codesign... ")
	if _, err := exec.LookPath("codesign"); err != nil {
		fmt.Printf("%sFAILED%s\n", utils.ColorRed, utils.ColorReset)
		errors++
	} else {
		fmt.Printf("%sOK%s\n", utils.ColorGreen, utils.ColorReset)
	}

	fmt.Print("Checking ldid... ")
	if _, err := exec.LookPath("ldid"); err != nil {
		fmt.Printf("%sNOT FOUND%s (Only needed for local fakesigning)\n", utils.ColorYellow, utils.ColorReset)
		warnings++
	} else {
		fmt.Printf("%sOK%s\n", utils.ColorGreen, utils.ColorReset)
	}

	fmt.Print("Checking dpkg (packaging)... ")
	if _, err := exec.LookPath("dpkg-deb"); err != nil {
		fmt.Printf("%sNOT FOUND%s (Install 'dpkg' to use the package/install commands)\n", utils.ColorYellow, utils.ColorReset)
		warnings++
	} else {
		fmt.Printf("%sOK%s\n", utils.ColorGreen, utils.ColorReset)
	}

	// 5. libimobiledevice Check
	fmt.Print("Checking libimobiledevice (USB)... ")
	hasIdeviceId := false
	if _, err := exec.LookPath("idevice_id"); err == nil {
		hasIdeviceId = true
	}
	hasIproxy := false
	if _, err := exec.LookPath("iproxy"); err == nil {
		hasIproxy = true
	}

	if hasIdeviceId && hasIproxy {
		fmt.Printf("%sFOUND%s (USB development supported via iproxy)\n", utils.ColorGreen, utils.ColorReset)
	} else {
		fmt.Printf("%sLIMITED%s (Install libimobiledevice for fast USB deployment)\n", utils.ColorYellow, utils.ColorReset)
		warnings++
	}

	// 6. config.Config Check
	fmt.Print("Checking gios.json... ")
	if _, err := os.Stat("gios.json"); os.IsNotExist(err) {
		fmt.Printf("%sNOT FOUND%s (Run in a project folder)\n", utils.ColorYellow, utils.ColorReset)
		warnings++
	} else {
		fmt.Printf("%sOK%s\n", utils.ColorGreen, utils.ColorReset)
		// Try to ping device if IP set
		conf := config.LoadConfig()
		if conf.Deploy.IP != "" {
			fmt.Printf("Checking device reachability (ping %s)... ", conf.Deploy.IP)
			cmd := exec.Command("ping", "-c", "1", "-t", "2", conf.Deploy.IP)
			if err := cmd.Run(); err != nil {
				fmt.Printf("%sOFFLINE%s (Check USB/WiFi connection)\n", utils.ColorRed, utils.ColorReset)
				warnings++
			} else {
				fmt.Printf("%sONLINE%s\n", utils.ColorGreen, utils.ColorReset)
				
				// Also check SSH Auth
				fmt.Printf("Checking SSH Authentication (root@%s)... ", conf.Deploy.IP)
				sshKeyPath := filepath.Join(config.GiosDir, "id_rsa")
				sshCmd := exec.Command("ssh", "-i", sshKeyPath, "-o", "BatchMode=yes", "-o", "ConnectTimeout=2", "root@"+conf.Deploy.IP, "echo auth_ok")
				if err := sshCmd.Run(); err != nil {
					fmt.Printf("%sNOT AUTHORIZED%s (Run 'gios connect' first)\n", utils.ColorRed, utils.ColorReset)
					errors++
				} else {
					fmt.Printf("%sAUTHORIZED%s\n", utils.ColorGreen, utils.ColorReset)
				}
			}
		}
	}

	fmt.Println("==========================================")
	if errors == 0 && warnings == 0 {
		fmt.Printf("%s[+] Conclusion: Environment is PERFECT for development!%s\n", utils.ColorGreen, utils.ColorBold)
	} else if errors == 0 {
		fmt.Printf("%s[+] Conclusion: Environment is usable but has minor warnings.%s\n", utils.ColorYellow, utils.ColorBold)
	} else {
		fmt.Printf("%s[-] Conclusion: Found %d critical errors. Please fix them to build correctly.%s\n", utils.ColorRed, errors, utils.ColorBold)
	}
}
