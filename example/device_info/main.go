package main

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"syscall"
)

/*
#include <sys/types.h>
#include <sys/sysctl.h>
#include <stdlib.h>

char* get_sysctl(char *name) {
    size_t size;
    sysctlbyname(name, NULL, &size, NULL, 0);
    char *value = malloc(size);
    sysctlbyname(name, value, &size, NULL, 0);
    return value;
}

uint64_t get_sysctl_uint64(char *name) {
    uint64_t value = 0;
    size_t size = sizeof(value);
    sysctlbyname(name, &value, &size, NULL, 0);
    return value;
}
*/
import "C"

func main() {
	fmt.Println("========================================")
	fmt.Println("      ADVANCED iDEVICE REPORT (CGO)     ")
	fmt.Println("========================================")

	// Hardware Info via sysctl
	model := C.GoString(C.get_sysctl(C.CString("hw.model")))
	machine := C.GoString(C.get_sysctl(C.CString("hw.machine")))
	mem := uint64(C.get_sysctl_uint64(C.CString("hw.memsize")))
	
	fmt.Printf("Device Model: %s (%s)\n", model, machine)
	fmt.Printf("Total RAM:    %d MB\n", mem/1024/1024)

	// OS Info
	osType := C.GoString(C.get_sysctl(C.CString("kern.ostype")))
	osRelease := C.GoString(C.get_sysctl(C.CString("kern.osrelease")))
	osVersion := C.GoString(C.get_sysctl(C.CString("kern.version")))

	fmt.Printf("Kernel:       %s %s\n", osType, osRelease)
	fmt.Printf("Build:        %s\n", osVersion)

	// Runtime Info
	fmt.Printf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Go Version:   %s\n", runtime.Version())
	fmt.Printf("CPUs:         %d\n", runtime.NumCPU())

	// Hostname
	hostname, _ := os.Hostname()
	fmt.Printf("Hostname:     %s\n", hostname)

	// User info
	u, err := user.Current()
	if err == nil {
		fmt.Printf("Current User: %s (UID: %s)\n", u.Username, u.Uid)
		fmt.Printf("Home Dir:     %s\n", u.HomeDir)
	}

	// Disk info (root)
	var stat syscall.Statfs_t
	err = syscall.Statfs("/", &stat)
	if err == nil {
		free := stat.Bfree * uint64(stat.Bsize)
		total := stat.Blocks * uint64(stat.Bsize)
		fmt.Printf("Disk Free:    %d MB / %d MB\n", free/1024/1024, total/1024/1024)
	}

	// Environment
	fmt.Println("\nRelevant Environment Variables:")
	for _, env := range []string{"USER", "SHELL", "PATH", "DYLD_LIBRARY_PATH"} {
		if val := os.Getenv(env); val != "" {
			fmt.Printf("  %s: %s\n", env, val)
		}
	}

	fmt.Println("========================================")
}
