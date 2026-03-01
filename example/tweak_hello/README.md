# Tweak: Hello Alert

A Cydia Tweak written in Go that displays a native system alert when SpringBoard loads.

## What it implements
- **Injection**: Targets `com.apple.springboard` using MobileSubstrate.
- **CGO Bridge**: Calls `CFUserNotificationDisplayAlert` from `CoreFoundation`.
- **C Constructor**: Uses `__attribute__((constructor))` to ensure Go code runs immediately upon library load.
- **Goroutines**: Runs the alert logic in a separate thread to avoid blocking the main UI thread during boot.

## How to Apply
1. Navigate to this directory:
   ```bash
   cd example/tweak_hello
   ```
2. Build and install:
   ```bash
   gios install
   ```
3. The device will automatically Respring. Wait 5 seconds for the alert to appear on your iPad.

## Design Pattern
This template shows the canonical way to write a Go tweak:
- Use `filter.plist` to define targets.
- Use `//export TweakEntry` for the entry point.
- Link against `CoreFoundation` and other frameworks via `#cgo LDFLAGS`.
