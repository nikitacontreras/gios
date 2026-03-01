package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	AppVersion = "v1.2.0"
	BuildTime  = "unknown"
)

type Config struct {
	Name         string `json:"name"`
	PackageID    string `json:"package_id"`
	Version      string `json:"version"`
	GoVersion    string `json:"go_version"`
	SDKVersion   string `json:"sdk_version"`
	Arch         string `json:"arch"`
	Output       string `json:"output"`
	Main         string `json:"main"`
	Entitlements string `json:"entitlements"`
	Daemon       bool   `json:"daemon"`
	Deploy       struct {
		IP   string `json:"ip"`
		Path string `json:"path"`
	} `json:"deploy"`
}

var giosDir string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error: Could not determine user home directory.")
		os.Exit(1)
	}
	giosDir = filepath.Join(home, ".gios")
}

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("GIOS CLI version %s (Built: %s)\n", AppVersion, BuildTime)
		return
	}

	if len(os.Args) < 2 {
		fmt.Println("GIOS - The modern build system for legacy & modern iOS")
		fmt.Println("======================================================")
		fmt.Printf("Version: %s (Built: %s)\n\n", AppVersion, BuildTime)
		fmt.Println("Usage: gios <command>")
		fmt.Println("\nAvailable commands:")
		fmt.Println("  init       - Initializes a new project (Interactive Setup)")
		fmt.Println("  build      - Builds and signs the binary using gios.json configuration")
		fmt.Println("               (Use '--unsafe' to transpile vendor dependencies)")
		fmt.Println("  run        - Builds, signs and sends the binary to the device via SCP")
		fmt.Println("               (Use 'run --watch' to execute and stream output)")
		fmt.Println("               (Use '--unsafe' to transpile vendor dependencies)")
		fmt.Println("  package    - Prepares the binary in a .deb file (Cydia)")
		fmt.Println("  install    - Packages and automatically installs the .deb on the iDevice (DPKG)")
		fmt.Println("  connect    - Opens a persistent connection to the device for faster deploys")
		fmt.Println("  disconnect - Closes the active persistent connection")
		fmt.Println("  update     - Updates Gios CLI from GitHub to the latest release")
		fmt.Println("  sdk        - Manages iOS SDKs from Theos (list, add, remove)")
		fmt.Println("  analyze    - Scans dependencies for legacy compatibility (risk factors)")
		fmt.Println("  doctor     - Diagnoses local environment for build readiness")
		fmt.Println("  diff       - Shows what changes gios would apply to a specific file")
		fmt.Println("  logs       - Streams syslog from the device for debugging")
		fmt.Println("  watch      - Automatically builds and runs on file changes (alias for run --watch)")
		fmt.Println("\nExample: gios init")
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "build":
		build()
	case "run":
		build()
		run()
	case "package", "pkg":
		build()
		createDeb()
	case "install":
		build()
		createDeb()
		installDeb()
	case "connect":
		connect()
	case "disconnect":
		disconnect()
	case "update":
		updateGios()
	case "sdk":
		handleSDK()
	case "logs":
		runLogs()
	case "watch":
		runWatch()
	case "analyze":
		analyzeProject()
	case "doctor":
		runDoctor()
	case "diff":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gios diff <file>")
			return
		}
		runDiff(os.Args[2])
	case "init":
		initProject()
	default:
		fmt.Printf("Error: Unknown command '%s'.\n", cmd)
		fmt.Println("Run 'gios' with no arguments to see the list of available commands.")
	}
}

func loadConfig() Config {
	data, err := ioutil.ReadFile("gios.json")
	if err != nil {
		fmt.Println("Error: gios.json not found. Run 'gios init' first.")
		os.Exit(1)
	}
	var conf Config
	json.Unmarshal(data, &conf)

	// Defaults logic
	if conf.Output == "" {
		if conf.Name != "" {
			conf.Output = conf.Name
		} else {
			conf.Output = "out_bin"
		}
	}

	// Override with --out flag if present
	if outFlag := getFlagValue("--out"); outFlag != "" {
		// If it's a path, use the base for local file and full for deploy
		if strings.Contains(outFlag, "/") {
			conf.Output = filepath.Base(outFlag)
			conf.Deploy.Path = outFlag
		} else {
			conf.Output = outFlag
			// For simple name, we update the filename part of the deploy path if it exists
			if conf.Deploy.Path != "" {
				dir := filepath.Dir(conf.Deploy.Path)
				conf.Deploy.Path = filepath.Join(dir, outFlag)
			}
		}
	}

	return conf
}

