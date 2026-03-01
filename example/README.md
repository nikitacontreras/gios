# GIOS Project Examples

A comprehensive collection of Go-based projects for jailbroken iOS devices, ranging from simple CLI tools to advanced system tweaks and background services.

## Project Structure

```text
example/
├── 📂 hello_world/        Basic CLI entry point.
├── 📂 device_info/        Hardware & System info via Go stdlib.
├── 📂 web_debug_server/   Premium Web Dashboard & Remote Controls. (LaunchDaemon)
├── 📂 sys_notify_proxy/   Real-time Darwin Notification listener. (LaunchDaemon)
├── 📂 tweak_hello/        SpringBoard Tweak: Native Alert Popup. (MobileSubstrate)
└── 📂 tweak_status_bar/   SpringBoard Tweak: Pulse Heartbeat Logger. (MobileSubstrate)
```

## Quick Reference Table

| Example | Type | Key Features | Readme |
|:--- |:--- |:--- |:--- |
| **Hello World** | CLI Tool | Simple print output | [Explore](hello_world/README.md) |
| **Device Info** | CLI Tool | Kernel & Arch detection | [Explore](device_info/README.md) |
| **Web Debug** | Daemon | Web Dashboard, Alerts, Respring | [Explore](web_debug_server/README.md) |
| **Sys Notify** | Daemon | Intercept System Events (Darwin) | [Explore](sys_notify_proxy/README.md) |
| **Tweak Hello** | Tweak | Alert injection into SpringBoard | [Explore](tweak_hello/README.md) |
| **Status Bar** | Tweak | Persistent logging within SB | [Explore](tweak_status_bar/README.md) |

## Implementation Overview

- **MobileSubstrate Tweaks**: Located in `tweak_*` folders. They compile to `.dylib` and hook into existing iOS processes.
- **LaunchDaemons**: Located in `web_*` and `sys_*` folders. They run as persistent background services with root privileges.
- **CLI Utilities**: Standalone binaries for manual execution via SSH terminal.

For more information on how to build and deploy these, refer to the [Main GIOS Readme](../README.md).
