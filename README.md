<div align="center">

<img src="assets/logo.svg" width="96" height="96" alt="GIOS Logo">

# GIOS
**The Modern Build System for Legacy and Modern iOS Jailbreak Development**

</div>

GIOS (Go on iOS) is an ultra-fast, cross-platform CLI tool completely written in Go. It empowers jailbreak developers and modders to build, package, and deploy Go-based tweaks, background services (LaunchDaemons), and utilities directly to legacy iOS devices (32-bit: iOS 5 to 10) or modern devices (64-bit Rootless environments) effortlessly.

No complex theos setup, zero-hassle compiler flags, and a modern CLI developer experience.

---

## ✨ Key Capabilities

- **Intelligent Orchestration**: Interactive initialization wizard for Legacy (armv7) or Modern Rootless (arm64) targets.
- **Auto SDK Retrieval**: Automatically downloads and routes required iOS SDKs (e.g., 9.3) from Theos mirrors.
- **Advanced Reverse Engineering**: Extract Objective-C classes/methods and generate Go wrappers instantly via `gios headers`.
- **USB & SSH Support**: Deploy via WiFi (SSH) or high-speed USB tunnel (usbmuxd). No SSH setup needed for core USB utilities.
- **Native Packaging**: Create Cydia-ready `.deb` packages with automated control file and metadata generation.
- **Legacy Transpiler**: Automatically patches modern Go code to run on historical iOS versions (32-bit).
- **Self-Updating**: Keep your toolchain fresh with `gios update`.

---

## 🛠 Installation

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
*(On Windows, ensure Go is in your PATH. On UNIX, `dpkg` is required for `.deb` generation).*

---

## 📖 Command Reference

### 🚀 Core Workflow
| Command | Description |
|:--- |:--- |
| `gios init` | Start the interactive wizard to create a new project. |
| `gios build` | Compile and sign the binary for your target architecture. |
| `gios run` | Build, sign, and execute the application on the device. |
| `gios package` | Generate a `.deb` package for distribution. |
| `gios install` | Full pipeline: Build → Package → Install on device. |
| `gios watch` | Enable Pro-mode: Auto-build and redeploy on every file save. |

### 🌐 Connectivity & Debugging
| Command | Description |
|:--- |:--- |
| `gios connect` | Establish a persistent SSH tunnel (ControlMaster) for passwordless access. |
| `gios shell` | Open an interactive SSH terminal directly on the iDevice. |
| `gios logs` | Stream live system logs (`syslog`) to your terminal. |
| `gios daemon` | Start the background bridge for high-performance remote execution. |
| `gios disconnect`| Gracefully close all active background tunnels. |

### 🛠 USB Utilities (No SSH Required)
| Command | Description |
|:--- |:--- |
| `gios info` | Display detailed hardware, battery, and system diagnostics via USB. |
| `gios screenshot`| Capture the device screen and save it to your local machine. |
| `gios reboot` | Perform a remote force-reboot of the connected device. |
| `gios mount` | Smart-detect and mount the required Developer Disk Image (DDI). |

### 🧪 Advanced RE & Tweaks
| Command | Description |
|:--- |:--- |
| `gios headers` | Extract ObjC headers from a process (e.g., `SpringBoard`) and generate Go APIs. |
| `gios hook` | Generate DSL-based hook boilerplates for SpringBoard modifications. |
| `gios polyfill` | Manually trigger the compatibility engine for legacy iOS targets. |

### 🩺 Maintenance
| Command | Description |
|:--- |:--- |
| `gios doctor` | Run a diagnostic check on your environment (SDKs, Go, toolchains). |
| `gios analyze` | Scan your code for symbols that might break on legacy iOS. |
| `gios diff` | View transpilation changes made to a specific file for 32-bit targets. |
| `gios sdk` | Manage local iOS SDKs (list, add, or prune unused versions). |
| `gios update` | Pull the latest version of GIOS from GitHub. |

---

## ⚙️ Configuration (`gios.json`)

The `gios.json` file is the brain of your project. Here's a complete example:

```json
{
  "name": "my-project",
  "package_id": "com.developer.project",
  "version": "1.0.0",
  "arch": "armv7",           // armv7 (Legacy) or arm64 (Modern/Rootless)
  "sdk_version": "9.3",      // Target iOS version
  "main": "main.go",
  "output": "my_binary",     // Name of the final executable
  "entitlements": "ents.plist",
  "daemon": false,           // Set to true to install as a LaunchDaemon
  "deploy": {
    "ip": "192.168.1.50",    // Device IP for SSH
    "path": "/var/root/",    // Remote deployment directory
    "usb": false             // Set to true to prioritize USB over WiFi
  }
}
```

---

## 🚩 Global Flags

- `--watch, -w`: Enable continuous development mode.
- `--unsafe`: Force-transpile `vendor/` dependencies (use with caution).
- `--out <path>`: Override the output binary name/path.
- `--ip <ip>`: Temporarily target a different device IP.
- `--syslog`: Use native USB streaming for logs (requires `idevicesyslog`).
- `--v, --version`: Show the current GIOS version.

---

## 🌟 Examples

Jumpstart your development with these ready-to-use templates in the `example/` directory:

| Type | Project | Description |
|:--- |:--- |:--- |
| **CLI** | [Hello World](example/hello_world/README.md) | Basic Go execution on iOS. |
| **CLI**| [Device Info](example/device_info/README.md) | Deep system diagnostics. |
| **Daemon** | [Web Server](example/web_debug_server/README.md) | Premium diagnostics dashboard. |
| **Advanced**| [Headers RE](example/headers/README.md) | Auto-generate Go APIs from iOS binaries. |
| **Tweak** | [Hello Tweak](example/tweak_hello/README.md) | Inject code into SpringBoard (armv7). |