func getFlagValue(flagName string) string {
	for i, arg := range os.Args {
		if arg == flagName && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return ""
}

func ensureWrapper(sdkPath string) string {
	err := os.MkdirAll(giosDir, 0755)
	if err != nil {
		fmt.Printf("Error creating .gios directory: %v\n", err)
		os.Exit(1)
	}

	wrapperPath := filepath.Join(giosDir, "arm-clang.sh")

	content := `#!/bin/bash
if [ -z "$GIOS_SDK_PATH" ]; then
    echo "GIOS_SDK_PATH is not configured."
    exit 1
fi
exec clang -target armv7-apple-ios5.0 -marm -march=armv7-a -mfpu=vfpv3-d16 \
     -isysroot "$GIOS_SDK_PATH" \
     -Wno-unused-command-line-argument \
     -Wno-incompatible-sysroot \
     -Wno-error=incompatible-sysroot \
     -fno-asynchronous-unwind-tables \
     "$@"
`
	err = ioutil.WriteFile(wrapperPath, []byte(content), 0755)
	if err != nil {
		fmt.Printf("Error writing wrapper: %v\n", err)
		os.Exit(1)
	}
	    return wrapperPath
}

func ensureShims(sdkPath, wrapperPath string) string {
	libDir := filepath.Join(giosDir, "lib")
	os.MkdirAll(libDir, 0755)
	

	shimC := filepath.Join(libDir, "shims.c")
	shimO := filepath.Join(libDir, "shims.o")
	shimA := filepath.Join(libDir, "libgios_libc.a")

	content := `#include <stddef.h>
#include <dirent.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/stat.h>
#include <errno.h>

void *memmove(void *dest, const void *src, size_t n) {
    unsigned char *d = (unsigned char *)dest;
    const unsigned char *s = (const unsigned char *)src;
    if (d < s) {
        while (n--) *d++ = *s++;
    } else {
        d += n; s += n;
        while (n--) *--d = *--s;
    }
    return dest;
}

void *memcpy(void *dest, const void *src, size_t n) {
    return memmove(dest, src, n);
}

void *memset(void *s, int c, size_t n) {
    unsigned char *p = (unsigned char *)s;
    while (n--) *p++ = (unsigned char)c;
    return s;
}

int memcmp(const void *s1, const void *s2, size_t n) {
    const unsigned char *p1 = (const unsigned char *)s1;
    const unsigned char *p2 = (const unsigned char *)s2;
    while (n--) {
        if (*p1 != *p2) return (int)(*p1 - *p2);
        p1++; p2++;
    }
    return 0;
}

void memset_pattern16(void *b, const void *pattern16, size_t len) {
    unsigned char *dest = (unsigned char *)b;
    const unsigned char *pat = (const unsigned char *)pattern16;
    while (len >= 16) {
        for(int i=0; i<16; i++) dest[i] = pat[i];
        dest += 16; len -= 16;
    }
    for(size_t i=0; i<len; i++) dest[i] = pat[i];
}

void *__memcpy_chk(void *dest, const void *src, size_t len, size_t destlen) { return memcpy(dest, src, len); }
void *__memset_chk(void *dest, int c, size_t len, size_t destlen) { return memset(dest, c, len); }
void *__memmove_chk(void *dest, const void *src, size_t len, size_t destlen) { return memmove(dest, src, len); }

// Backport for iOS 5.1.1 (missing fdopendir)
DIR *fdopendir(int fd) {
    char path[1024];
    if (fcntl(fd, F_GETPATH, path) != -1) {
        return opendir(path);
    }
    errno = ENOSYS;
    return NULL;
}
`
	ioutil.WriteFile(shimC, []byte(content), 0644)
	
	// Compile shims (force ARM mode to match Go runtime expectations)
	cmd := exec.Command(wrapperPath, "-Os", "-c", shimC, "-o", shimO)
	cmd.Env = append(os.Environ(), "GIOS_SDK_PATH="+sdkPath)
	cmd.Run()
	
	// Create archive
	exec.Command("ar", "rcs", shimA, shimO).Run()
	
	return libDir
}

func build() {
	conf := loadConfig()
	cwd, _ := os.Getwd()

	unsafeFlag := false
	for _, arg := range os.Args {
		if arg == "--unsafe" {
			unsafeFlag = true
		}
	}

	fmt.Printf("[gios] Project: %s\n", conf.Name)
	fmt.Printf("[gios] Arch: %s, SDK: %s\n", conf.Arch, conf.SDKVersion)

	home, _ := os.UserHomeDir()
	var goBin string

	// Toolchain routing
	if conf.GoVersion == "local" || conf.GoVersion == "system" {
		goBin, _ = exec.LookPath("go")
		if goBin == "" {
			goBin = "go"
		}
	} else {
		goBin = filepath.Join(home, ".gvm", "gos", conf.GoVersion, "bin", "go")
		if _, err := os.Stat(goBin); os.IsNotExist(err) {
			fmt.Printf("[gios] Error: Go not found at %s. Did you use gvm to install it?\n", goBin)
			os.Exit(1)
		}
	}

	var envOS, envArch, envArm, cc, sdkPath string

	if conf.Arch == "armv7" {
		// Legacy configuration
		envOS = "darwin"
		envArch = "arm"
		envArm = "7"
		sdkPath = filepath.Join(giosDir, "sdks", "iPhoneOS"+conf.SDKVersion+".sdk")
		if _, err := os.Stat(sdkPath); os.IsNotExist(err) {
			fmt.Printf("[gios] SDK not found at %s. Attempting to download...\n", sdkPath)
			if err := ensureSDK(conf.SDKVersion, sdkPath); err != nil {
				fmt.Printf("[!] Error installing SDK: %v\n", err)
				os.Exit(1)
			}
		}
		cc = ensureWrapper(sdkPath)
		
		// Gios Legacy Code Transpiler (Modern -> 1.14)
		fmt.Println("[gios] Legacy 32-bit Target Detected.")

		if unsafeFlag {
			fmt.Println("[gios] [Transpiler] WARNING: --unsafe flag active. Transpiling 'vendor' third-party dependencies.")
		}

		if err := TranspileLegacy(cwd, unsafeFlag); err != nil {
			fmt.Println("[!] Transpiler Error:", err)
		}
		
	} else if conf.Arch == "arm64" {
		// Modern configuration (Rootless / 64-bit)
		envOS = "ios"
		envArch = "arm64"
		envArm = ""
		// We can rely on native Xcode SDK if installed, or just let CGO do its magic
		cc = ""
		sdkPath = ""
	}

	cgoState := "1"
	var giosLibDir string
	if conf.Arch == "armv7" {
		giosLibDir = ensureShims(sdkPath, cc)
	}

	var ldflags string
	if conf.Arch == "armv7" {
		ldflags = fmt.Sprintf("-s -w -extld=%s \"-extldflags=-L%s -lgios_libc\"", cc, giosLibDir)
	} else {
		ldflags = "-s -w"
	}

	cmdEnv := append(os.Environ(),
		"CGO_ENABLED="+cgoState,
		"GOOS="+envOS,
		"GOARCH="+envArch,
	)
	if conf.Arch == "armv7" {
		cmdEnv = append(cmdEnv, "CGO_LDFLAGS=-L"+giosLibDir+" -lgios_libc")
	}
	if envArm != "" {
		cmdEnv = append(cmdEnv, "GOARM="+envArm)
	}
	if cc != "" {
		cmdEnv = append(cmdEnv, "CC="+cc)
	}
	if sdkPath != "" {
		cmdEnv = append(cmdEnv, "GIOS_SDK_PATH="+sdkPath)
	}

	isDylib := strings.HasSuffix(conf.Output, ".dylib")
	
	if isDylib && conf.Arch == "armv7" {
		// armv7 doesn't support c-shared in old go versions easily.
		// Use c-archive + manual link hack.
		tempA := "lib_gios_tmp.a"
		tempH := "lib_gios_tmp.h"
		buildArgs := []string{"build", "-trimpath", "-buildmode=c-archive", "-o", tempA, conf.Main}
		if _, err := os.Stat(filepath.Join(cwd, "vendor")); err == nil {
			buildArgs = append(buildArgs, "-mod=vendor")
		}
		
		cmd := exec.Command(goBin, buildArgs...)
		cmd.Dir = cwd
		cmd.Env = cmdEnv
		drawProgress("Compiling Archive", 30)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("\nGo Archive Error:\n%s\n", string(out))
			os.Exit(1)
		}
		
		drawProgress("Linking Dynamic Lib", 60)
		// Manual link with CC (which has SDK, arch, etc)
		// Using -Wl,-all_load to force inclusion of the Go runtime from the .a file
		linkArgs := []string{"-shared", "-dynamiclib", "-o", conf.Output, "-Wl,-all_load", tempA, "-L" + giosLibDir, "-lgios_libc", "-framework", "CoreFoundation", "-framework", "UIKit"}
		linkCmd := exec.Command(cc, linkArgs...)
		linkCmd.Env = cmdEnv
		if out, err := linkCmd.CombinedOutput(); err != nil {
			fmt.Printf("\nLinker Error:\n%s\n", string(out))
			os.Exit(1)
		}
		os.Remove(tempA)
		os.Remove(tempH)
		drawProgress("Linking", 80)

	} else {
		buildArgs := []string{"build", "-trimpath", "-ldflags=" + ldflags}
		if isDylib {
			buildArgs = append(buildArgs, "-buildmode=c-shared")
		}
		if _, err := os.Stat(filepath.Join(cwd, "vendor")); err == nil {
			buildArgs = append(buildArgs, "-mod=vendor")
		}
		buildArgs = append(buildArgs, "-o", conf.Output, conf.Main)

		cmd := exec.Command(goBin, buildArgs...)
		cmd.Dir = cwd
		cmd.Env = cmdEnv

		drawProgress("Compiling", 30)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("\nCompilation error:\n%s\n", string(out))
			os.Exit(1)
		}
		drawProgress("Compiling", 70)
	}

	// Sign
	if _, err := exec.LookPath("ldid"); err == nil {
		drawProgress("Signing", 85)
		var signCmd *exec.Cmd
		if conf.Entitlements != "" && conf.Entitlements != "none" {
			signCmd = exec.Command("ldid", "-S"+conf.Entitlements, conf.Output)
		} else {
			signCmd = exec.Command("ldid", "-S", conf.Output)
		}
		signCmd.CombinedOutput()
	}
	drawProgress("Ready!", 100)
}

