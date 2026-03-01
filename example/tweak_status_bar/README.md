# Tweak: Status Bar Heartbeat (Go inside SpringBoard)

A sophisticated template for building Cydia Tweaks using Go. This tweak injects itself into the SpringBoard process and runs a persistent background "heartbeat" to demonstrate Go's stability inside native iOS processes.

## 🚀 What it implements
- **Objective-C Bridging**: Uses `#cgo` to link against `Foundation` and `UIKit` frameworks.
- **C Constructor**: Implements `__attribute__((constructor))` to ensure the library initializes as soon as MobileSubstrate loads it.
- **Persistent Goroutine**: Starts a long-running Go routine (heartbeat) that doesn't block the main UI thread.
- **Legacy WebKit Support**: Configured to work on old iOS versions (like iOS 5.1.1) thanks to the GIOS transpiler.

## 🛠 Prerequisites
- **GIOS CLI**: Installed on your Mac.
- **iOS SDK**: Version 9.3 or similar (managed automatically by GIOS).
- **Jailbroken Device**: iPad/iPhone with MobileSubstrate or Substitute installed.

## 📖 How to Apply
1. **Navigate to the folder**:
   ```bash
   cd example/tweak_status_bar
   ```
2. **Deploy and Watch**:
   ```bash
   gios run --watch
   ```
   *This will build the .dylib, sign it with `ldid`, upload it to `/Library/MobileSubstrate/DynamicLibraries/`, and trigger a Respring.*
3. **Check the Logs**:
   Look at your terminal. You should see:
   - `[GIOS] Status Bar Tweak loaded inside SpringBoard` (C Level)
   - `[GIOS] Heartbeat #1 from SpringBoard...` (Go Level)

## 🔍 Technical Deep Dive

### Objective-C Headers in Go
To avoid compilation errors like `unknown type name 'NSString'`, we use a specific CGO configuration:
```c
#cgo CFLAGS: -x objective-c
#ifdef __OBJC__
#import <Foundation/Foundation.h>
#endif
```
This tells the compiler to treat the headers as Objective-C only when necessary, preventing the Go-generated C wrappers from failing.

### Extending this Tweak
This project is designed as a starting point. To perform real UI modifications (like changing the Carrier name), you can add `MSHookMessageEx` calls via CGO to intercept the `UIStatusBar` methods.

## 📦 File Structure
- `main.go`: The core logic (C bridge + Go goroutine).
- `gios.json`: Project configuration (targets SpringBoard).
- `filter.plist`: MobileSubstrate filter (specifies `com.apple.springboard`).

---
*Part of the GIOS Project - Building the future of iOS development with Go.*
