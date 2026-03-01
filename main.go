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

const RemoteAssetsURL = "https://raw.githubusercontent.com/nikitacontreras/gios-platform-assets/main/assets.json"

type AssetManifest struct {
	SDKs []struct {
		Name     string `json:"name"`
		Platform string `json:"platform"`
		URL      string `json:"url"`
		Hash     string `json:"hash,omitempty"`
	} `json:"sdks"`
	DDIs []struct {
		Version  string `json:"version"`
		Platform string `json:"platform"`
		URL      string `json:"url"`
		SigURL   string `json:"sig_url"`
		Hash     string `json:"hash,omitempty"`
	} `json:"ddis"`
}

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
		USB  bool   `json:"usb"`
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

	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "?" {
		printHelp()
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
	case "headers":
		runHeaders()
	case "hook":
		runHook()
	case "polyfill":
		runPolyfill()
	case "sdk":
		handleSDK()
	case "logs":
		runLogs()
	case "watch":
		runWatch()
	case "analyze":
		analyzeProject()
	case "daemon":
		runDaemon()
	case "shell":
		runShell()
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
	case "info":
		runInfo()
	case "screenshot":
		runScreenshot()
	case "reboot":
		runReboot()
	case "mount":
		runMount()
	default:
		fmt.Printf("%s[!] Error:%s Unknown command '%s'.\n", ColorRed, ColorReset, cmd)
		fmt.Println("Run 'gios help' to see the list of available commands.")
	}
}

func printHelp() {
	fmt.Println(ColorCyan + ColorBold + `
 dP""b8 88  dP"Yb  .dP"Y8 
dP   ` + "`" + `" 88 dP   Yb ` + "`" + `Ybo." 
Yb  "88 88 Yb   dP o.` + "`" + `Y8b 
 YboodP 88  YbodP  8bodP' 
` + ColorReset)
	
	fmt.Printf("%s[GIOS CLI]%s %sThe modern build system for legacy & modern iOS%s\n", ColorCyan, ColorReset, ColorBold, ColorReset)
	fmt.Printf("Version: %s | Built: %s\n", AppVersion, BuildTime)
	fmt.Println("----------------------------------------------------------------")
	
	fmt.Println("\n" + ColorBold + "USAGE:" + ColorReset)
	fmt.Println("  gios <command> [arguments] [flags...]")

	fmt.Println("\n" + ColorBold + "CORE COMMANDS:" + ColorReset)
	fmt.Printf("  %-12s %s\n", "init", "Initializes a new project (Interactive Setup)")
	fmt.Printf("  %-12s %s\n", "build", "Builds and signs the binary for the configured target")
	fmt.Printf("  %-12s %s\n", "run", "Builds, signs and executes the app on the iDevice")
	fmt.Printf("  %-12s %s\n", "package", "Creates a .deb package for Cydia/Sileo distribution")
	fmt.Printf("  %-12s %s\n", "install", "Builds, packages and installs the .deb on the device")
	fmt.Printf("  %-12s %s\n", "watch", "Shortcut for 'run --watch' (Auto-build & deploy)")

	fmt.Println("\n" + ColorBold + "CONNECTIVITY:" + ColorReset)
	fmt.Printf("  %-12s %s\n", "connect", "Opens a persistent SSH tunnel to the device")
	fmt.Printf("  %-12s %s\n", "daemon", "Starts the background Go SSH service (Native mode)")
	fmt.Printf("  %-12s %s\n", "shell", "Opens an interactive SSH terminal on the device")
	fmt.Printf("  %-12s %s\n", "logs", "Streams live syslog for real-time debugging")
	fmt.Printf("  %-12s %s\n", "disconnect", "Closes all active background connections")

	fmt.Println("\n" + ColorBold + "UTILITIES:" + ColorReset)
	fmt.Printf("  %-12s %s\n", "doctor", "Diagnoses your environment for build readiness")
	fmt.Printf("  %-12s %s\n", "analyze", "Analyzes code for legacy iOS compatibility risks")
	fmt.Printf("  %-12s %s\n", "sdk", "Manage iOS SDKs (list, add, remove)")
	fmt.Printf("  %-12s %s\n", "diff", "Shows transpilation changes for a specific file")
	fmt.Printf("  %-12s %s\n", "update", "Updates Gios CLI to the latest GitHub release")

	fmt.Println("\n" + ColorBold + "USB TOOLS (No SSH required):" + ColorReset)
	fmt.Printf("  %-12s %s\n", "info", "Display detailed hardware/battery info via USB")
	fmt.Printf("  %-12s %s\n", "screenshot", "Capture a screenshot of the iDevice (saved to Mac)")
	fmt.Printf("  %-12s %s\n", "reboot", "Force a reboot of the connected device")
	fmt.Printf("  %-12s %s\n", "mount", "Mount Developer Disk Image (needed for screenshot/debug)")

	fmt.Println("\n" + ColorBold + "ADVANCED (Elite Pro Tools):" + ColorReset)
	fmt.Printf("  %-12s %s\n", "headers", "Auto-extract ObjC headers from any process")
	fmt.Printf("  %-12s %s\n", "hook", "Generate DSL-based hooks for Go tweaks")
	fmt.Printf("  %-12s %s\n", "polyfill", "Intelligent compatibility patching for legacy iOS")

	fmt.Println("\n" + ColorBold + "GLOBAL FLAGS:" + ColorReset)
	fmt.Printf("  %-15s %s\n", "--watch, -w", "Enable Auto-build & run on file change (Pro Mode)")
	fmt.Printf("  %-15s %s\n", "--unsafe", "Force-transpile vendor/ dependencies for legacy targets")
	fmt.Printf("  %-15s %s\n", "--out <path>", "Overwrite output binary filename")
	fmt.Printf("  %-15s %s\n", "--ip <ip>", "Temporarily override target device IP")
	fmt.Printf("  %-15s %s\n", "--syslog", "Use native USB streaming for logs (needs idevicesyslog)")
	fmt.Printf("  %-15s %s\n", "--v, --version", "Show current Gios version")
	
	fmt.Println("\n" + ColorBold + "EXAMPLES:")
	fmt.Println("  gios init")
	fmt.Println("  gios run --watch")
	fmt.Println("  gios headers SpringBoard")
	fmt.Println("  gios hook SpringBoard init")
	fmt.Println("")
}