func run() {
	conf := loadConfig()
	isDylib := strings.HasSuffix(conf.Output, ".dylib")
	if conf.Deploy.IP == "" || conf.Deploy.IP == "192.168.1.XX" {
		fmt.Println("[gios] Error: Deployment IP is not configured.")
		return
	}

	watch := false
	for _, arg := range os.Args {
		if arg == "--watch" {
			watch = true
			break
		}
	}

	fmt.Printf("[gios] Sending to %s...\n", conf.Deploy.IP)
	dest := fmt.Sprintf("root@%s:%s", conf.Deploy.IP, conf.Deploy.Path)

	sshKey := filepath.Join(giosDir, "id_rsa")

	cmd := exec.Command("scp",
		"-i", sshKey,
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=~/.ssh/gios-%r@%h:%p",
		conf.Output, dest)

	drawProgress("Uploading", 50)
	if err := cmd.Run(); err != nil {
		fmt.Println("\n[gios] Error uploading file via SCP.")
		return
	}

	if isDylib {
		// Also upload the .plist filter
		plistName := strings.TrimSuffix(conf.Output, ".dylib") + ".plist"
		localPlist := "filter.plist"
		if _, err := os.Stat(localPlist); os.IsNotExist(err) {
			// Create a proper XML plist
			plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Filter</key>
	<dict>
		<key>Bundles</key>
		<array>
			<string>com.apple.springboard</string>
		</array>
	</dict>
</dict>
</plist>`
			ioutil.WriteFile("tmp_filter.plist", []byte(plistContent), 0644)
			localPlist = "tmp_filter.plist"
		}
		
		destPlist := filepath.Join(conf.Deploy.Path, plistName)
		scpPlist := exec.Command("scp", "-i", sshKey, "-o", "ControlPath=~/.ssh/gios-%r@%h:%p", localPlist, "root@"+conf.Deploy.IP+":"+destPlist)
		scpPlist.Run()
		os.Remove("tmp_filter.plist")
	}
	drawProgress("Uploading", 100)

	if isDylib {
		fmt.Printf("[gios] Tweak detected. Triggering Respring to apply...\n")
		respringCmd := exec.Command("ssh",
			"-i", sshKey,
			"-o", "ControlPath=~/.ssh/gios-%r@%h:%p",
			"root@"+conf.Deploy.IP,
			"killall -9 SpringBoard")
		respringCmd.Run()
		fmt.Println("[gios] Respring triggered! Wait 2 seconds for injection.")
		time.Sleep(2 * time.Second)
		return
	}

	if watch {
		runPath := conf.Deploy.Path
		outputBase := filepath.Base(conf.Output)
		deployBase := filepath.Base(runPath)

		// Only join if it's clearly a directory path or doesn't already end with the output name
		if (strings.HasSuffix(runPath, "/") || !strings.Contains(deployBase, ".")) && deployBase != outputBase {
			runPath = filepath.Join(runPath, outputBase)
		}

		fmt.Printf("[gios] Executing %s on device...\n", runPath)
		fmt.Println("--------------------------------------------------")

		sshCmd := exec.Command("ssh",
			"-i", sshKey,
			"-o", "ControlMaster=auto",
			"-o", "ControlPath=~/.ssh/gios-%r@%h:%p",
			"-t", "root@"+conf.Deploy.IP,
			fmt.Sprintf("chmod +x %s && env GOGC=20 GOMAXPROCS=1 GODEBUG=asyncpreemptoff=1 %s", runPath, runPath))

		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr
		sshCmd.Run()

		fmt.Println("\n--------------------------------------------------")
	} else {
		fmt.Println("[gios] OK!")
	}
}

func createDeb() {
	conf := loadConfig()
	pkgID := conf.PackageID
	if pkgID == "" {
		pkgID = "com.gios." + conf.Name
	}
	version := conf.Version
	if version == "" {
		version = "1.0.0"
	}

	drawProgress("Packaging", 10)

	stage := "deb_stage"
	os.RemoveAll(stage)

	// Check if modern or legacy
	var binPath string
	var debArch string
	if conf.Arch == "arm64" {
		// Rootless behavior
		binPath = filepath.Join(stage, "var", "jb", "usr", "bin")
		debArch = "iphoneos-arm64"
	} else {
		// Legacy behavior
		binPath = filepath.Join(stage, "usr", "bin")
		debArch = "iphoneos-arm"
	}

	debianDir := filepath.Join(stage, "DEBIAN")
	os.MkdirAll(binPath, 0755)
	os.MkdirAll(debianDir, 0755)

	// Move binary
	if strings.HasSuffix(conf.Output, ".dylib") {
		// Handle dylib for tweaks
		dylibPath := filepath.Join(stage, "Library", "MobileSubstrate", "DynamicLibraries")
		if conf.Arch == "arm64" {
			dylibPath = filepath.Join(stage, "var", "jb", "Library", "MobileSubstrate", "DynamicLibraries")
		}
		os.MkdirAll(dylibPath, 0755)
		exec.Command("cp", conf.Output, filepath.Join(dylibPath, conf.Output)).Run()
		os.Chmod(filepath.Join(dylibPath, conf.Output), 0755)
		// Create a .plist for the dylib
		plistName := strings.TrimSuffix(conf.Output, ".dylib") + ".plist"
		plistPath := filepath.Join(dylibPath, plistName)
		
		filterSrc := "filter.plist"
		if _, err := os.Stat(filterSrc); err == nil {
			exec.Command("cp", filterSrc, plistPath).Run()
			fmt.Println("[gios] (+) Used custom filter.plist from project root")
		} else {
			plistContent := fmt.Sprintf(`{ Filter = { Bundles = ( "com.apple.springboard" ); }; }`) // Default filter
			ioutil.WriteFile(plistPath, []byte(plistContent), 0644)
			fmt.Println("[gios] (+) Used default SpringBoard filter")
		}
		fmt.Println("[gios] (+) Packaged as a Tweak (.dylib)")
	} else {
		// Handle regular executable
		exec.Command("cp", conf.Output, filepath.Join(binPath, conf.Output)).Run()
		os.Chmod(filepath.Join(binPath, conf.Output), 0755)
	}

	// Control file
	control := fmt.Sprintf("Package: %s\nName: %s\nVersion: %s\nArchitecture: %s\nDescription: App created with Gios\nMaintainer: Gios User\nAuthor: Gios\nSection: Utilities\n",
		pkgID, conf.Name, version, debArch)
	ioutil.WriteFile(filepath.Join(debianDir, "control"), []byte(control), 0644)

	// LaunchDaemon
	if conf.Daemon {
		var daemonDir string
		if conf.Arch == "arm64" {
			daemonDir = filepath.Join(stage, "var", "jb", "Library", "LaunchDaemons")
		} else {
			daemonDir = filepath.Join(stage, "Library", "LaunchDaemons")
		}

		os.MkdirAll(daemonDir, 0755)

		plistName := fmt.Sprintf("%s.plist", pkgID)
		plistPath := filepath.Join(daemonDir, plistName)

		plistOutputTarget := fmt.Sprintf("/usr/bin/%s", conf.Output)
		if conf.Arch == "arm64" {
			plistOutputTarget = fmt.Sprintf("/var/jb/usr/bin/%s", conf.Output)
		}

		plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/var/log/%s.log</string>
	<key>StandardErrorPath</key>
	<string>/var/log/%s.log</string>
</dict>
</plist>`, pkgID, plistOutputTarget, pkgID, pkgID)

		ioutil.WriteFile(plistPath, []byte(plistContent), 0644)

		var daemonLoadPath = fmt.Sprintf("/Library/LaunchDaemons/%s", plistName)
		if conf.Arch == "arm64" {
			daemonLoadPath = fmt.Sprintf("/var/jb/Library/LaunchDaemons/%s", plistName)
		}

		// Post-Install script
		postinstPath := filepath.Join(debianDir, "postinst")
		postinstContent := fmt.Sprintf(`#!/bin/bash
chown 0:0 %s
launchctl load %s
exit 0
`, daemonLoadPath, daemonLoadPath)
		ioutil.WriteFile(postinstPath, []byte(postinstContent), 0755)

		// Pre-Remove script
		prermPath := filepath.Join(debianDir, "prerm")
		prermContent := fmt.Sprintf(`#!/bin/bash
launchctl unload %s
exit 0
`, daemonLoadPath)
		ioutil.WriteFile(prermPath, []byte(prermContent), 0755)

		fmt.Println("[gios] (+) Configured as a LaunchDaemon (Background execution)")
	}

	// Build deb
	drawProgress("Building .deb", 80)
	debName := fmt.Sprintf("%s_%s_%s.deb", pkgID, version, debArch)
	out, err := exec.Command("dpkg-deb", "-Zgzip", "-b", stage, debName).CombinedOutput()
	if err != nil {
		fmt.Printf("\n[!] Error in dpkg-deb:\n%s\n", string(out))
		fmt.Println("    Note: You need to install dpkg on your Mac (brew install dpkg)")
	} else {
		drawProgress("Done!", 100)
		fmt.Printf("[+] Generated: %s\n", debName)
	}

	os.RemoveAll(stage)
}

