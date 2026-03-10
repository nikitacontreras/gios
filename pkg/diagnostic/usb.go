package diagnostic

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/diagnostics"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
	"github.com/danielpaulus/go-ios/ios/instruments"
	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/deploy"
	"github.com/nikitastrike/gios/pkg/sdk"
	"github.com/nikitastrike/gios/pkg/utils"
)

func RunInfo() {
	fmt.Printf("%s[gios]%s Fetching detailed device information natively...\n", utils.ColorCyan, utils.ColorReset)
	
	device, err := deploy.GetFirstDevice()
	if err != nil {
		fmt.Printf("%s[!] Error:%s %v\n", utils.ColorRed, utils.ColorReset, err)
		return
	}

	values, err := ios.GetValues(device)
	if err != nil {
		fmt.Printf("%s[!] Error:%s Could not retrieve lockdown values: %v\n", utils.ColorRed, utils.ColorReset, err)
		return
	}

	v := values.Value
	fmt.Println(utils.ColorBold + "--- Device Details ---" + utils.ColorReset)
	fmt.Printf("%s%-18s%s %s\n", utils.ColorCyan, "DeviceName:", utils.ColorReset, v.DeviceName)
	fmt.Printf("%s%-18s%s %s\n", utils.ColorCyan, "ProductType:", utils.ColorReset, v.ProductType)
	fmt.Printf("%s%-18s%s %s\n", utils.ColorCyan, "ProductVersion:", utils.ColorReset, v.ProductVersion)
	fmt.Printf("%s%-18s%s %s\n", utils.ColorCyan, "CPUArchitecture:", utils.ColorReset, v.CPUArchitecture)
	fmt.Printf("%s%-18s%s %s\n", utils.ColorCyan, "WiFiAddress:", utils.ColorReset, v.WiFiAddress)
	fmt.Printf("%s%-18s%s %s\n", utils.ColorCyan, "UniqueDeviceID:", utils.ColorReset, v.UniqueDeviceID)
	fmt.Printf("%s%-18s%s %s\n", utils.ColorCyan, "SerialNumber:", utils.ColorReset, v.SerialNumber)
	fmt.Printf("%s%-18s%s %t\n", utils.ColorCyan, "HasSiDP:", utils.ColorReset, v.HasSiDP)

	// Battery via diagnostics service
	diagConn, err := diagnostics.New(device)
	if err == nil {
		defer diagConn.Close()
		batt, err := diagConn.Battery()
		if err == nil {
			fmt.Println("\n" + utils.ColorBold + "--- Battery Status ---" + utils.ColorReset)
			fmt.Printf("%s%-18s%s %v mA\n", utils.ColorCyan, "InstantAmperage:", utils.ColorReset, batt.InstantAmperage)
			fmt.Printf("%s%-18s%s %v mAh\n", utils.ColorCyan, "CurrentCapacity:", utils.ColorReset, batt.CurrentCapacity)
			fmt.Printf("%s%-18s%s %v mV\n", utils.ColorCyan, "Voltage:", utils.ColorReset, batt.Voltage)
			fmt.Printf("%s%-18s%s %v\n", utils.ColorCyan, "IsCharging:", utils.ColorReset, batt.IsCharging)
		}
	}
}

