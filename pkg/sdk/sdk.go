package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nikitastrike/gios/pkg/config"
	"github.com/schollz/progressbar/v3"
)

const RemoteAssetsURL = "https://raw.githubusercontent.com/nikitacontreras/gios-platform-assets/main/assets.json"

type SDKInfo struct {
	Name     string
	URL      string
	Platform string
	Hash     string
	Version  string
}

func GetDownloadedSDKs() []string {
	var downloaded []string
	sdksDir := filepath.Join(config.GiosDir, "sdks")
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

func FetchAvailableSDKs() ([]SDKInfo, error) {
	manifest, err := FetchAssetManifest()
	var sdkList []SDKInfo

	if err != nil {
		// Legacy fallback if gios-platform-assets fails
		req, _ := http.NewRequest("GET", "https://api.github.com/repos/theos/sdks/releases/latest", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var release struct{ Assets []struct{ Name, BrowserDownloadURL string } `json:"assets"` }
			json.NewDecoder(resp.Body).Decode(&release)
			for _, asset := range release.Assets {
				if strings.HasSuffix(asset.Name, ".tar.xz") || strings.HasSuffix(asset.Name, ".tar.gz") {
					name := strings.TrimSuffix(strings.TrimSuffix(asset.Name, ".tar.xz"), ".tar.gz")
					sdkList = append(sdkList, SDKInfo{
						Name:     name,
						URL:      asset.BrowserDownloadURL,
						Platform: "iOS",
						Version:  name,
					})
				}
			}
		}
	} else {
		for _, sdk := range manifest.SDKs {
			sdkList = append(sdkList, SDKInfo{
				Name:     sdk.Name,
				URL:      sdk.URL,
				Platform: sdk.Platform,
				Hash:     sdk.Hash,
				Version:  sdk.Name,
			})
		}
	}

	return sdkList, nil
}

func ListSDKs() {
	sdks, err := FetchAvailableSDKs()
	if err != nil {
		fmt.Printf("Error fetching SDKs: %v\n", err)
		return
	}

	downloaded := GetDownloadedSDKs()
	downloadedMap := make(map[string]bool)
	for _, d := range downloaded {
		downloadedMap[d] = true
	}

	fmt.Println("\nAvailable iOS SDKs:")
	for _, sdk := range sdks {
		status := " "
		if downloadedMap[sdk.Name] {
			status = "*"
		}
		fmt.Printf(" [%s] %s (%s)\n", status, sdk.Name, sdk.Platform)
	}
}

func AddSDK() {
	// This will now be handled by cmd TUI for better experience, 
	// but keeping a CLI version for non-interactive use.
	sdks, _ := FetchAvailableSDKs()
	if len(sdks) == 0 {
		fmt.Println("No SDKs found.")
		return
	}
	// (Simplified CLI select omitted, we use TUI in cmd package)
}

func RemoveSDKByName(name string) {
	path := filepath.Join(config.GiosDir, "sdks", name)
	os.RemoveAll(path)
}

func EnsureSDK(version, targetPath string) error {
	sdks, err := FetchAvailableSDKs()
	if err != nil {
		return err
	}
	var selected *SDKInfo
	for _, s := range sdks {
		if strings.Contains(s.Name, version) {
			selected = &s
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("SDK version %s not found in manifest", version)
	}
	return EnsureSDKFromURL(selected.Name, targetPath, selected.URL)
}

func DownloadURLToFile(url, targetPath string, showProgress bool) error {
	escapedURL := strings.ReplaceAll(url, " ", "%20")
	req, err := http.NewRequest("GET", escapedURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	var reader io.Reader = resp.Body
	if showProgress {
		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			"Downloading",
		)
		reader = io.TeeReader(resp.Body, bar)
	}

	_, err = io.Copy(f, reader)
	return err
}

func EnsureSDKFromURL(version, targetPath, customUrl string) error {
	sdkDir := filepath.Dir(targetPath)
	os.MkdirAll(sdkDir, 0755)

	fileName := filepath.Base(customUrl)
	tarPath := filepath.Join(sdkDir, fileName)

	if err := DownloadURLToFile(customUrl, tarPath, true); err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	fmt.Printf("[gios] Extracting SDK: %s...\n", version)
	// Create target dir
	os.MkdirAll(targetPath, 0755)
	
	// Determine extraction based on extension
	extractCmd := "tar"
	extractArgs := []string{"-xf", tarPath, "-C", targetPath, "--strip-components=1"}
	if strings.Contains(fileName, ".gz") {
		extractArgs = append([]string{"-z"}, extractArgs...)
	}

	if err := exec.Command(extractCmd, extractArgs...).Run(); err != nil {
		return fmt.Errorf("extraction failed: %v", err)
	}

	// Removal of tarball
	os.Remove(tarPath)
	return nil
}

func FetchAssetManifest() (*config.PlatformAssets, error) {
	resp, err := http.Get(RemoteAssetsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var manifest config.PlatformAssets
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