func installDeb() {
	conf := loadConfig()
	pkgID := conf.PackageID
	if pkgID == "" {
		pkgID = "com.gios." + conf.Name
	}
	version := conf.Version
	if version == "" {
		version = "1.0.0"
	}

	debArch := "iphoneos-arm"
	if conf.Arch == "arm64" {
		debArch = "iphoneos-arm64"
	}

	debName := fmt.Sprintf("%s_%s_%s.deb", pkgID, version, debArch)

	if conf.Deploy.IP == "" || conf.Deploy.IP == "192.168.1.XX" {
		fmt.Println("[gios] Error: Deployment IP not configured to install the .deb")
		return
	}

	fmt.Printf("[gios] Installing on %s...\n", conf.Deploy.IP)
	dest := fmt.Sprintf("root@%s:/tmp/%s", conf.Deploy.IP, debName)

	sshKey := filepath.Join(giosDir, "id_rsa")

	err := exec.Command("scp",
		"-i", sshKey,
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=~/.ssh/gios-%r@%h:%p",
		debName, dest).Run()
	if err != nil {
		fmt.Println("[!] Error uploading the .deb to the device (SCP)")
		return
	}

	fmt.Println("[gios] Running DPKG on the iDevice...")

	dpkgCmd := "dpkg"
	if conf.Arch == "arm64" {
		dpkgCmd = "dpkg"
	}

	sshInstall := fmt.Sprintf("%s -i /tmp/%s && rm -f /tmp/%s", dpkgCmd, debName, debName)
	out, err := exec.Command("ssh",
		"-i", sshKey,
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=~/.ssh/gios-%r@%h:%p",
		"root@"+conf.Deploy.IP, sshInstall).CombinedOutput()
	if err != nil {
		fmt.Printf("[!] Installation failed (DPKG):\n%s\n", string(out))
		return
	}

	fmt.Printf("[+] Installation successfully completed on the iPad!\n%s\n", string(out))

	// Tweak check
	if strings.HasSuffix(conf.Output, ".dylib") {
		fmt.Println("[gios] Tweak detected. Triggering Respring to apply changes...")
		exec.Command("ssh", "-i", filepath.Join(giosDir, "id_rsa"), "-o", "ControlPath=~/.ssh/gios-%r@%h:%p", "root@"+conf.Deploy.IP, "killall -9 SpringBoard").Run()
	}
}

