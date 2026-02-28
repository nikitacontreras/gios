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
)

const AppVersion = "v1.2.0"

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
	if len(os.Args) < 2 {
		fmt.Println("Gios - The modern build system for legacy & modern iOS")
		fmt.Println("======================================================")
		fmt.Println("Usage: gios <command>")
		fmt.Println("\nAvailable commands:")
		fmt.Println("  init    - Initializes a new project (Interactive Setup)")
		fmt.Println("  build   - Builds and signs the binary using gios.json configuration")
		fmt.Println("  run     - Builds, signs and sends the binary to the device via SCP")
		fmt.Println("            (Use 'run --watch' to execute and stream output)")
		fmt.Println("  package - Prepares the binary in a .deb file (Cydia)")
		fmt.Println("  install - Packages and automatically installs the .deb on the iDevice (DPKG)")
		fmt.Println("  connect - Opens a persistent SSH connection to the device for faster deploys")
		fmt.Println("  disconnect- Closes the active SSH persistent connection")
		fmt.Println("  update  - Updates Gios CLI from GitHub to the latest release")
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
	case "package":
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
	return conf
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
exec clang -target armv7-apple-ios5.0 \
     -isysroot "$GIOS_SDK_PATH" \
     -Wno-unused-command-line-argument \
     -Wno-incompatible-sysroot \
     -Wno-error=incompatible-sysroot \
     "$@"
`
	err = ioutil.WriteFile(wrapperPath, []byte(content), 0755)
	if err != nil {
		fmt.Printf("Error writing wrapper: %v\n", err)
		os.Exit(1)
	}
	return wrapperPath
}

func build() {
	conf := loadConfig()
	cwd, _ := os.Getwd()
	
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
	} else if conf.Arch == "arm64" {
		// Modern configuration (Rootless / 64-bit)
		envOS = "ios"
		envArch = "arm64"
		envArm = ""
		// We can rely on native Xcode SDK if installed, or just let CGO do its magic
		cc = ""
		sdkPath = ""
	}

	cmd := exec.Command(goBin, "build", "-o", conf.Output, conf.Main)
	cmd.Dir = cwd
	cmdEnv := append(os.Environ(),
		"CGO_ENABLED=1",
		"GOOS="+envOS,
		"GOARCH="+envArch,
	)
	if envArm != "" {
		cmdEnv = append(cmdEnv, "GOARM="+envArm)
	}
	if cc != "" {
		cmdEnv = append(cmdEnv, "CC="+cc)
	}
	if sdkPath != "" {
		cmdEnv = append(cmdEnv, "GIOS_SDK_PATH="+sdkPath)
	}
	cmd.Env = cmdEnv

	fmt.Println("[gios] Compiling...")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Compilation error:\n%s\n", string(out))
		os.Exit(1)
	}

	fmt.Printf("[gios] Success -> %s\n", conf.Output)

	// Sign
	if _, err := exec.LookPath("ldid"); err == nil {
		fmt.Println("[gios] Signing...")
		var signCmd *exec.Cmd
		if conf.Entitlements != "" && conf.Entitlements != "none" {
			signCmd = exec.Command("ldid", "-S"+conf.Entitlements, conf.Output)
		} else {
			signCmd = exec.Command("ldid", "-S", conf.Output)
		}
		if out, err := signCmd.CombinedOutput(); err != nil {
			fmt.Printf("Signing error:\n%s\n", string(out))
		}
	} else {
		fmt.Println("[gios] Warning: ldid not found. The binary will not be signed.")
	}
}

func run() {
	conf := loadConfig()
	if conf.Deploy.IP == "" || conf.Deploy.IP == "192.168.1.XX" {
		fmt.Println("[gios] Error: Deployment IP is not configured.")
		return
	}

	watch := false
	if len(os.Args) >= 3 && os.Args[2] == "--watch" {
		watch = true
	}

	fmt.Printf("[gios] Sending to %s...\n", conf.Deploy.IP)
	dest := fmt.Sprintf("root@%s:%s", conf.Deploy.IP, conf.Deploy.Path)
	
	sshKey := filepath.Join(giosDir, "id_rsa")
	
	cmd := exec.Command("scp",
		"-i", sshKey,
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=~/.ssh/gios-%r@%h:%p",
		conf.Output, dest)
	
	if err := cmd.Run(); err != nil {
		fmt.Println("[gios] Error uploading file via SCP.")
		return
	}

	if watch {
		runPath := conf.Deploy.Path
		// Fix path if it's considered a directory
		if conf.Arch == "arm64" || strings.HasSuffix(runPath, "/") {
			runPath = filepath.Join(runPath, conf.Output)
		}

		fmt.Printf("[gios] Executing %s on device...\n", runPath)
		fmt.Println("--------------------------------------------------")
		
		sshCmd := exec.Command("ssh",
			"-i", sshKey,
			"-o", "ControlMaster=auto",
			"-o", "ControlPath=~/.ssh/gios-%r@%h:%p",
			"-t", "root@"+conf.Deploy.IP,
			fmt.Sprintf("chmod +x %s && %s", runPath, runPath))
			
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

	fmt.Printf("[gios] Packaging %s (%s) to .deb...\n", pkgID, version)

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
	exec.Command("cp", conf.Output, filepath.Join(binPath, conf.Output)).Run()
	os.Chmod(filepath.Join(binPath, conf.Output), 0755)

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
</dict>
</plist>`, pkgID, plistOutputTarget)

		ioutil.WriteFile(plistPath, []byte(plistContent), 0644)
		
		var daemonLoadPath = fmt.Sprintf("/Library/LaunchDaemons/%s", plistName)
		if conf.Arch == "arm64" {
			daemonLoadPath = fmt.Sprintf("/var/jb/Library/LaunchDaemons/%s", plistName)
		}

		// Post-Install script
		postinstPath := filepath.Join(debianDir, "postinst")
		postinstContent := fmt.Sprintf(`#!/bin/bash
launchctl load %s
exit 0
`, daemonLoadPath)
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
	debName := fmt.Sprintf("%s_%s_%s.deb", pkgID, version, debArch)
	out, err := exec.Command("dpkg-deb", "-Zgzip", "-b", stage, debName).CombinedOutput()
	if err != nil {
		fmt.Printf("[!] Error in dpkg-deb:\n%s\n", string(out))
		fmt.Println("    Note: You need to install dpkg on your Mac (brew install dpkg)")
	} else {
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
}

