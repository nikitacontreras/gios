package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Global Gios directory (~/.gios)
var GiosDir string

// Global Flags (managed by Cobra in pkg/cmd)
var (
	WatchFlag  bool
	UnsafeFlag bool
	OutFlag    string
	IPFlag     string
	SyslogFlag bool
)

// PlatformAssets represents the manifest of available SDKs and DDIs
type PlatformAssets struct {
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

// Config represents the gios.json structure
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

// LoadConfig loads the gios.json and exits on error
func LoadConfig() Config {
	conf, err := LoadConfigSafe()
	if err != nil {
		fmt.Println("Error: gios.json not found. Run 'gios init' first.")
		os.Exit(1)
	}
	return conf
}

// LoadConfigSafe attempts to load gios.json searching parent directories
func LoadConfigSafe() (Config, error) {
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

	// Override with flags
	if OutFlag != "" {
		if strings.Contains(OutFlag, "/") {
			conf.Output = filepath.Base(OutFlag)
			conf.Deploy.Path = OutFlag
		} else {
			conf.Output = OutFlag
			if conf.Deploy.Path != "" {
				dir := filepath.Dir(conf.Deploy.Path)
				conf.Deploy.Path = filepath.Join(dir, OutFlag)
			}
		}
	}

	if IPFlag != "" {
		conf.Deploy.IP = IPFlag
	}

	return conf, nil
}

func init() {
	home, err := os.UserHomeDir()
	if err == nil {
		GiosDir = filepath.Join(home, ".gios")
	}
}

// GetSSHArgs returns the common SSH arguments for target device
func (c Config) GetSSHArgs(extra ...string) []string {
	sshKeyPath := filepath.Join(GiosDir, "id_rsa")
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
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}
	if c.Deploy.USB {
		target = "root@127.0.0.1"
		args = append(args, "-p", "2222")
	}
	args = append(args, target)
	return append(args, extra...)
}

// GetSCPArgs returns the common SCP arguments for target device
func (c Config) GetSCPArgs() []string {
	sshKeyPath := filepath.Join(GiosDir, "id_rsa")
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
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}
	if c.Deploy.USB {
		args = append(args, "-P", "2222")
	}
	return args
}

// GetSCPTarget returns the SCP formatted target "root@ip:path"
func (c Config) GetSCPTarget(filePath string) string {
	target := "root@" + c.Deploy.IP
	if c.Deploy.USB {
		target = "root@127.0.0.1"
	}
	return target + ":" + filePath
}
