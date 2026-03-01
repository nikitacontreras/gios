# Web Diagnostic Dashboard

A powerful LaunchDaemon (Background Service) that provides a web-based dashboard for iOS diagnostics.

## What it implements
- **Premium UI**: Dark mode Dashboard optimized for legacy iOS 5 WebKit.
- **Diagnostics**: Real-time Kernel, Uptime, Memory (vm_stat), and Disk Usage (df -h).
- **Control Center**: Remote buttons for **Respring** and **Reboot**.
- **Remote Alerts**: Textbox to push native popup notifications to the iPad screen.
- **Process Manager**: List of the TOP 10 CPU-consuming processes.

## How to use
1. Navigate to this directory:
   ```bash
   cd example/web_debug_server
   ```
2. Build and install:
   ```bash
   gios install
   ```
3. Open your browser on your computer and go to: `http://<ipad_ip>:8080`

## Developer Notes
This example demonstrates how to build a persistent service (LaunchDaemon) in Go. It handles root privileges to execute system commands and provides a bridge between web inputs and native iOS APIs via CGO.