func prompt(label, def string) string {
	fmt.Printf("%s [%s]: ", label, def)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	text := scanner.Text()
	if text == "" {
		return def
	}
	return text
}

func drawProgress(step string, percent int) {
	width := 30
	pos := (percent * width) / 100
	bar := "[" + strings.Repeat("█", pos) + strings.Repeat("░", width-pos) + "]"
	fmt.Printf("\r%s %s %d%%  ", ColorCyan+step+ColorReset, bar, percent)
	if percent >= 100 {
		fmt.Println()
	}
}

func connect() {
	var ip string
	if len(os.Args) >= 3 {
		ip = os.Args[2]
	} else {
		conf := loadConfig()
		if conf.Deploy.IP == "" || conf.Deploy.IP == "192.168.1.XX" {
			fmt.Println("Error: IP not provided and not found in gios.json")
			return
		}
		ip = conf.Deploy.IP
	}

	fmt.Printf("[gios] Setting up background SSH connection to %s...\n", ip)

	sshKeyPath := filepath.Join(giosDir, "id_rsa")
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		fmt.Println("[gios] Generating new SSH key for passwordless auth...")
		exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-f", sshKeyPath, "-N", "").Run()
	}

	fmt.Println("[gios] Checking existing SSH authorization...")
	testCmd := exec.Command("ssh", "-i", sshKeyPath, "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "root@"+ip, "echo auth_ok")
	if err := testCmd.Run(); err != nil {
		fmt.Println("[gios] Passwordless login not configured or failed.")
		fmt.Println("[gios] Sending SSH key to device (it may ask for your root password one last time)...")
		cmdCopy := exec.Command("ssh-copy-id", "-i", sshKeyPath, "root@"+ip)
		cmdCopy.Stdin = os.Stdin
		cmdCopy.Stdout = os.Stdout
		cmdCopy.Stderr = os.Stderr
		cmdCopy.Run()
	} else {
		fmt.Println("[gios] Identity confirmed. Key already installed.")
	}

	infoCmd := exec.Command("ssh", "-i", sshKeyPath, "-o", "BatchMode=yes", "root@"+ip, `uname -n; uname -m; sw_vers -productVersion 2>/dev/null || echo 'Unknown'`)
	infoOut, _ := infoCmd.Output()
	lines := strings.Split(strings.TrimSpace(string(infoOut)), "\n")
	if len(lines) >= 3 {
		name := strings.TrimSpace(lines[0])
		arch := strings.TrimSpace(lines[1])
		ver := strings.TrimSpace(lines[2])
		fmt.Printf("[gios] Device Info: %s | %s | iOS %s\n", name, arch, ver)
		saveDeviceData(ip, name, arch, ver)
	}

	os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".ssh"), 0700)

	// Close any previous dangling socket to avoid "socket already exists" error
	exec.Command("ssh", "-O", "exit", "-o", "ControlPath=~/.ssh/gios-%r@%h:%p", "root@"+ip).Run()

	fmt.Println("[gios] Opening persistent background tunnel...")
	cmd := exec.Command("ssh",
		"-f", "-N", "-M",
		"-o", "ControlPath=~/.ssh/gios-%r@%h:%p",
		"-o", "ServerAliveInterval=60",
		"-i", sshKeyPath,
		"root@"+ip)

	if err := cmd.Run(); err != nil {
		fmt.Println("[!] Failed to establish background connection.")
	} else {
		fmt.Println("[+] Magic connection open in background! 'gios install' and 'gios run --watch' will fly.")
	}
}

