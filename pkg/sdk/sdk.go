package sdk

import (
	"archive/tar"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"

	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/utils"
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
		if f.IsDir() && strings.HasSuffix(f.Name(), ".sdk") {
			downloaded = append(downloaded, f.Name())
		}
	}
	return downloaded
}

func FetchAvailableSDKs() ([]SDKInfo, error) {
	manifest, err := FetchAssetManifest()
	if err != nil {
		return nil, fmt.Errorf("could not fetch manifest from %s: %v", RemoteAssetsURL, err)
	}

	var sdkList []SDKInfo
	for _, sdk := range manifest.SDKs {
		sdkList = append(sdkList, SDKInfo{
			Name:     sdk.Name,
			URL:      sdk.URL,
			Platform: sdk.Platform,
			Hash:     sdk.Hash,
			Version:  sdk.Name,
		})
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
	return EnsureSDKFromURL(selected.Name, targetPath, selected.URL, selected.Hash)
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

func VerifyMD5(filePath, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

func EnsureSDKFromURL(version, targetPath, customUrl, expectedHash string) error {
	sdkDir := filepath.Dir(targetPath)
	os.MkdirAll(sdkDir, 0755)

	// 1. Check if already installed and hash matches
	hashFile := filepath.Join(targetPath, ".md5")
	if expectedHash != "" {
		if storedHash, err := ioutil.ReadFile(hashFile); err == nil {
			if strings.TrimSpace(string(storedHash)) == expectedHash {
				fmt.Printf("%s[gios]%s SDK %s is already installed and verified.\n", utils.ColorGreen, utils.ColorReset, version)
				return nil // Matches, no need to do anything
			}
		}
	} else {
		// Fallback: if no hash provided, check if folder exists
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Printf("%s[gios]%s SDK %s is already installed.\n", utils.ColorGreen, utils.ColorReset, version)
			return nil
		}
	}

	fileName := filepath.Base(customUrl)
	tarPath := filepath.Join(sdkDir, fileName)

	// Download only if needed (check if file exists and matches hash)
	needDownload := true
	if _, err := os.Stat(tarPath); err == nil && expectedHash != "" {
		if VerifyMD5(tarPath, expectedHash) == nil {
			needDownload = false
			fmt.Printf("[gios] Existing archive %s matches hash, skipping download.\n", fileName)
		}
	}

	if needDownload {
		if err := DownloadURLToFile(customUrl, tarPath, true); err != nil {
			return fmt.Errorf("download failed: %v", err)
		}
		// Verify Hash if provided
		if expectedHash != "" {
			fmt.Printf("[gios] Verifying integrity...\n")
			if err := VerifyMD5(tarPath, expectedHash); err != nil {
				os.Remove(tarPath)
				return err
			}
		}
	}

	fmt.Printf("[gios] Installing SDK: %s...\n", version)
	// Preparar target dir (limpiar si existe pero no coincidía el hash)
	os.RemoveAll(targetPath)
	os.MkdirAll(targetPath, 0755)
	
	fmt.Printf("[gios] Unpacking archive (Native Go Extractor)...\n")
	
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %v", err)
	}
	defer f.Close()

	// 1. Descomprimir XZ
	xzReader, err := xz.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to initialize xz reader: %v", err)
	}

	// 2. Leer TAR
	tr := tar.NewReader(xzReader)
	
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %v", err)
		}

		// Implementar --strip-components=1 logic
		parts := strings.Split(filepath.ToSlash(header.Name), "/")
		if len(parts) <= 1 {
			continue // Saltamos la carpeta raíz misma
		}
		
		relPath := filepath.Join(parts[1:]...)
		target := filepath.Join(targetPath, relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent dir for %s: %v", target, err)
			}
			
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %v", target, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %v", target, err)
			}
			outFile.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent dir for symlink %s: %v", target, err)
			}
			_ = os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				fmt.Printf("[!] Warning: Could not create symlink %s -> %s: %v\n", target, header.Linkname, err)
			}
		}
	}

	// Validate extraction (check if dir is not empty)
	files, _ := ioutil.ReadDir(targetPath)
	if len(files) == 0 {
		os.RemoveAll(targetPath)
		return fmt.Errorf("extraction succeeded but target directory is empty")
	}

	// Save hash for future skip
	if expectedHash != "" {
		ioutil.WriteFile(hashFile, []byte(expectedHash), 0644)
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status from manifest server: %s", resp.Status)
	}

	var manifest config.PlatformAssets
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func GetLatestTag(repo string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo), nil)
	if err != nil {
		return "", err
	}
	// Add user-agent to avoid 403 from GitHub API in some environments
	req.Header.Set("User-Agent", "gios-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func ValidateRemoteURL(url string) error {
	escapedURL := strings.ReplaceAll(url, " ", "%20")
	resp, err := http.Head(escapedURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote asset not found (HTTP %d)", resp.StatusCode)
	}
	return nil
}
