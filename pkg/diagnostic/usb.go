package diagnostic

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/sdk"
	"github.com/nikitastrike/gios/pkg/utils"
)

func RunInfo() {
	fmt.Printf("%s[gios]%s Fetching detailed device information via USB...\n", utils.ColorCyan, utils.ColorReset)
	
	cmd := exec.Command("ideviceinfo")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s[!] Error:%s libimobiledevice is required for this command. (%v)\n", utils.ColorRed, utils.ColorReset, err)
		return
	}

	lines := strings.Split(string(out), "\n")
	importantFields := []string{
		"DeviceName:",
		"ProductType:",
		"ProductVersion:",
		"CPUArchitecture:",
		"EthernetAddress:",
		"WiFiAddress:",
		"UniqueDeviceID:",
		"SerialNumber:",
		"HasSiDP:",
	}

	fmt.Println(utils.ColorBold + "--- Device Details ---" + utils.ColorReset)
	for _, line := range lines {
		for _, field := range importantFields {
			if strings.HasPrefix(line, field) {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					fmt.Printf("%s%-18s%s %s\n", utils.ColorCyan, parts[0]+":", utils.ColorReset, strings.TrimSpace(parts[1]))
				}
			}
		}
	}
	
	// Try to get battery info too
	battCmd := exec.Command("ideviceinfo", "-q", "com.apple.mobile.battery")
	battOut, _ := battCmd.CombinedOutput()
	if len(battOut) > 0 {
		fmt.Println("\n" + utils.ColorBold + "--- Battery Status ---" + utils.ColorReset)
		battLines := strings.Split(string(battOut), "\n")
		for _, bl := range battLines {
			if strings.Contains(bl, "BatteryCurrentCapacity") || strings.Contains(bl, "ExternalConnected") {
				parts := strings.SplitN(bl, ":", 2)
				if len(parts) == 2 {
					fmt.Printf("%s%-18s%s %s\n", utils.ColorCyan, parts[0]+":", utils.ColorReset, strings.TrimSpace(parts[1]))
				}
			}
		}
	}
}

func RunScreenshot() {
	fmt.Printf("%s[gios]%s Capturing screenshot via USB...\n", utils.ColorCyan, utils.ColorReset)
	
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("screenshot_%s.png", timestamp)
	
	cmd := exec.Command("idevicescreenshot", filename)
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s[!] Error:%s Capture failed. (Device may need a Developer Disk Image mounted)\n", utils.ColorRed, utils.ColorReset)
		return
	}

	absPath, _ := filepath.Abs(filename)
	fmt.Printf("%s[+] Success!%s Screenshot saved to: %s\n", utils.ColorGreen, utils.ColorReset, absPath)
}

func RunReboot() {
	var confirm string
	fmt.Printf("%s[?]%s Are you sure you want to reboot the connected device? (y/N): ", utils.ColorYellow, utils.ColorReset)
	fmt.Scanln(&confirm)
	
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Aborted.")
		return
	}

	fmt.Printf("%s[gios]%s Sending reboot command...\n", utils.ColorCyan, utils.ColorReset)
	cmd := exec.Command("idevicediagnostics", "restart")
	if err := cmd.Run(); err != nil {
		fmt.Printf("%s[!] Error:%s Could not reboot device: %v\n", utils.ColorRed, utils.ColorReset, err)
	} else {
		fmt.Println("[+] Reboot command sent successfully.")
	}
}

type DeviceInfo struct {
	Version  string
	Platform string
}

func getConnectedDeviceInfo() (*DeviceInfo, error) {
	cmd := exec.Command("ideviceinfo", "-s")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	info := &DeviceInfo{}
	lines := strings.Split(string(out), "\n")
	productType := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "ProductVersion:") {
			info.Version = strings.TrimSpace(strings.Split(line, ":")[1])
		}
		if strings.HasPrefix(line, "ProductType:") {
			productType = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}

    if strings.HasPrefix(productType, "iPhone") || strings.HasPrefix(productType, "iPad") || strings.HasPrefix(productType, "iPod") {
		info.Platform = "iPhoneOS"
	} else if strings.HasPrefix(productType, "AppleTV") {
		info.Platform = "AppleTVOS"
	} else if strings.HasPrefix(productType, "Watch") {
		info.Platform = "WatchOS"
	} else {
		info.Platform = "iPhoneOS"
	}

	return info, nil
}