func ensureSDK(version, targetPath string) error {
	fileName := fmt.Sprintf("iPhoneOS%s.sdk.tar.xz", version)
	url := fmt.Sprintf("https://github.com/theos/sdks/releases/download/master-146e41f/%s", fileName)
	return ensureSDKFromURL(version, targetPath, url)
}

func disconnect() {
	var ip string
	if len(os.Args) >= 3 {
		ip = os.Args[2]
	} else {
		conf := loadConfig()
		if conf.Deploy.IP == "" || conf.Deploy.IP == "192.168.1.XX" {
			fmt.Println("Error: IP not provided and not found in gios.json")
			return
		}
		ip = conf.Deploy.IP
	}

	fmt.Printf("[gios] Closing connection to %s...\n", ip)

	cmd := exec.Command("ssh", "-O", "exit", "-o", "ControlPath=~/.ssh/gios-%r@%h:%p", "root@"+ip)
	err := cmd.Run()
	if err != nil {
		fmt.Println("[gios] Connection was already closed or not found.")
		home, _ := os.UserHomeDir()
		os.Remove(filepath.Join(home, ".ssh", fmt.Sprintf("gios-root@%s:22", ip)))
	} else {
		fmt.Println("[gios] Connection closed successfully.")
	}
}

func saveDeviceData(ip, name, arch, version string) {
	type DeviceInfo struct {
		IP      string `json:"ip"`
		Name    string `json:"name"`
		Arch    string `json:"arch"`
		Version string `json:"version"`
	}

	devicesFile := filepath.Join(giosDir, "devices.json")
	var devices map[string]DeviceInfo
	data, err := ioutil.ReadFile(devicesFile)
	if err == nil {
		json.Unmarshal(data, &devices)
	}
	if devices == nil {
		devices = make(map[string]DeviceInfo)
	}
	devices[ip] = DeviceInfo{
		IP:      ip,
		Name:    name,
		Arch:    arch,
		Version: version,
	}
	out, _ := json.MarshalIndent(devices, "", "  ")
	ioutil.WriteFile(devicesFile, out, 0644)
}

func updateGios() {
	fmt.Println("[gios] Checking for updates...")

	// API de GitHub para listar todos los releases (incluyendo pre-releases como "latest")
	req, err := http.NewRequest("GET", "https://api.github.com/repos/nikitacontreras/gios/releases", nil)
	if err != nil {
		fmt.Printf("[!] Could not check for updates: %v\n", err)
		return
	}
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil || resp.StatusCode == http.StatusNotFound {
		fmt.Println("[gios] No pending updates found.")
		return
	} else if resp.StatusCode != http.StatusOK {
		fmt.Printf("[!] Error fetching releases. (HTTP %d)\n", resp.StatusCode)
		return
	}
	defer resp.Body.Close()

	var releases []struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &releases)

	if len(releases) == 0 {
		fmt.Println("[gios] No pending updates found.")
		return
	}

	release := releases[0] // El primero es el más reciente

	if release.TagName == AppVersion {
		fmt.Printf("[gios] You already have the latest version: %s\n", AppVersion)
		return
	}

	fmt.Printf("[gios] Newer version found: %s. Current is: %s\n", release.TagName, AppVersion)

	osName := runtime.GOOS
	archName := runtime.GOARCH

	expectedFileName := fmt.Sprintf("gios-%s-%s", osName, archName)
	if osName == "windows" {
		expectedFileName += ".exe"
	}

	downloadUrl := ""
	for _, asset := range release.Assets {
		if asset.Name == expectedFileName {
			downloadUrl = asset.BrowserDownloadURL
			break
		}
	}

	if downloadUrl == "" {
		fmt.Println("[!] No binary available for your architecture:", expectedFileName)
		return
	}

	fmt.Printf("[gios] Downloading binary for %s/%s...\n", osName, archName)

	res, err := http.Get(downloadUrl)
	if err != nil {
		fmt.Println("[!] Download failed:", err)
		return
	}
	defer res.Body.Close()

	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("[!] Could not determine executable path.")
		return
	}

	tmpPath := exePath + ".tmp"
	out, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		fmt.Println("[!] Error creating tmp file:", err)
		return
	}

	_, err = io.Copy(out, res.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		fmt.Println("[!] Error copying data:", err)
		return
	}

	// Rename the tmp to current executable
	err = os.Rename(tmpPath, exePath)
	if err != nil {
		os.Remove(tmpPath)
		fmt.Println("[!] Error overwriting old binary. You might need to run this command with 'sudo'. Error:", err)
		return
	}

	fmt.Printf("[+] Success! Gios updated to %s\n", release.TagName)
}

