package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func runInfo() {
	fmt.Printf("%s[gios]%s Fetching detailed device information via USB...\n", ColorCyan, ColorReset)
	
	cmd := exec.Command("ideviceinfo")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s[!] Error:%s libimobiledevice is required for this command. (%v)\n", ColorRed, ColorReset, err)
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

	fmt.Println(ColorBold + "--- Device Details ---" + ColorReset)
	for _, line := range lines {
		for _, field := range importantFields {
			if strings.HasPrefix(line, field) {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					fmt.Printf("%s%-18s%s %s\n", ColorCyan, parts[0]+":", ColorReset, strings.TrimSpace(parts[1]))
				}
			}
		}
	}
	
	// Try to get battery info too
	battCmd := exec.Command("ideviceinfo", "-q", "com.apple.mobile.battery")
	battOut, _ := battCmd.CombinedOutput()
	if len(battOut) > 0 {
		fmt.Println("\n" + ColorBold + "--- Battery Status ---" + ColorReset)
		battLines := strings.Split(string(battOut), "\n")
		for _, bl := range battLines {
			if strings.Contains(bl, "BatteryCurrentCapacity") || strings.Contains(bl, "ExternalConnected") {
				parts := strings.SplitN(bl, ":", 2)
				if len(parts) == 2 {
					fmt.Printf("%s%-18s%s %s\n", ColorCyan, parts[0]+":", ColorReset, strings.TrimSpace(parts[1]))
				}
			}
		}
	}
}

func runScreenshot() {
	fmt.Printf("%s[gios]%s Capturing screenshot via USB...\n", ColorCyan, ColorReset)
	
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("screenshot_%s.png", timestamp)
	
	cmd := exec.Command("idevicescreenshot", filename)
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s[!] Error:%s Capture failed. (Device may need a Developer Disk Image mounted)\n", ColorRed, ColorReset)
		return
	}

	absPath, _ := filepath.Abs(filename)
	fmt.Printf("%s[+] Success!%s Screenshot saved to: %s\n", ColorGreen, ColorReset, absPath)
}

func runReboot() {
	var confirm string
	fmt.Printf("%s[?]%s Are you sure you want to reboot the connected device? (y/N): ", ColorYellow, ColorReset)
	fmt.Scanln(&confirm)
	
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Aborted.")
		return
	}

	fmt.Printf("%s[gios]%s Sending reboot command...\n", ColorCyan, ColorReset)
	cmd := exec.Command("idevicediagnostics", "restart")
	if err := cmd.Run(); err != nil {
		fmt.Printf("%s[!] Error:%s Could not reboot device: %v\n", ColorRed, ColorReset, err)
	} else {
		fmt.Println("[+] Reboot command sent successfully.")
	}
}

type DeviceInfo struct {
	Version  string
	Platform string
}

func getConnectedDeviceInfo() (*DeviceInfo, error) {
	cmd := exec.Command("ideviceinfo", "-s") // simple output
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
		info.Platform = "iPhoneOS" // Default
	}

	return info, nil
}