func RunMount() {
	var dmgPath string
	var sigPath string

	if len(os.Args) < 3 {
		fmt.Printf("%s[gios]%s Detecting connected device via USB...\n", utils.ColorCyan, utils.ColorReset)
		device, devErr := getConnectedDeviceInfo()
		
		manifest, err := sdk.FetchAssetManifest()
		if err != nil {
			fmt.Printf("%s[!] Error:%s Could not fetch assets manifest: %v\n", utils.ColorRed, utils.ColorReset, err)
			return
		}

		if len(manifest.DDIs) == 0 {
			fmt.Println("[!] No DDIs available in the manifest.")
			return
		}

		var selectedIdx int = -1

		if devErr == nil && device != nil && device.Version != "" {
			fmt.Printf("[+] Detected: %s %s\n", device.Platform, device.Version)
			for i, ddi := range manifest.DDIs {
				if ddi.Version == device.Version && ddi.Platform == device.Platform {
					selectedIdx = i
					break
				}
			}
			if selectedIdx == -1 {
				for i, ddi := range manifest.DDIs {
					if ddi.Platform == device.Platform && (strings.HasPrefix(device.Version, ddi.Version) || strings.HasPrefix(ddi.Version, device.Version)) {
						selectedIdx = i
						break
					}
				}
			}
			if selectedIdx == -1 {
				devMajor := strings.Split(device.Version, ".")[0]
				for i, ddi := range manifest.DDIs {
					ddiMajor := strings.Split(ddi.Version, ".")[0]
					if ddi.Platform == device.Platform && ddiMajor == devMajor {
						selectedIdx = i
						break
					}
				}
			}
		}

		if selectedIdx != -1 {
			ddi := manifest.DDIs[selectedIdx]
			fmt.Printf("\n%s[Smart Suggest]%s Target matched: %s %s\n", utils.ColorGreen, utils.ColorReset, ddi.Platform, ddi.Version)
			ans := utils.Prompt("      -> Press [Enter] to mount this, or type 'L' to see all list", "")
			if strings.ToLower(ans) == "l" {
				selectedIdx = -1
			}
		}

		if selectedIdx == -1 {
			fmt.Println("\nAvailable Developer Disk Images:")
			for i, ddi := range manifest.DDIs {
				fmt.Printf("  [%d] %s %s\n", i+1, ddi.Platform, ddi.Version)
			}
			choiceStr := utils.Prompt("\nEnter number", "")
			var choice int
			fmt.Sscanf(choiceStr, "%d", &choice)
			if choice < 1 || choice > len(manifest.DDIs) {
				fmt.Println("[!] Invalid selection.")
				return
			}
			selectedIdx = choice - 1
		}

		selected := manifest.DDIs[selectedIdx]
		ddiDir := filepath.Join(config.GiosDir, "ddis", selected.Version)
		os.MkdirAll(ddiDir, 0755)

		if strings.HasSuffix(selected.URL, ".zip") {
			zipFilePath := filepath.Join(ddiDir, "DeveloperDiskImage.zip")
			if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
				fmt.Printf("[gios] Downloading DeveloperDiskImage.zip for iOS %s...\n", selected.Version)
				if err := sdk.DownloadURLToFile(selected.URL, zipFilePath, true); err != nil {
					fmt.Printf("[!] Download fail: %v\n", err)
					return
				}
			}

			fmt.Printf("[gios] Unzipping DeveloperDiskImage.zip...\n")
			r, err := zip.OpenReader(zipFilePath)
			if err != nil {
				fmt.Printf("%s[!] Error:%s Failed to open zip file: %v\n", utils.ColorRed, utils.ColorReset, err)
				return
			}
			defer r.Close()

			for _, f := range r.File {
				fpath := filepath.Join(ddiDir, f.Name)
				if f.FileInfo().IsDir() {
					os.MkdirAll(fpath, os.ModePerm)
					continue
				}
				os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
				outFile, _ := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				rc, _ := f.Open()
				io.Copy(outFile, rc)
				outFile.Close()
				rc.Close()
			}

			// Find DMG and Signature
			filepath.Walk(ddiDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() { return nil }
				if strings.HasSuffix(path, ".dmg") { dmgPath = path }
				if strings.HasSuffix(path, ".signature") || strings.HasSuffix(path, ".dmg.signature") { sigPath = path }
				return nil
			})
			os.Remove(zipFilePath)
		} else {
			dmgPath = filepath.Join(ddiDir, "DeveloperDiskImage.dmg")
			sigPath = dmgPath + ".signature"
			if _, err := os.Stat(dmgPath); os.IsNotExist(err) {
				sdk.DownloadURLToFile(selected.URL, dmgPath, true)
			}
			if selected.SigURL != "" {
				sdk.DownloadURLToFile(selected.SigURL, sigPath, false)
			}
		}
	} else {
		dmgPath = os.Args[2]
		sigPath = dmgPath + ".signature"
	}

	fmt.Printf("%s[gios]%s Mounting Image...\n", utils.ColorCyan, utils.ColorReset)
	var cmd *exec.Cmd
	if sigPath != "" {
		cmd = exec.Command("ideviceimagemounter", dmgPath, sigPath)
	} else {
		cmd = exec.Command("ideviceimagemounter", dmgPath)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s[!] Error:%s Mounting failed: %v\n", utils.ColorRed, utils.ColorReset, err)
		if len(out) > 0 {
			fmt.Printf("    Output: %s\n", string(out))
		}
		return
	}
	fmt.Printf("%s[+] Success!%s Developer Disk Image mounted successfully.\n", utils.ColorGreen, utils.ColorReset)
}
