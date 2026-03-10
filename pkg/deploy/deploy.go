package deploy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/utils"
)

type DaemonRequest struct {
	Command string `json:"command"`
	Payload string `json:"payload"`
	Remote  string `json:"remote,omitempty"`
}

type DaemonResponse struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

func CallDaemon(req DaemonRequest) (DaemonResponse, error) {
	socketPath := filepath.Join(config.GiosDir, "gios.sock")
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return DaemonResponse{}, err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return DaemonResponse{}, err
	}

	var resp DaemonResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return DaemonResponse{}, err
	}

	return resp, nil
}

func RemoteExec(c config.Config, cmd string) (string, error) {
	resp, err := CallDaemon(DaemonRequest{Command: "exec", Payload: cmd})
	if err == nil {
		if resp.Error != "" {
			return resp.Output, fmt.Errorf(resp.Error)
		}
		return resp.Output, nil
	}

	sshArgs := c.GetSSHArgs(cmd)
	out, err := exec.Command("ssh", sshArgs...).CombinedOutput()
	return string(out), err
}

func RemoteUpload(c config.Config, local, remote string) error {
	resp, err := CallDaemon(DaemonRequest{Command: "upload", Payload: local, Remote: remote})
	if err == nil {
		if resp.Error != "" {
			return fmt.Errorf(resp.Error)
		}
		return nil
	}

	target := c.GetSCPTarget(remote)
	scpArgs := c.GetSCPArgs()
	scpArgs = append(scpArgs, local, target)
	return exec.Command("scp", scpArgs...).Run()
}

func EnsureUSBTunnel(conf config.Config) bool {
	if !conf.Deploy.USB {
		return true
	}

	if _, err := exec.LookPath("idevice_id"); err != nil {
		fmt.Printf("%s[!] Error: 'idevice_id' not found. Please install libimobiledevice%s\n", utils.ColorRed, utils.ColorReset)
		return false
	}

	if _, err := exec.LookPath("iproxy"); err != nil {
		fmt.Printf("%s[!] Error: 'iproxy' not found. Please install libimobiledevice.%s\n", utils.ColorRed, utils.ColorReset)
		return false
	}

	out, _ := exec.Command("idevice_id", "-l").Output()
	if len(strings.TrimSpace(string(out))) == 0 {
		fmt.Printf("%s[!] Warning: No iOS device detected via USB. Ensure it's plugged in.%s\n", utils.ColorYellow, utils.ColorReset)
		return false
	}

	return true
}