func loadConfig() Config {
	conf, err := loadConfigSafe()
	if err != nil {
		fmt.Println("Error: gios.json not found. Run 'gios init' first.")
		os.Exit(1)
	}
	return conf
}

func loadConfigSafe() (Config, error) {
	// Search for gios.json in current and parent directories
	curr, _ := os.Getwd()
	var data []byte
	var err error
	
	for {
		data, err = os.ReadFile(filepath.Join(curr, "gios.json"))
		if err == nil {
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			return Config{}, fmt.Errorf("gios.json not found in any parent directory")
		}
		curr = parent
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

	return conf, nil
}

func getFlagValue(flagName string) string {
	for i, arg := range os.Args {
		if arg == flagName && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return ""
}

func (c Config) GetSSHArgs(extra ...string) []string {
	sshKeyPath := filepath.Join(giosDir, "id_rsa")
	target := "root@" + c.Deploy.IP
	
	home, _ := os.UserHomeDir()
	controlPath := filepath.Join(home, ".ssh", "gios-%r@%h:%p")

	args := []string{
		"-i", sshKeyPath,
		"-o", "HostKeyAlgorithms=+ssh-rsa",
		"-o", "PubkeyAcceptedAlgorithms=+ssh-rsa",
		"-o", "KexAlgorithms=+diffie-hellman-group1-sha1,diffie-hellman-group14-sha1,diffie-hellman-group-exchange-sha1",
		"-o", "Ciphers=+aes128-cbc,3des-cbc",
		"-o", "ControlPath=" + controlPath,
	}
	if c.Deploy.USB {
		target = "root@127.0.0.1"
		args = append(args, "-p", "2222")
	}
	args = append(args, target)
	return append(args, extra...)
}

func (c Config) GetSCPArgs() []string {
	sshKeyPath := filepath.Join(giosDir, "id_rsa")
	home, _ := os.UserHomeDir()
	controlPath := filepath.Join(home, ".ssh", "gios-%r@%h:%p")

	args := []string{
		"-i", sshKeyPath,
		"-o", "HostKeyAlgorithms=+ssh-rsa",
		"-o", "PubkeyAcceptedAlgorithms=+ssh-rsa",
		"-o", "KexAlgorithms=+diffie-hellman-group1-sha1,diffie-hellman-group14-sha1,diffie-hellman-group-exchange-sha1",
		"-o", "Ciphers=+aes128-cbc,3des-cbc",
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=" + controlPath,
	}
	if c.Deploy.USB {
		args = append(args, "-P", "2222")
	}
	return args
}

func (c Config) GetSCPTarget(filePath string) string {
	target := "root@" + c.Deploy.IP
	if c.Deploy.USB {
		target = "root@127.0.0.1"
	}
	return target + ":" + filePath
}

func ensureUSBTunnel(conf Config) bool {
	if !conf.Deploy.USB {
		return true
	}

	// Check for idevice_id
	if _, err := exec.LookPath("idevice_id"); err != nil {
		fmt.Printf("%s[!] Error: 'idevice_id' not found. Please install libimobiledevice%s\n", ColorRed, ColorReset)
		return false
	}

	// Check for iproxy
	if _, err := exec.LookPath("iproxy"); err != nil {
		fmt.Printf("%s[!] Error: 'iproxy' not found. Please install libimobiledevice.%s\n", ColorRed, ColorReset)
		return false
	}

	// Check if any device is connected
	out, _ := exec.Command("idevice_id", "-l").Output()
	if len(strings.TrimSpace(string(out))) == 0 {
		fmt.Printf("%s[!] Warning: No iOS device detected via USB. Ensure it's plugged in.%s\n", ColorYellow, ColorReset)
		return false
	}

	// Check if iproxy is already listening on 2222
	checkCmd := exec.Command("nc", "-z", "127.0.0.1", "2222")
	if err := checkCmd.Run(); err != nil {
		fmt.Printf("%s[gios] (USB Mode) Initializing usbmuxd tunnel (2222 -> 22)...%s\n", ColorCyan, ColorReset)
		go func() {
			exec.Command("iproxy", "2222", "22").Run()
		}()
		time.Sleep(1500 * time.Millisecond)
	}
	return true
}

func (c Config) remoteExec(cmd string) (string, error) {
	resp, err := callDaemon(DaemonRequest{Command: "exec", Payload: cmd})
	if err == nil {
		if resp.Error != "" {
			return resp.Output, fmt.Errorf(resp.Error)
		}
		return resp.Output, nil
	}

	// Fallback to legacy SSH command
	sshArgs := c.GetSSHArgs(cmd)
	out, err := exec.Command("ssh", sshArgs...).CombinedOutput()
	return string(out), err
}

func (c Config) remoteUpload(local, remote string) error {
	resp, err := callDaemon(DaemonRequest{Command: "upload", Payload: local, Remote: remote})
	if err == nil {
		if resp.Error != "" {
			return fmt.Errorf(resp.Error)
		}
		return nil
	}

	// Fallback to legacy SCP command
	target := c.GetSCPTarget(remote)
	scpArgs := c.GetSCPArgs()
	scpArgs = append(scpArgs, local, target)
	return exec.Command("scp", scpArgs...).Run()
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
	if !conf.Deploy.USB && (conf.Deploy.IP == "" || conf.Deploy.IP == "192.168.1.XX") {
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

	targetDisp := conf.Deploy.IP
	if conf.Deploy.USB {
		targetDisp = "USB Device"
		if !ensureUSBTunnel(conf) {
			return
		}
	}
	fmt.Printf("[gios] Sending to %s...\n", targetDisp)

	drawProgress("Uploading", 50)
	if err := conf.remoteUpload(conf.Output, conf.Deploy.Path); err != nil {
		fmt.Printf("\n[gios] Error uploading file: %v\n", err)
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
		conf.remoteUpload(localPlist, destPlist)
		os.Remove("tmp_filter.plist")
	}
	drawProgress("Uploading", 100)

	if isDylib {
		fmt.Printf("[gios] Tweak detected. Triggering Respring to apply...\n")
		conf.remoteExec("killall -9 SpringBoard")
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

		sshFullHost := "root@" + conf.Deploy.IP
		sshPort := "22"
		if conf.Deploy.USB {
			sshFullHost = "root@127.0.0.1"
			sshPort = "2222"
		}

		home, _ := os.UserHomeDir()
		controlPath := filepath.Join(home, ".ssh", "gios-%r@%h:%p")

		sshArgs := []string{
			"-i", filepath.Join(giosDir, "id_rsa"),
			"-o", "HostKeyAlgorithms=+ssh-rsa",
			"-o", "PubkeyAcceptedAlgorithms=+ssh-rsa",
			"-o", "KexAlgorithms=+diffie-hellman-group1-sha1,diffie-hellman-group14-sha1,diffie-hellman-group-exchange-sha1",
			"-o", "Ciphers=+aes128-cbc,3des-cbc",
			"-o", "ControlMaster=auto",
			"-o", "ControlPath=" + controlPath,
			"-t", "-p", sshPort, sshFullHost,
			fmt.Sprintf("chmod +x %s && env GOGC=20 GOMAXPROCS=1 GODEBUG=asyncpreemptoff=1 %s", runPath, runPath),
		}

		sshCmd := exec.Command("ssh", sshArgs...)
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
		fmt.Println("    Note: You need to install dpkg (e.g., via your package manager)")
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

	if !conf.Deploy.USB && (conf.Deploy.IP == "" || conf.Deploy.IP == "192.168.1.XX") {
		fmt.Println("[gios] Error: Deployment IP not configured to install the .deb")
		return
	}

	targetDisp := conf.Deploy.IP
	if conf.Deploy.USB {
		targetDisp = "USB Device"
		if !ensureUSBTunnel(conf) {
			return
		}
	}
	fmt.Printf("[gios] Installing on %s...\n", targetDisp)

	if err := conf.remoteUpload(debName, "/tmp/"+debName); err != nil {
		fmt.Printf("[!] Error uploading the .deb: %v\n", err)
		return
	}

	fmt.Println("[gios] Running DPKG on the iDevice...")
	dpkgCmd := "dpkg"
	sshInstall := fmt.Sprintf("%s -i /tmp/%s && rm -f /tmp/%s", dpkgCmd, debName, debName)

	out, err := conf.remoteExec(sshInstall)
	if err != nil {
		fmt.Printf("[!] Installation failed (DPKG):\n%s\n", out)
		return
	}
	fmt.Printf("[+] Installation successfully completed on the iPad!\n%s\n", out)

	if strings.HasSuffix(conf.Output, ".dylib") {
		fmt.Println("[gios] Tweak detected. Triggering Respring to apply changes...")
		conf.remoteExec("killall -9 SpringBoard")
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
	conf := loadConfig()
	if len(os.Args) >= 3 {
		arg := strings.ToLower(os.Args[2])
		if arg == "usb" {
			conf.Deploy.USB = true
			conf.Deploy.IP = "127.0.0.1"
		} else {
			conf.Deploy.IP = os.Args[2]
		}
	}

	targetDisp := conf.Deploy.IP
	if conf.Deploy.USB {
		targetDisp = "USB Device"
		if !ensureUSBTunnel(conf) {
			return
		}
	}

	if !conf.Deploy.USB && (conf.Deploy.IP == "" || conf.Deploy.IP == "192.168.1.XX") {
		fmt.Println("Error: IP not provided and not found in gios.json")
		return
	}

	fmt.Printf("[gios] Setting up background SSH connection to %s...\n", targetDisp)

	sshKeyPath := filepath.Join(giosDir, "id_rsa")
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		fmt.Println("[gios] Generating new SSH key for passwordless auth...")
		exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-f", sshKeyPath, "-N", "").Run()
	}

	fmt.Println("[gios] Checking existing SSH authorization (Native Go Client)...")
	client, err := NewSSHClient(conf)
	if err != nil {
		fmt.Println("[gios] Key-based login failed or not configured. Trying password auth...")
		fmt.Printf("[gios] Please enter the root password for %s (default: alpine): ", targetDisp)
		bytePassword := prompt("Password", "alpine")
		
		client, err = NewSSHClientWithPassword(conf, bytePassword)
		if err != nil {
			fmt.Printf("[!] Could not connect with password either: %v\n", err)
			return
		}

		fmt.Println("[gios] Installing SSH key for future passwordless logins...")
		if err := client.InstallKey(sshKeyPath + ".pub"); err != nil {
			fmt.Printf("[!] Failed to install public key: %v\n", err)
			// Non-fatal, we can continue for this session
		} else {
			fmt.Println("[+] SSH key installed successfully.")
		}
	} else {
		fmt.Println("[gios] Identity confirmed via Native Go Client.")
	}
	defer client.Close()

	infoOut, err := client.Run(`uname -n; uname -m; sw_vers -productVersion 2>/dev/null || echo 'Unknown'`)
	if err == nil {
		lines := strings.Split(strings.TrimSpace(infoOut), "\n")
		if len(lines) >= 3 {
			name := strings.TrimSpace(lines[0])
			arch := strings.TrimSpace(lines[1])
			ver := strings.TrimSpace(lines[2])
			fmt.Printf("[gios] Device Info: %s | %s | iOS %s\n", name, arch, ver)
			saveDeviceData(conf.Deploy.IP, name, arch, ver)
		}
	}

	os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".ssh"), 0700)

	// Still start the OpenSSH Master for backward compatibility in scripts
	sshExitArgs := conf.GetSSHArgs()
	sshExitArgs = append([]string{"-O", "exit"}, sshExitArgs...)
	exec.Command("ssh", sshExitArgs...).Run()

	fmt.Println("[gios] Opening persistent background tunnel (OpenSSH Compatibility)...")
	sshMasterArgs := []string{"-f", "-N", "-M", "-o", "ServerAliveInterval=60"}
	sshMasterArgs = append(sshMasterArgs, conf.GetSSHArgs()...)
	
	cmdMaster := exec.Command("ssh", sshMasterArgs...)
	masterErrOut, err := cmdMaster.CombinedOutput()
	if err != nil {
		fmt.Printf("[!] Warning: Could not establish OpenSSH background connection: %v\n", err)
		if len(masterErrOut) > 0 {
			fmt.Printf("    SSH Error: %s\n", strings.TrimSpace(string(masterErrOut)))
		}
	} else {
		fmt.Println("[+] OpenSSH Magic connection open!")
	}

	if conf.Deploy.USB {
		fmt.Println("\n[gios] TIP: You can now run 'gios daemon usb' in another terminal for an even faster native Go experience.")
	} else {
		fmt.Println("\n[gios] TIP: You can now run 'gios daemon' in another terminal for an even faster native Go experience.")
	}
}

func ensureSDK(version, targetPath string) error {
	fileName := fmt.Sprintf("iPhoneOS%s.sdk.tar.xz", version)
	url := fmt.Sprintf("https://github.com/theos/sdks/releases/download/master-146e41f/%s", fileName)
	return ensureSDKFromURL(version, targetPath, url)
}

func disconnect() {
	conf := loadConfig()
	if len(os.Args) >= 3 {
		if os.Args[2] == "usb" {
			conf.Deploy.USB = true
		} else {
			conf.Deploy.IP = os.Args[2]
		}
	}

	targetDisp := conf.Deploy.IP
	if conf.Deploy.USB {
		targetDisp = "USB Device"
	}

	fmt.Printf("[gios] Closing connection to %s...\n", targetDisp)

	sshExitArgs := conf.GetSSHArgs()
	sshExitArgs = append([]string{"-O", "exit"}, sshExitArgs...)
	err := exec.Command("ssh", sshExitArgs...).Run()
	if err != nil {
		fmt.Println("[gios] Connection was already closed or not found.")
	} else {
		fmt.Println("[gios] Connection closed successfully.")
	}
}

func runShell() {
	conf := loadConfig()
	if len(os.Args) >= 3 && strings.ToLower(os.Args[2]) == "usb" {
		conf.Deploy.USB = true
		conf.Deploy.IP = "127.0.0.1"
	}

	targetDisp := conf.Deploy.IP
	if conf.Deploy.USB {
		targetDisp = "USB Device"
		ensureUSBTunnel(conf)
	}

	fmt.Printf("[gios] Opening interactive shell on %s...\n", targetDisp)

	sshArgs := conf.GetSSHArgs()
	// Add -t for TTY
	sshArgs = append([]string{"-t"}, sshArgs...)

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("[!] Shell session ended: %v\n", err)
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

func fetchAssetManifest() (*AssetManifest, error) {
	resp, err := http.Get(RemoteAssetsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var manifest AssetManifest
	body, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func listSDKs() {
	fmt.Println("[gios] Fetching available SDKs from Gios Platform Assets...")
	manifest, err := fetchAssetManifest()
	if err != nil {
		fmt.Printf("[!] Could not fetch manifest: %v. Using legacy fallback...\n", err)
		listSDKsLegacy()
		return
	}

	downloaded := getDownloadedSDKs()
	downloadedMap := make(map[string]bool)
	for _, d := range downloaded {
		downloadedMap[d] = true
	}

	fmt.Println("\nAvailable iOS SDKs:")
	fmt.Println("--------------------------------------------------")
	if len(manifest.SDKs) == 0 {
		fmt.Println(" [!] No SDKs found in the manifest.")
		fmt.Println(" [TIP] Make sure assets.json is populated in the platform-assets repo.")
	}
	for _, sdk := range manifest.SDKs {
		status := " "
		if downloadedMap[sdk.Name] {
			status = "*"
		}
		fmt.Printf(" [%s] %s (%s)\n", status, sdk.Name, sdk.Platform)
	}
	fmt.Println("--------------------------------------------------")
	fmt.Println("(*) = Already downloaded")
}

func listSDKsLegacy() {
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

	fmt.Println("\nAvailable iOS SDKs (Legacy):")
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
}

func addSDK() {
	fmt.Println("[gios] Fetching available SDKs...")
	manifest, err := fetchAssetManifest()
	var sdkList []struct {
		Name     string
		URL      string
		Platform string
		Hash     string
	}

	if err != nil {
		fmt.Printf("[!] Could not fetch manifest: %v. Using legacy fallback...\n", err)
		// Minimal fallback for addSDK
		req, _ := http.NewRequest("GET", "https://api.github.com/repos/theos/sdks/releases/latest", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			var release struct{ Assets []struct{ Name, BrowserDownloadURL string } `json:"assets"` }
			json.NewDecoder(resp.Body).Decode(&release)
			for _, asset := range release.Assets {
				if strings.HasSuffix(asset.Name, ".sdk.tar.xz") || strings.HasSuffix(asset.Name, ".sdk.tar.gz") {
					sdkList = append(sdkList, struct {
						Name     string
						URL      string
						Platform string
						Hash     string
					}{
						Name:     strings.TrimSuffix(strings.TrimSuffix(asset.Name, ".tar.xz"), ".tar.gz"),
						URL:      asset.BrowserDownloadURL,
						Platform: "iPhoneOS",
						Hash:     "",
					})
				}
			}
			resp.Body.Close()
		}
	} else {
		for _, sdk := range manifest.SDKs {
			sdkList = append(sdkList, struct {
				Name     string
				URL      string
				Platform string
				Hash     string
			}{sdk.Name, sdk.URL, sdk.Platform, sdk.Hash})
		}
	}

	if len(sdkList) == 0 {
		fmt.Println("[!] No SDKs available to download.")
		return
	}

	fmt.Println("\nSelect an SDK to download:")
	for i, sdk := range sdkList {
		fmt.Printf("  [%d] %s (%s)\n", i+1, sdk.Name, sdk.Platform)
	}

	choiceStr := prompt("\nEnter number", "")
	var choice int
	fmt.Sscanf(choiceStr, "%d", &choice)

	if choice < 1 || choice > len(sdkList) {
		fmt.Println("[!] Invalid selection.")
		return
	}

	selected := sdkList[choice-1]
	targetPath := filepath.Join(giosDir, "sdks", selected.Name)

	if _, err := os.Stat(targetPath); err == nil {
		fmt.Printf("[gios] SDK %s is already installed.\n", selected.Name)
		return
	}

	// For downloading:
	fmt.Printf("[gios] Downloading %s from %s...\n", selected.Name, selected.URL)
	ensureSDKFromURL(selected.Name, targetPath, selected.URL)

	// If hash is present, we should verify it (todo in ensureSDKFromURL)
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

func downloadURLToFile(url, targetPath string, showProgress bool) error {
	// Escape spaces in URL
	escapedURL := strings.ReplaceAll(url, " ", "%20")
	args := []string{"-L", "-o", targetPath}
	if showProgress {
		args = append(args, "-#")
	} else {
		args = append(args, "-s")
	}
	args = append(args, escapedURL)
	cmd := exec.Command("curl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ensureSDKFromURL(version, targetPath, customUrl string) error {
	sdkDir := filepath.Dir(targetPath)
	os.MkdirAll(sdkDir, 0755)

	fmt.Printf("[gios] Downloading iOS %s SDK...\n", version)

	fileName := filepath.Base(customUrl)
	tarPath := filepath.Join(sdkDir, fileName)

	if err := downloadURLToFile(customUrl, tarPath, true); err != nil {
		return fmt.Errorf("download failed: %v", err)
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

	fmt.Println("\n6. Deployment Method:")
	fmt.Println("   [1] Wi-Fi (IP Address)")
	fmt.Println("   [2] USB (libimobiledevice/iproxy)")
	deployOption := prompt("   Option", "1")

	if deployOption == "2" {
		conf.Deploy.USB = true
		conf.Deploy.IP = "127.0.0.1"
	} else {
		conf.Deploy.IP = prompt("   Target Device IP Address", "192.168.1.XX")
	}

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