func handleSDK() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: gios sdk <list|add|remove>")
		return
	}
	subcmd := os.Args[2]
	switch subcmd {
	case "list":
		listSDKs()
	case "add":
		addSDK()
	case "remove":
		removeSDK()
	default:
		fmt.Printf("Unknown sdk command: %s\n", subcmd)
	}
}

func getDownloadedSDKs() []string {
	var downloaded []string
	sdksDir := filepath.Join(giosDir, "sdks")
	files, err := ioutil.ReadDir(sdksDir)
	if err != nil {
		return downloaded
	}
	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), "iPhoneOS") {
			downloaded = append(downloaded, f.Name())
		}
	}
	return downloaded
}

func listSDKs() {
	fmt.Println("[gios] Fetching available SDKs from Theos...")
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/theos/sdks/releases/latest", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("[!] Failed to fetch SDKs list.")
		return
	}
	defer resp.Body.Close()

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &release)

	downloaded := getDownloadedSDKs()
	downloadedMap := make(map[string]bool)
	for _, d := range downloaded {
		downloadedMap[d] = true
	}

	fmt.Println("\nAvailable iOS SDKs:")
	fmt.Println("--------------------------------------------------")
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".sdk.tar.xz") || strings.HasSuffix(asset.Name, ".sdk.tar.gz") {
			sdkName := strings.TrimSuffix(strings.TrimSuffix(asset.Name, ".tar.xz"), ".tar.gz")
			status := " "
			if downloadedMap[sdkName] {
				status = "*"
			}
			fmt.Printf(" [%s] %s\n", status, sdkName)
		}
	}
	fmt.Println("--------------------------------------------------")
	fmt.Println("(*) = Already downloaded")
}

func addSDK() {
	fmt.Println("[gios] Fetching available SDKs...")
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/theos/sdks/releases/latest", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("[!] Failed to fetch SDKs list.")
		return
	}
	defer resp.Body.Close()

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &release)

	var validAssets []struct{ Name, URL string }
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".sdk.tar.xz") || strings.HasSuffix(asset.Name, ".sdk.tar.gz") {
			validAssets = append(validAssets, struct{ Name, URL string }{asset.Name, asset.BrowserDownloadURL})
		}
	}

	fmt.Println("\nSelect an SDK to download:")
	for i, asset := range validAssets {
		sdkName := strings.TrimSuffix(strings.TrimSuffix(asset.Name, ".tar.xz"), ".tar.gz")
		fmt.Printf("  [%d] %s\n", i+1, sdkName)
	}

	choiceStr := prompt("\nEnter number", "")
	var choice int
	fmt.Sscanf(choiceStr, "%d", &choice)

	if choice < 1 || choice > len(validAssets) {
		fmt.Println("[!] Invalid selection.")
		return
	}

	selected := validAssets[choice-1]
	sdkName := strings.TrimSuffix(strings.TrimSuffix(selected.Name, ".tar.xz"), ".tar.gz")
	targetPath := filepath.Join(giosDir, "sdks", sdkName)

	if _, err := os.Stat(targetPath); err == nil {
		fmt.Printf("[gios] SDK %s is already installed.\n", sdkName)
		return
	}

	version := strings.TrimPrefix(sdkName, "iPhoneOS")
	version = strings.TrimSuffix(version, ".sdk")
	ensureSDKFromURL(version, targetPath, selected.URL)
}

func removeSDK() {
	downloaded := getDownloadedSDKs()
	if len(downloaded) == 0 {
		fmt.Println("[gios] No SDKs currently installed.")
		return
	}

	fmt.Println("\nInstalled iOS SDKs:")
	for i, sdk := range downloaded {
		fmt.Printf("  [%d] %s\n", i+1, sdk)
	}

	choiceStr := prompt("\nEnter number to remove", "")
	var choice int
	fmt.Sscanf(choiceStr, "%d", &choice)

	if choice < 1 || choice > len(downloaded) {
		fmt.Println("[!] Invalid selection.")
		return
	}

	selected := downloaded[choice-1]
	fmt.Printf("[gios] Removing %s...\n", selected)
	os.RemoveAll(filepath.Join(giosDir, "sdks", selected))
	fmt.Println("[+] Removed successfully.")
}

func ensureSDKFromURL(version, targetPath, customUrl string) error {
	sdkDir := filepath.Dir(targetPath)
	os.MkdirAll(sdkDir, 0755)

	fmt.Printf("[gios] Downloading iOS %s SDK...\n", version)

	fileName := filepath.Base(customUrl)
	tarPath := filepath.Join(sdkDir, fileName)

	cmd := exec.Command("curl", "-L", "-#", "-o", tarPath, customUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("curl failed: %v", err)
	}

	fmt.Println("[gios] Extracting SDK (this might take a few seconds)...")
	cmdExt := exec.Command("tar", "-xf", tarPath, "-C", sdkDir)
	if err := cmdExt.Run(); err != nil {
		return fmt.Errorf("tar extraction failed: %v", err)
	}

	os.Remove(tarPath)
	fmt.Println("[+] SDK ready for cross-compilation!")
	return nil
}

