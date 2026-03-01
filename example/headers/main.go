package main

import (
	"fmt"
	"headers_example/headers" // Import the generated headers
)

/*
   TWEAK: PROJECT HEADERS EXAMPLE
   ------------------------------
   This is a complete GIOS project designed to show the use of generated Headers.
   
   The goal of this example is for YOU to generate your own headers.
   Just run: 'gios headers [ProcessName]' in the 'headers' folder.
*/

func main() {
	fmt.Println("[gios] Starting Example Tweak with Headers...")

	// 1. We use one of the classes generated in headers/headers.go
	sb := headers.SpringBoard{
		ObjCObject: headers.ObjCObject{Ptr: 0xDEADBEEF},
	}

	fmt.Printf("[gios] Class detected: SpringBoard at %v\n", sb.Self())
	
	// 2. We call a real system method mapped in Go
	fmt.Println("[gios] Requesting persistent cache state...")
	cache := sb.PersistentCache()
	fmt.Printf("[gios] Cache Pointer: %v\n", cache)

	fmt.Println("[gios] Tweak finished successfully.")
}
