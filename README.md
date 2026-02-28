# Gios CLI
**The Modern Build System for Legacy and Modern iOS Jailbreak Development**

Gios is an ultra-fast, cross-platform CLI tool completely written in Go. It empowers jailbreak developers and modders to build, package, and deploy Go-based tweaks, background services (LaunchDaemons), and utilities directly to legacy iOS devices (32-bit: iOS 5 to 10) or modern devices (64-bit Rootless environments) effortlessly.

No complex theos setup, zero-hassle compiler flags, and a modern CLI developer experience.

## ✨ Key Capabilities

- **Intelligent Orchestration**: Interactive initialization wizard for Legacy (armv7) or Modern Rootless (arm64) targets.
- **Auto SDK Retrieval**: If targeting legacy APIs, Gios will securely download, unpack, and route the required iOS (e.g., 9.3) `.tbd` SDK straight from the Theos mirror repositories. 
- **Automated Deployments**: Connect to your iDevice once to generate and register SSH keys. Subsequently deploy via persistent `ControlMaster` socket without typing a password again.
- **Native Packaging**: Easily compile and package native Cydia `.deb` packages with proper control files and metadata.
- **Daemon Mode**: Set `"daemon": true` in your configuration, and Gios will construct the Plist and install scripts to run your application in the background forever on the device.
- **Self-Updating**: Fetches the newest binary release from its GitHub repository via `gios update`.

---

## 🛠 Installation

Gios is built to be cross-platform out-of-the-box.

### macOS & Linux
```bash
git clone https://github.com/nikitacontreras/gios.git
cd gios
chmod +x install.sh
./install.sh
```

### Windows (PowerShell)
```powershell
git clone https://github.com/nikitacontreras/gios.git
cd gios
.\install.ps1
```
*(On Windows, you must have Go installed and accessible from your `$env:PATH`. On UNIX, you will additionally need `dpkg` installed (e.g. `brew install dpkg`) to use the local `.deb` generator).*

---

## 📖 Usage Guide

Creating an advanced tweak or backend for an iOS device has never been this declarative. All settings live in a clean `gios.json` file inside your project.

### 1. Initialize a new Project
Go to an empty folder and run:
```bash
gios init
```
An interactive wizard will ask you standard questions like package ID, targeted architecture (Legacy vs Modern Rootless), and whether it should run as a background service. It immediately provisions the Go module, the `gios.json` definition, and custom jailbreak `ents.plist` entitlements.

### 2. The Gios Configuration (`gios.json`)
The configuration is the heart of Gios. Here is an example of what the CLI builds for you:
```json
{
  "name": "my-gios-project",
  "package_id": "com.nikitastrike.miproyecto",
  "version": "1.0.0",
  "go_version": "go1.14.15",
  "sdk_version": "9.3",
  "arch": "armv7",
  "output": "out_bin",
  "main": "main.go",
  "entitlements": "ents.plist",
  "daemon": false,
  "deploy": {
    "ip": "192.168.1.50",
    "path": "/var/root/out_bin"
  }
}
```

### 3. Build the Application
Ensure that your configuration is right and type:
```bash
gios build
```
Gios will find or download the compiler sysroot, invoke the isolated toolchains (GVM/Native Go), cross-compile your `main.go`, and sign the executable using `ldid`.

### 4. Connect to Device
Avoid endless password logins by provisioning a fast, background SSH socket tunnel to your iPhone/iPad:
```bash
gios connect
# Passwords will only be requested once to inject the RSA keys.
```
*Note: Run `gios disconnect` to destroy the tunnel gracefully when your development session concludes.*

### 5. Install & Test Real-Time
To skip building manually, packaging, sending, and executing commands manually, merely type:
```bash
gios install
```
It runs `build`, creates the Cydia package internally, and pushes it through the rapid tunnel to the target machine while executing native `dpkg -i` on the device's shell.

If you just quickly want to test standard output to your local macOS terminal, run:
```bash
gios run --watch
```
Gios will bypass `.deb` creations, transfer the stark binary, and execute the application—piping all iOS device output to your desktop terminal.

### 6. Keep Gios Updated
As new versions enter the main repository branch, upgrading your global Gios CLI is immediate:
```bash
gios update
```