func initProject() {
	cwd, _ := os.Getwd()
	baseName := filepath.Base(cwd)

	fmt.Println("\n" + ColorCyan + "=============================================")
	fmt.Println("    GIOS Interactive Project Initialization  ")
	fmt.Println("=============================================" + ColorReset)

	// Questions
	conf := Config{}
	conf.Name = prompt("1. Project Name", baseName)
	conf.PackageID = prompt("2. Package ID (e.g. com.yourname.app)", "com.gios."+conf.Name)
	conf.Version = prompt("3. Version", "1.0.0")

	fmt.Println("\n4. Project Template:")
	fmt.Println("   [1] CLI Tool       - Simple standalone binary")
	fmt.Println("   [2] LaunchDaemon   - Background service (auto .plist)")
	fmt.Println("   [3] Cydia Tweak    - Injected library (WIP - Phase 3)")
	templateType := prompt("   Option", "1")

	fmt.Println("\n5. Target Device Architecture:")
	fmt.Println("   [1] Legacy 32-bit (iPhone 2G to 5c, iPad 1 to 4)")
	fmt.Println("   [2] Modern 64-bit Rootless (iPhone 5s to 16, iPad Air/Pro)")
	targetOption := prompt("   Option", "1")

	isModern := (targetOption == "2")
	if !isModern {
		conf.Arch = "armv7"
		conf.GoVersion = "go1.14.15"
		conf.SDKVersion = "9.3"
		conf.Entitlements = "ents.plist"
	} else {
		conf.Arch = "arm64"
		conf.GoVersion = "system"
		conf.SDKVersion = "system"
		conf.Entitlements = "none"
	}

	conf.Deploy.IP = prompt("\n6. Target Device IP Address", "192.168.1.XX")
	conf.Output = "out_" + conf.Name
	conf.Main = "main.go"

	if templateType == "2" {
		conf.Daemon = true
	}

	if isModern {
		conf.Deploy.Path = "/var/jb/var/root"
	} else {
		conf.Deploy.Path = "/var/root/" + conf.Output
	}

	fmt.Println("\n" + ColorCyan + "=============================================" + ColorReset)

	// Create go.mod
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("[+] Initializing Go modules...")
		modGoVer := "1.14"
		if isModern {
			modGoVer = "1.23"
		}
		modData := fmt.Sprintf("module %s\n\ngo %s\n", conf.Name, modGoVer)
		ioutil.WriteFile("go.mod", []byte(modData), 0644)
	}

	// Create JSON
	jsonData, _ := json.MarshalIndent(conf, "", "  ")
	ioutil.WriteFile("gios.json", jsonData, 0644)
	fmt.Println("[+] Created gios.json")

	// Entitlements
	if !isModern && conf.Entitlements != "none" {
		if _, err := os.Stat("ents.plist"); os.IsNotExist(err) {
			entsData := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>get-task-allow</key>
	<true/>
</dict>
</plist>
`
			ioutil.WriteFile("ents.plist", []byte(entsData), 0644)
			fmt.Println("[+] Created ents.plist")
		}
	}

	// Project Type Specific Files
	switch templateType {
	case "2": // Daemon
		if _, err := os.Stat(conf.Name + ".plist"); os.IsNotExist(err) {
			plistData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s/%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`, conf.PackageID, conf.Deploy.Path, conf.Output)
			ioutil.WriteFile(conf.Name+".plist", []byte(plistData), 0644)
			fmt.Println("[+] Created LaunchDaemon plist")
		}
		if _, err := os.Stat("main.go"); os.IsNotExist(err) {
			mainGoData := `package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Printf("Daemon %s starting...\n", "` + conf.Name + `")
	for {
		// Your background logic here
		fmt.Println("Heartbeat from Gios Daemon")
		time.Sleep(30 * time.Second)
	}
}
`
			ioutil.WriteFile("main.go", []byte(mainGoData), 0644)
			fmt.Println("[+] Created Daemon main.go")
		}

	case "3": // Tweak
		if _, err := os.Stat("filter.plist"); os.IsNotExist(err) {
			filterData := `{ Filter = { Bundles = ( "com.apple.springboard" ); }; }`
			ioutil.WriteFile("filter.plist", []byte(filterData), 0644)
			fmt.Println("[+] Created Cydia Tweak filter.plist")
		}
		if _, err := os.Stat("main.go"); os.IsNotExist(err) {
			mainGoData := `package main

import "fmt"

// Tweak Entry point (WIP)
// In Go, we'll export a function that Cydia's substrate can load
func init() {
	fmt.Println("Gios Tweak Injected successfully!")
}

func main() {}
`
			ioutil.WriteFile("main.go", []byte(mainGoData), 0644)
			fmt.Println("[+] Created Tweak main.go")
		}

	default: // CLI
		if _, err := os.Stat("main.go"); os.IsNotExist(err) {
			mainGoData := `package main

import "fmt"

func main() {
	fmt.Println("Hello from Gios CLI!")
}
`
			ioutil.WriteFile("main.go", []byte(mainGoData), 0644)
			fmt.Println("[+] Created CLI main.go")
		}
	}

	fmt.Printf("\n%s[+] Project initialized successfully!%s\n", ColorGreen, ColorBold)
	fmt.Println("Run 'gios build' to compile or 'gios run' to test on device.")
}
