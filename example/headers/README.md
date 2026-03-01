# 🔍 GIOS Headers: Native Reverse Engineering

This example demonstrates the **GIOS Headers** engine, a powerful automated reverse engineering tool that extracts Objective-C metadata from iDevice binaries and transforms them into native Go APIs.

## 🚀 Overview
The "Headers" feature eliminates the guesswork in iOS development. Instead of searching for class names and method signatures online, GIOS:
1.  **Locates** the binary directly on your connected iPad/iPhone.
2.  **Analyzes** the Mach-O structure (supporting both 32-bit `armv7` and 64-bit `arm64`).
3.  **Generates** a `headers.go` file containing Go-friendly wrappers for every discovered class and method.

## 🛠 How to use it

### 1. Generate the Headers
Navigate to this folder and run the following command targeting any system process (e.g., `SpringBoard`, `AccountSettings`, `BackBoardD`):
```bash
gios headers SpringBoard
```

**What happens next?**
- GIOS connects via SSH (using your stored keys).
- It downloads the binary metadata (optimized partial copy).
- It scans for classes like `SBIconController`, `SBApplication`, etc.
- It creates a **`headers.go`** file in your current directory.

### 2. Use them in your Go code
Once the headers are generated, you can interact with iOS classes as if they were native Go structs.

```go
package main

import "fmt"

func main() {
	// 1. Wrap a raw pointer from the iOS runtime
	// (In a real hook, you get this from the stack or gios hook)
	sb := SpringBoard{ 
		ObjCObject: ObjCObject{Ptr: 0xDEADBEEF},
	}

	fmt.Println("[gios] Requesting Persistent Cache...")
	
	// 2. Call methods using standard Go syntax
	// This method is defined automatically in headers.go
	cache := sb.PersistentCache()
	fmt.Printf("Cache Pointer: %v\n", cache)
	
	// 3. Perform system actions
	sb.RemoveCacheFiles()
}
```

## 💎 Key Features
- **Auto-Discovery**: Highlights the most relevant classes (Controllers, Managers, Icons) found in the binary.
- **CamelCase Mapping**: Automatically converts ObjC signatures like `initWithObjects:` into Go-compatible names like `InitWithObjects`.
- **Zero-Password**: Uses the built-in GIOS SSH client; no manual password entry required if you've run `gios connect`.
- **Project Aware**: If you run this inside a GIOS project (with a `gios.json`), it automatically uses `package main` so you can compile immediately.

---
> [!TIP]
> Use `gios headers SpringBoard` as your starting point for any tweak. Knowing exactly what methods are available is 90% of the work in reverse engineering.