func runMount() {
	var dmgPath string
	var sigPath string

	if len(os.Args) < 3 {
		// INTENTO DE AUTO-DETECCIÓN
		fmt.Printf("%s[gios]%s Detecting connected device via USB...\n", ColorCyan, ColorReset)
		device, devErr := getConnectedDeviceInfo()
		
		manifest, err := fetchAssetManifest()
		if err != nil {
			fmt.Printf("%s[!] Error:%s Could not fetch assets manifest: %v\n", ColorRed, ColorReset, err)
			return
		}

		if len(manifest.DDIs) == 0 {
			fmt.Println("[!] No DDIs available in the manifest.")
			return
		}

		var selectedIdx int = -1

		if devErr == nil && device != nil && device.Version != "" {
			fmt.Printf("[+] Detected: %s %s\n", device.Platform, device.Version)
			
			// 1. Coincidencia exacta
			for i, ddi := range manifest.DDIs {
				if ddi.Version == device.Version && ddi.Platform == device.Platform {
					selectedIdx = i
					break
				}
			}

			// 2. Coincidencia por prefijo (p.ej. 15.6.1 -> 15.6)
			if selectedIdx == -1 {
				for i, ddi := range manifest.DDIs {
					if ddi.Platform == device.Platform && (strings.HasPrefix(device.Version, ddi.Version) || strings.HasPrefix(ddi.Version, device.Version)) {
						selectedIdx = i
						break
					}
				}
			}

			// 3. Coincidencia por "Misma versión mayor" si no hubo suerte
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
		} else {
			fmt.Printf("%s[!]%s No device hardware detected. Falling back to manual list...\n", ColorYellow, ColorReset)
		}

		// Sistema Inteligente de Sugerencia
		if selectedIdx != -1 {
			ddi := manifest.DDIs[selectedIdx]
			fmt.Printf("\n%s[Smart Suggest]%s Target matched: %s %s\n", ColorGreen, ColorReset, ddi.Platform, ddi.Version)
			ans := prompt("      -> Press [Enter] to mount this, or type 'L' to see all list", "")
			if strings.ToLower(ans) == "l" {
				selectedIdx = -1 // Forzar modo lista
			}
		}

		if selectedIdx == -1 {
			fmt.Println("\nAvailable Developer Disk Images:")
			for i, ddi := range manifest.DDIs {
				fmt.Printf("  [%d] %s %s\n", i+1, ddi.Platform, ddi.Version)
			}

			choiceStr := prompt("\nEnter number", "")
			var choice int
			fmt.Sscanf(choiceStr, "%d", &choice)

			if choice < 1 || choice > len(manifest.DDIs) {
				fmt.Println("[!] Invalid selection.")
				return
			}
			selectedIdx = choice - 1
		}

		selected := manifest.DDIs[selectedIdx]
		ddiDir := filepath.Join(giosDir, "ddis", selected.Version)
		os.MkdirAll(ddiDir, 0755)

		// Check if the asset is a zip file
		if strings.HasSuffix(selected.URL, ".zip") {
			zipFilePath := filepath.Join(ddiDir, "DeveloperDiskImage.zip")
			
			// Download the zip file if missing
			if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
				fmt.Printf("[gios] Downloading DeveloperDiskImage.zip for iOS %s...\n", selected.Version)
				if err := downloadURLToFile(selected.URL, zipFilePath, true); err != nil {
					fmt.Printf("[!] Download fail: %v\n", err)
					return
				}
			}

			fmt.Printf("[gios] Unzipping DeveloperDiskImage.zip...\n")
			r, err := zip.OpenReader(zipFilePath)
			if err != nil {
				fmt.Printf("%s[!] Error:%s Failed to open zip file: %v\n", ColorRed, ColorReset, err)
				return
			}
			defer r.Close()

			for _, f := range r.File {
				fpath := filepath.Join(ddiDir, f.Name)

				if f.FileInfo().IsDir() {
					os.MkdirAll(fpath, os.ModePerm)
					continue
				}

				if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
					fmt.Printf("%s[!] Error:%s Failed to create directory %s: %v\n", ColorRed, ColorReset, filepath.Dir(fpath), err)
					return
				}

				outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				if err != nil {
					fmt.Printf("%s[!] Error:%s Failed to open file %s for writing: %v\n", ColorRed, ColorReset, fpath, err)
					return
				}

				rc, err := f.Open()
				if err != nil {
					fmt.Printf("%s[!] Error:%s Failed to open file %s in zip: %v\n", ColorRed, ColorReset, f.Name, err)
					outFile.Close()
					return
				}

				_, err = io.Copy(outFile, rc)
				outFile.Close()
				rc.Close()

				if err != nil {
					fmt.Printf("%s[!] Error:%s Failed to extract file %s: %v\n", ColorRed, ColorReset, f.Name, err)
					return
				}

			}

			// AFTER EXTRACTION: Find the actual DMG and Signature files recursively
			finalDmg := ""
			finalSig := ""
			filepath.Walk(ddiDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				if strings.HasSuffix(path, ".dmg") {
					finalDmg = path
				} else if strings.HasSuffix(path, ".signature") || strings.HasSuffix(path, ".dmg.signature") {
					finalSig = path
				}
				return nil
			})

			if finalDmg != "" {
				dmgPath = finalDmg
			}
			if finalSig != "" {
				sigPath = finalSig
			} else {
				sigPath = "" // No signature found
			}

			if finalDmg == "" {
				fmt.Printf("%s[!] Error:%s No .dmg file found in the zip archive.\n", ColorRed, ColorReset)
				return
			}

			// Version Mismatch Check
			if device != nil && device.Version != "" {
				if !strings.HasPrefix(selected.Version, strings.Split(device.Version, ".")[0]) {
					fmt.Printf("\n%s[!] WARNING:%s You are trying to mount DDI %s on a device running %s.\n", ColorYellow, ColorReset, selected.Version, device.Version)
					fmt.Printf("    This will likely fail with 'Unknown error'.\n")
					ans := prompt("    Continue anyway? (y/N)", "N")
					if strings.ToLower(ans) != "y" {
						return
					}
				}
			}

			// Clean up the downloaded zip file
			os.Remove(zipFilePath)

		} else {
			// Original logic for non-zip assets
			dmgPath = filepath.Join(ddiDir, "DeveloperDiskImage.dmg")
			sigPath = dmgPath + ".signature"

			// Download DMG if missing
			if _, err := os.Stat(dmgPath); os.IsNotExist(err) {
				fmt.Printf("[gios] Downloading DeveloperDiskImage for iOS %s...\n", selected.Version)
				if err := downloadURLToFile(selected.URL, dmgPath, true); err != nil {
					fmt.Printf("[!] Download fail: %v\n", err)
					return
				}
			}

			// Download Signature if missing (if URL is provided)
			if selected.SigURL != "" {
				if _, err := os.Stat(sigPath); os.IsNotExist(err) {
					fmt.Printf("[gios] Downloading Signature for iOS %s...\n", selected.Version)
					if err := downloadURLToFile(selected.SigURL, sigPath, false); err != nil {
						fmt.Printf("[!] Signature download fail: %v\n", err)
						return
					}
				}
			} else {
				sigPath = "" // Ensure it's empty if no URL
			}

			// Version Mismatch Check
			if device != nil && device.Version != "" {
				if !strings.HasPrefix(selected.Version, strings.Split(device.Version, ".")[0]) {
					fmt.Printf("\n%s[!] WARNING:%s You are trying to mount DDI %s on a device running %s.\n", ColorYellow, ColorReset, selected.Version, device.Version)
					fmt.Printf("    This will likely fail with 'Unknown error'.\n")
					ans := prompt("    Continue anyway? (y/N)", "N")
					if strings.ToLower(ans) != "y" {
						return
					}
				}
			}
		}
	} else {
		// Modo manual: usar ruta proporcionada
		dmgPath = os.Args[2]
		sigPath = dmgPath + ".signature"

		if _, err := os.Stat(dmgPath); os.IsNotExist(err) {
			fmt.Printf("%s[!] Error:%s DMG file not found: %s\n", ColorRed, ColorReset, dmgPath)
			return
		}
	}

	if sigPath != "" { // Only check if sigPath was actually set
		if _, err := os.Stat(sigPath); os.IsNotExist(err) {
			fmt.Printf("%s[!] Warning:%s Signature file (.signature) not found at %s\n", ColorYellow, ColorReset, sigPath)
			fmt.Println("    Usually required for iOS 5+. Attempting anyway...")
		}
	} else {
		fmt.Printf("%s[!] Warning:%s Signature file (.signature) was not provided or found.\n", ColorYellow, ColorReset)
		fmt.Println("    Usually required for iOS 5+. Attempting anyway...")
	}

	fmt.Printf("%s[gios]%s Mounting Image... (This targets the connected USB device)\n", ColorCyan, ColorReset)
	
	var cmd *exec.Cmd
	if sigPath != "" {
		cmd = exec.Command("ideviceimagemounter", dmgPath, sigPath)
	} else {
		cmd = exec.Command("ideviceimagemounter", dmgPath)
	}
	out, err := cmd.CombinedOutput()
	
	if err != nil {
		fmt.Printf("%s[!] Error:%s Mounting failed: %v\n", ColorRed, ColorReset, err)
		if len(out) > 0 {
			fmt.Printf("    Output: %s\n", strings.TrimSpace(string(out)))
		}
		return
	}

	fmt.Printf("%s[+] Success!%s Developer Disk Image mounted successfully.\n", ColorGreen, ColorReset)
}