func RunScreenshot() {
	fmt.Printf("%s[gios]%s Capturing screenshot natively via Instruments...\n", utils.ColorCyan, utils.ColorReset)
	
	device, err := deploy.GetFirstDevice()
	if err != nil {
		fmt.Printf("%s[!] Error:%s %v\n", utils.ColorRed, utils.ColorReset, err)
		return
	}

	svc, err := instruments.NewScreenshotService(device)
	if err != nil {
		fmt.Printf("%s[!] Error:%s Failed to start screenshot service: %v (Ensure DDI is mounted)\n", utils.ColorRed, utils.ColorReset, err)
		return
	}
	defer svc.Close()

	pngBytes, err := svc.TakeScreenshot()
	if err != nil {
		fmt.Printf("%s[!] Error:%s Capture failed: %v\n", utils.ColorRed, utils.ColorReset, err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("screenshot_%s.png", timestamp)
	
	err = os.WriteFile(filename, pngBytes, 0644)
	if err != nil {
		fmt.Printf("%s[!] Error:%s Failed to save file: %v\n", utils.ColorRed, utils.ColorReset, err)
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

	device, err := deploy.GetFirstDevice()
	if err != nil {
		fmt.Printf("%s[!] Error:%s %v\n", utils.ColorRed, utils.ColorReset, err)
		return
	}

	fmt.Printf("%s[gios]%s Sending native reboot command...\n", utils.ColorCyan, utils.ColorReset)
	err = diagnostics.Reboot(device)
	if err != nil {
		fmt.Printf("%s[!] Error:%s Could not reboot device: %v\n", utils.ColorRed, utils.ColorReset, err)
	} else {
		fmt.Println("[+] Reboot command sent successfully.")
	}
}

func RunMount() {
	device, err := deploy.GetFirstDevice()
	if err != nil {
		fmt.Printf("%s[!] Error:%s %v\n", utils.ColorRed, utils.ColorReset, err)
		return
	}

	val, err := ios.GetValues(device)
	if err != nil {
		fmt.Printf("%s[!] Error:%s Could not detect device version: %v\n", utils.ColorRed, utils.ColorReset, err)
		return
	}

	ver := val.Value.ProductVersion
	model := val.Value.ProductType
	platform := "iPhoneOS" // Default
	if strings.Contains(model, "AppleTV") { platform = "AppleTVOS" }
	if strings.Contains(model, "Watch") { platform = "WatchOS" }

	fmt.Printf("%s[gios]%s Detected %s %s via USB\n", utils.ColorCyan, utils.ColorReset, platform, ver)

	manifest, err := sdk.FetchAssetManifest()
	if err != nil {
		fmt.Printf("%s[!] Error:%s Could not fetch assets manifest: %v\n", utils.ColorRed, utils.ColorReset, err)
		return
	}

	var selectedIdx int = -1
	for i, ddi := range manifest.DDIs {
		if ddi.Version == ver && ddi.Platform == platform {
			selectedIdx = i
			break
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

	dmgPath := filepath.Join(ddiDir, "DeveloperDiskImage.dmg")

	if _, err := os.Stat(dmgPath); os.IsNotExist(err) {
		tempZip := filepath.Join(ddiDir, "ddi.zip")
		fmt.Printf("[gios] Downloading DDI for %s...\n", selected.Version)
		if err := sdk.DownloadURLToFile(selected.URL, tempZip, true); err != nil {
			fmt.Printf("[!] Download failed: %v\n", err)
			return
		}

		// Extract dmg and signature
		r, _ := zip.OpenReader(tempZip)
		for _, f := range r.File {
			if strings.HasSuffix(f.Name, ".dmg") || strings.HasSuffix(f.Name, ".signature") {
				target := filepath.Join(ddiDir, filepath.Base(f.Name))
				if strings.HasSuffix(f.Name, ".dmg") { dmgPath = target }
				dst, _ := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				src, _ := f.Open()
				io.Copy(dst, src)
				dst.Close()
				src.Close()
			}
		}
		r.Close()
		os.Remove(tempZip)
	}

	fmt.Printf("%s[gios]%s Mounting Image natively...\n", utils.ColorCyan, utils.ColorReset)
	err = imagemounter.MountImage(device, dmgPath)
	if err != nil {
		fmt.Printf("%s[!] Error:%s Mounting failed: %v\n", utils.ColorRed, utils.ColorReset, err)
		return
	}
	fmt.Printf("%s[+] Success!%s Developer Disk Image mounted successfully.\n", utils.ColorGreen, utils.ColorReset)
}