func Run(unsafeFlag, watch bool) {
	conf := config.LoadConfig()
	isDylib := strings.HasSuffix(conf.Output, ".dylib")
	if !conf.Deploy.USB && (conf.Deploy.IP == "" || conf.Deploy.IP == "192.168.1.XX") {
		fmt.Println("[gios] Error: Deployment IP is not configured.")
		return
	}

	targetDisp := conf.Deploy.IP
	if conf.Deploy.USB {
		targetDisp = "USB Device"
		if !EnsureUSBTunnel(conf) {
			return
		}
	}
	fmt.Printf("[gios] Sending to %s...\n", targetDisp)

	utils.DrawProgress("Uploading", 50)
	if err := RemoteUpload(conf, conf.Output, conf.Deploy.Path); err != nil {
		fmt.Printf("\n[gios] Error uploading file: %v\n", err)
		return
	}

	if isDylib {
		plistName := strings.TrimSuffix(conf.Output, ".dylib") + ".plist"
		localPlist := "filter.plist"
		if _, err := os.Stat(localPlist); os.IsNotExist(err) {
			plistContent := `{"Filter":{"Bundles":["com.apple.springboard"]}}`
			ioutil.WriteFile("tmp_filter.plist", []byte(plistContent), 0644)
			localPlist = "tmp_filter.plist"
		}
		
		destPlist := filepath.Join(conf.Deploy.Path, plistName)
		RemoteUpload(conf, localPlist, destPlist)
		os.Remove("tmp_filter.plist")
	}
	utils.DrawProgress("Uploading", 100)

	if isDylib {
		TriggerRestart(conf)
		return
	}

	if watch { return } // If it's silent watch mode, don't execute shell interactively

	runPath := conf.Deploy.Path
	outputBase := filepath.Base(conf.Output)
	deployBase := filepath.Base(runPath)

	if (strings.HasSuffix(runPath, "/") || !strings.Contains(deployBase, ".")) && deployBase != outputBase {
		runPath = filepath.Join(runPath, outputBase)
	}

	fmt.Printf("[gios] Executing %s on device...\n", runPath)
	fmt.Println("--------------------------------------------------")

	sshArgs := conf.GetSSHArgs(fmt.Sprintf("chmod +x %s && env GOGC=20 GOMAXPROCS=1 GODEBUG=asyncpreemptoff=1 %s", runPath, runPath))
	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func CreateDeb(unsafeFlag bool) {
	conf := config.LoadConfig()
	pkgID := conf.PackageID
	version := conf.Version
	if pkgID == "" {
		pkgID = "com.gios." + conf.Name
	}

	stage := filepath.Join(os.TempDir(), "gios_deb_stage")
	os.RemoveAll(stage)

	debianDir := filepath.Join(stage, "DEBIAN")
	var binPath string
	if conf.Arch == "arm64" {
		binPath = filepath.Join(stage, "var", "jb", "usr", "bin")
	} else {
		binPath = filepath.Join(stage, "usr", "bin")
	}

	os.MkdirAll(debianDir, 0755)
	os.MkdirAll(binPath, 0755)

	if strings.HasSuffix(conf.Output, ".dylib") {
		var tweakDir string
		if conf.Arch == "arm64" {
			tweakDir = filepath.Join(stage, "var", "jb", "Library", "MobileSubstrate", "DynamicLibraries")
		} else {
			tweakDir = filepath.Join(stage, "Library", "MobileSubstrate", "DynamicLibraries")
		}
		os.MkdirAll(tweakDir, 0755)
		
		exec.Command("cp", conf.Output, filepath.Join(tweakDir, conf.Output)).Run()
		
		plistName := strings.TrimSuffix(conf.Output, ".dylib") + ".plist"
		if _, err := os.Stat("filter.plist"); err == nil {
			exec.Command("cp", "filter.plist", filepath.Join(tweakDir, plistName)).Run()
		} else {
			plistContent := `{"Filter":{"Bundles":["com.apple.springboard"]}}`
			ioutil.WriteFile(filepath.Join(tweakDir, plistName), []byte(plistContent), 0644)
		}
	} else {
		exec.Command("cp", conf.Output, filepath.Join(binPath, conf.Output)).Run()
		os.Chmod(filepath.Join(binPath, conf.Output), 0755)
	}

	debArch := conf.Arch
	control := fmt.Sprintf("Package: %s\nName: %s\nVersion: %s\nArchitecture: %s\nDescription: App created with Gios\nMaintainer: Gios User\nAuthor: Gios\nSection: Utilities\n",
		pkgID, conf.Name, version, debArch)
	ioutil.WriteFile(filepath.Join(debianDir, "control"), []byte(control), 0644)

	if conf.Daemon {
		var daemonDir string
		if conf.Arch == "arm64" {
			daemonDir = filepath.Join(stage, "var", "jb", "Library", "LaunchDaemons")
		} else {
			daemonDir = filepath.Join(stage, "Library", "LaunchDaemons")
		}
		os.MkdirAll(daemonDir, 0755)

		plistName := fmt.Sprintf("%s.plist", pkgID)
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
		ioutil.WriteFile(filepath.Join(daemonDir, plistName), []byte(plistContent), 0644)
	}

	debName := fmt.Sprintf("%s_%s_%s.deb", conf.Name, version, debArch)
	exec.Command("dpkg-deb", "-b", stage, debName).Run()
	fmt.Printf("[gios] Created package: %s\n", debName)
}

func InstallDeb(unsafeFlag bool) {
	conf := config.LoadConfig()
	debArch := conf.Arch
	version := conf.Version
	debName := fmt.Sprintf("%s_%s_%s.deb", conf.Name, version, debArch)

	fmt.Println("[gios] Installing .deb package on device...")
	RemoteUpload(conf, debName, "/tmp/"+debName)
	RemoteExec(conf, "dpkg -i /tmp/"+debName+" && rm /tmp/"+debName)
	
	if strings.HasSuffix(conf.Output, ".dylib") || conf.Daemon {
		TriggerRestart(conf)
	}
	fmt.Println("[+] Installed successfully.")
}

func TriggerRestart(conf config.Config) {
	fmt.Println("[gios] Triggering Respring...")
	RemoteExec(conf, "killall -9 SpringBoard BackBoard")
}

func Connect() {
	conf := config.LoadConfig()
	if conf.Deploy.USB {
		if !EnsureUSBTunnel(conf) {
			return
		}
	}
	
	sshKeyPath := filepath.Join(config.GiosDir, "id_rsa")
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		fmt.Println("[gios] SSH key not found. Generating a new one...")
		exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-f", sshKeyPath, "-N", "").Run()
	}

	pubKeyPath := sshKeyPath + ".pub"
	pubKey, _ := ioutil.ReadFile(pubKeyPath)

	fmt.Printf("[gios] Checking SSH access to %s...\n", conf.Deploy.IP)
	
	sshArgs := conf.GetSSHArgs("echo auth_ok")
	cmd := exec.Command("ssh", sshArgs...)
	out, err := cmd.CombinedOutput()

	if err != nil || !strings.Contains(string(out), "auth_ok") {
		fmt.Println("[gios] Public key not authorized. Attempting authorize...")
		sshInstall := fmt.Sprintf("mkdir -p ~/.ssh && echo '%s' >> ~/.ssh/authorized_keys && chmod 700 ~/.ssh && chmod 600 ~/.ssh/authorized_keys", strings.TrimSpace(string(pubKey)))
		
		target := "root@" + conf.Deploy.IP
		port := "22"
		if conf.Deploy.USB {
			target = "root@127.0.0.1"
			port = "2222"
		}

		fmt.Println("[!] Please enter device password (default is 'alpine'):")
		installCmd := exec.Command("ssh", "-p", port, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", target, sshInstall)
		installCmd.Stdin = os.Stdin
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		installCmd.Run()
	}

	fmt.Println("[+] Connection established and authorized.")
}

func Disconnect() {
	fmt.Println("[gios] Closing all background tunnels...")
	exec.Command("pkill", "iproxy").Run()
	fmt.Println("[+] Disconnected.")
}

func RunShell() {
	conf := config.LoadConfig()
	if conf.Deploy.USB {
		EnsureUSBTunnel(conf)
	}
	sshArgs := conf.GetSSHArgs("")
	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
