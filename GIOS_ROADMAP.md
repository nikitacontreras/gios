# 🚀 GIOS Improvement Roadmap

This document outlines the strategic plan to evolve **GIOS** into a professional-grade toolchain for iOS jailbreak development.

---

## 🏗 Phase 1: Architecture Refactoring
**Goal:** Modularize the codebase to improve maintainability and scalability.

### 1.1. CLI Framework Migration
- [x] Replace custom flag parsing in `main.go` with **Cobra**.
- [x] Move command definitions to `cmd/gios/`.
- [x] Create a consistent logging/output system with `spf13/pterm` or similar.

### 1.2. Internal Packaging
- [x] Create `pkg/sdk`: Logic for downloading, listing, and managing iOS SDKs.
- [x] Create `pkg/transpiler`: Move all transpilation logic (AST rewriting, polyfills) here.
- [x] Create `pkg/builder`: Logic for calling `go build`, signing binaries, and packaging `.deb`.
- [x] Create `pkg/deploy`: Logic for SSH, USB (iproxy), and tunnel management.
- [x] Create `pkg/config`: Core configuration structures and `gios.json` handling.

---

## 🧪 Phase 2: Transpiler Hardening
**Goal:** Make the "backporting" engine robust enough to handle modern Go (1.21-1.25) reliably on iOS 5-10.

### 2.1. Pure AST Rewriting
- [x] Eliminate Regex-based replacements for generics and arrays.
- [x] Use `ast.Inspect` to detect and safely strip/rewrite generic types (`K, V any` -> `interface{}`).
- [ ] Implement a pre-check to ensure the code doesn't use unsupported low-level runtime features.

### 2.2. Functional Polyfills
- [x] Expand `pkg/polyfills` to include real implementations of:
    - `slices.Contains`, `slices.Sort`.
    - `maps.Keys`, `maps.Values`.
    - `cmp.Compare`, `cmp.Less`.
- [ ] Improve `ioutil` automatic conversion (handling imports correctly in all cases).

---

## 🌐 Phase 3: Connectivity & USB
**Goal:** Remove external dependencies like `iproxy` and improve "plug-and-play" experience.

### 3.1. Native USB Tunneling (Universal)
- [x] Integrated a native **USBMuxd** client in Go (via go-ios).
- [x] Removed dependencies on `iproxy` and `libimobiledevice` for all platforms.

### 3.2. Automated DDI Mounting
- [x] Add `gios mount` logic that automatically detects device version.
- [x] Download the required *Developer Disk Image* (DDI) from public mirrors if missing.
- [x] Mount the DDI via USB to enable debugging and advanced diagnostics.

---

## ✨ Phase 4: Developer Experience (DX)
**Goal:** Make the tool intuitive and self-healing.

### 4.1. Interactive CLI
- [x] Implement `gios init` using a rich interactive library (e.g., **Survey**).
- [x] Add visual progress bars for SDK downloads and large file transfers.

### 4.2. Self-Healing "Doctor"
- [x] Enhance `gios doctor` to detect missing system dependencies (dpkg, Go, SDKs).
- [x] Add `--fix` flag to automatically resolve common environment issues.

### 4.3. Global Configuration
- [x] Support `~/.gios/config.json` for global settings (default IP, developer name, favorite SDK).

---

## 🔍 Phase 5: Advanced Reverse Engineering
**Goal:** Deep integration with Objective-C for powerful tweak development.

### 5.1. Typed Go Headers
- [ ] Parse `__objc_methtype` to identify argument and return types.
- [ ] Generate `headers.go` with specific types (e.g., `NSString`, `int64`) instead of generic `uintptr`.
- [ ] Add helper functions to convert between Go and Objective-C types (e.g., `GoStrToNSString`).

### 5.2. Tweak Boilerplate Engine
- [ ] Add `gios template tweak <Name>` to create a full CydiaSubstrate/Ellekit project structure.
- [x] **Pure Go Debian Packager**: Abolish `dpkg-deb` dependency for Windows/Linux.

---

## ⚡ Phase 6: Performance Optimization
**Goal:** Instant builds for a "Go-like" development speed.

### 6.1. Transpilation Cache
- [x] Implement file hashing to skip transpiling unchanged `.go` files.
- [x] Store processed files in `.gios/cache/`.

### 6.2. Parallel Execution
- [x] Use goroutines to transpile multiple files in parallel during a `build` or `run` command.

---

## 🛠 Next Steps
1. **Move `main.go` logic into packages.**
2. **Setup Cobra for command management.**
3. **Refine the Transpiler to be AST-only.**
