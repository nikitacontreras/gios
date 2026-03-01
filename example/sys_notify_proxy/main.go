package main

/*
#include <notify.h>
#include <stdio.h>
#include <stdlib.h>

// Forward declaration for Go export
extern void handleNotification(const char* name);

static void register_hook(const char* name) {
    int token;
    uint32_t status = notify_register_dispatch(name, &token, dispatch_get_main_queue(), ^(int t) {
        handleNotification(name);
    });
    if (status != NOTIFY_STATUS_OK) {
        printf("Failed to register %s\n", name);
    }
}
*/
import "C"
import (
	"fmt"
	"time"
)

//export handleNotification
func handleNotification(name *C.char) {
	goName := C.GoString(name)
	fmt.Printf("[Gios/Notify] [ %v ] Detected System Event: %s\n", time.Now().Format("15:04:05"), goName)
}

func main() {
	fmt.Println("[Gios/NotifyProxy] Starting System-wide Event Listener...")

	// Common iOS Notifications (some vary by iOS version)
	notifications := []string{
		"com.apple.springboard.hasBlankedScreen",
		"com.apple.springboard.lockstate",
		"com.apple.springboard.ringerstate",
		"com.apple.system.config.network_change",
	}

	for _, n := range notifications {
		C.register_hook(C.CString(n))
		fmt.Printf("[Gios/NotifyProxy] Registered for: %s\n", n)
	}

	// Keep alive (this is a daemon)
	select {}
}