func prompt(msg string, defaultVal string) string {
	reader := bufio.NewReader(os.Stdin)
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", msg, defaultVal)
	} else {
		fmt.Printf("%s: ", msg)
	}
	
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input == "" {
		return defaultVal
	}
	return input
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
	sdkDir := filepath.Dir(targetPath)
	os.MkdirAll(sdkDir, 0755)
	
	fmt.Printf("[gios] Downloading iOS %s SDK from Theos project...\n", version)
	
	fileName := fmt.Sprintf("iPhoneOS%s.sdk.tar.xz", version)
	url := fmt.Sprintf("https://github.com/theos/sdks/releases/download/master-146e41f/%s", fileName)
	tarPath := filepath.Join(sdkDir, fileName)
	
	cmd := exec.Command("curl", "-L", "-o", tarPath, url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("curl failed: %v", err)
	}
	
	fmt.Println("[gios] Extracting SDK... (this may take a minute)")
	cmdExt := exec.Command("tar", "-xf", tarPath, "-C", sdkDir)
	if err := cmdExt.Run(); err != nil {
		return fmt.Errorf("tar extraction failed: %v", err)
	}
	
	os.Remove(tarPath)
	fmt.Println("[gios] SDK installed successfully!")
	return nil
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
	fmt.Println("[gios] Checking for updates on github.com/nikitacontreras/gios...")
	
	// Utilizar la API de releases de GitHub para la version de tag (o pre-release via una release "latest")
	req, err := http.NewRequest("GET", "https://api.github.com/repos/nikitacontreras/gios/releases/latest", nil)
	if err != nil {
		fmt.Printf("[!] Could not check for updates: %v\n", err)
		return
	}
	// Add arbitrary Github token header if rate limited, or just let it be public
	client := &http.Client{}
	resp, err := client.Do(req)
	
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("[!] Error fetching latest release from GitHub. (HTTP %d)\n", resp.StatusCode)
		return
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &release)

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

func initProject() {
	cwd, _ := os.Getwd()
	baseName := filepath.Base(cwd)
	
	fmt.Println("\n=============================================")
	fmt.Println("    GIOS Interactive Project Initialization  ")
	fmt.Println("=============================================\n")

	// Questions
	conf := Config{}
	conf.Name = prompt("1. Project Name", baseName)
	conf.PackageID = prompt("2. Package ID (e.g. com.yourname.app)", "com.gios."+conf.Name)
	conf.Version = prompt("3. Version", "1.0.0")
	conf.Output = "out_bin"
	conf.Main = "main.go"

	fmt.Println("\n4. Target Device Architecture:")
	fmt.Println("   [1] Legacy 32-bit (iPhone 2G to 5c, iPad 1 to 4)")
	fmt.Println("   [2] Modern 64-bit Rootless (iPhone 5s to 16, iPad Air/Pro)")
	
	targetOption := prompt("   Option", "1")
	
	isModern := false
	if targetOption == "2" {
		isModern = true
	}

	if !isModern {
		conf.Arch = "armv7"
		conf.GoVersion = "go1.14.15"
		conf.SDKVersion = "9.3"
		conf.Entitlements = "ents.plist"
	} else {
		conf.Arch = "arm64"
		conf.GoVersion = "system"
		conf.SDKVersion = "system"
		conf.Entitlements = "none" // Usually coretrust bypass or ldid without explicit ents handles it
	}

	daemonInput := prompt("\n5. Run as background service (LaunchDaemon)? (y/N)", "N")
	if strings.ToLower(daemonInput) == "y" {
		conf.Daemon = true
	}

	conf.Deploy.IP = prompt("\n6. Target Device IP Address (optional)", "192.168.1.XX")
	
	if conf.Arch == "arm64" {
		conf.Deploy.Path = "/var/jb/var/root"
	} else {
		conf.Deploy.Path = "/var/root/out_bin"
	}

	fmt.Println("\n=============================================")
	
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("[+] Initializing Go modules...")
		modGoVer := "1.14"
		if isModern {
			modGoVer = "1.22"
		}
		modData := fmt.Sprintf("module %s\n\ngo %s\n", conf.Name, modGoVer)
		ioutil.WriteFile("go.mod", []byte(modData), 0644)
	}

	// Create JSON
	jsonData, _ := json.MarshalIndent(conf, "", "  ")
	ioutil.WriteFile("gios.json", jsonData, 0644)
	fmt.Println("[+] Created gios.json")

	// Entitlements for legacy
	if !isModern {
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

	// Create main.go template if it doesn't exist
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		mainGoData := `package main

import "fmt"

func main() {
	fmt.Println("Hello from Gios!")
}
`
		ioutil.WriteFile("main.go", []byte(mainGoData), 0644)
		fmt.Println("[+] Created initial main.go")
	}

	fmt.Println("\nProject initialized successfully!")
	fmt.Println("Run 'gios build' to compile or 'gios install' to send to device.")
}
