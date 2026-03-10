package builder

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/sdk"
	"github.com/nikitastrike/gios/pkg/transpiler"
	"github.com/nikitastrike/gios/pkg/utils"
)

func Build(unsafeFlag bool) {
	conf := config.LoadConfig()
	cwd, _ := os.Getwd()

	fmt.Printf("[gios] Project: %s\n", conf.Name)
	fmt.Printf("[gios] Arch: %s, SDK: %s\n", conf.Arch, conf.SDKVersion)

	home, _ := os.UserHomeDir()
	var goBin string

	// Toolchain routing
	if conf.GoVersion == "local" || conf.GoVersion == "system" {
		goBin, _ = exec.LookPath("go")
		if goBin == "" {
			goBin = "go"
		}
	} else {
		goBin = filepath.Join(home, ".gvm", "gos", conf.GoVersion, "bin", "go")
		if _, err := os.Stat(goBin); os.IsNotExist(err) {
			fmt.Printf("[gios] Error: Go not found at %s. Did you use gvm to install it?\n", goBin)
			os.Exit(1)
		}
	}

	var envOS, envArch, envArm, cc, sdkPath string

	if conf.Arch == "armv7" {
		envOS = "darwin"
		envArch = "arm"
		envArm = "7"
		sdkPath = filepath.Join(config.GiosDir, "sdks", "iPhoneOS"+conf.SDKVersion+".sdk")
		if _, err := os.Stat(sdkPath); os.IsNotExist(err) {
			fmt.Printf("[gios] SDK not found at %s. Attempting to download...\n", sdkPath)
			if err := sdk.EnsureSDK(conf.SDKVersion, sdkPath); err != nil {
				fmt.Printf("[!] Error installing SDK: %v\n", err)
				os.Exit(1)
			}
		}
		cc = EnsureWrapper(sdkPath)
		fmt.Println("[gios] Legacy 32-bit Target Detected.")

		if unsafeFlag {
			fmt.Println("[gios] [Transpiler] WARNING: --unsafe flag active. Transpiling 'vendor' third-party dependencies.")
		}

		if err := transpiler.TranspileLegacy(cwd, unsafeFlag); err != nil {
			fmt.Println("[!] Transpiler Error:", err)
		}
		
	} else if conf.Arch == "arm64" {
		envOS = "ios"
		envArch = "arm64"
		envArm = ""
		cc = ""
		sdkPath = ""
	}

	cgoState := "1"
	var giosLibDir string
	if conf.Arch == "armv7" {
		giosLibDir = EnsureShims(sdkPath, cc)
	}

	var ldflags string
	if conf.Arch == "armv7" {
		ldflags = fmt.Sprintf("-s -w -extld=%s \"-extldflags=-L%s -lgios_libc\"", cc, giosLibDir)
	} else {
		ldflags = "-s -w"
	}

	cmdEnv := append(os.Environ(),
		"CGO_ENABLED="+cgoState,
		"GOOS="+envOS,
		"GOARCH="+envArch,
		"CGO_CFLAGS_ALLOW=-fobjc-exceptions",
	)
	if conf.Arch == "armv7" {
		cmdEnv = append(cmdEnv, "CGO_LDFLAGS=-L"+giosLibDir+" -lgios_libc")
	}
	if envArm != "" {
		cmdEnv = append(cmdEnv, "GOARM="+envArm)
	}
	if cc != "" {
		cmdEnv = append(cmdEnv, "CC="+cc)
	}
	if sdkPath != "" {
		cmdEnv = append(cmdEnv, "GIOS_SDK_PATH="+sdkPath)
	}

	isDylib := strings.HasSuffix(conf.Output, ".dylib")
	
	if isDylib && conf.Arch == "armv7" {
		tempA := "lib_gios_tmp.a"
		tempH := "lib_gios_tmp.h"
		buildArgs := []string{"build", "-trimpath", "-buildmode=c-archive", "-o", tempA, conf.Main}
		if _, err := os.Stat(filepath.Join(cwd, "vendor")); err == nil {
			buildArgs = append(buildArgs, "-mod=vendor")
		}
		
		cmd := exec.Command(goBin, buildArgs...)
		cmd.Dir = cwd
		cmd.Env = cmdEnv
		utils.DrawProgress("Compiling Archive", 30)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("\nGo Archive Error:\n%s\n", string(out))
			os.Exit(1)
		}
		
		utils.DrawProgress("Linking Dynamic Lib", 60)
		linkArgs := []string{"-shared", "-dynamiclib", "-o", conf.Output, "-Wl,-all_load", tempA, "-L" + giosLibDir, "-lgios_libc", "-lobjc", "-framework", "CoreFoundation", "-framework", "Foundation", "-framework", "UIKit"}
		linkCmd := exec.Command(cc, linkArgs...)
		linkCmd.Env = cmdEnv
		if out, err := linkCmd.CombinedOutput(); err != nil {
			fmt.Printf("\nLinker Error:\n%s\n", string(out))
			os.Exit(1)
		}
		os.Remove(tempA)
		os.Remove(tempH)
		utils.DrawProgress("Linking", 80)

	} else {
		buildArgs := []string{"build", "-trimpath", "-ldflags=" + ldflags}
		if isDylib {
			buildArgs = append(buildArgs, "-buildmode=c-shared")
		}
		if _, err := os.Stat(filepath.Join(cwd, "vendor")); err == nil {
			buildArgs = append(buildArgs, "-mod=vendor")
		}
		buildArgs = append(buildArgs, "-o", conf.Output, conf.Main)

		cmd := exec.Command(goBin, buildArgs...)
		cmd.Dir = cwd
		cmd.Env = cmdEnv

		utils.DrawProgress("Compiling", 30)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("\nCompilation error:\n%s\n", string(out))
			os.Exit(1)
		}
		utils.DrawProgress("Compiling", 70)
	}

	if _, err := exec.LookPath("ldid"); err == nil {
		utils.DrawProgress("Signing", 85)
		var signCmd *exec.Cmd
		if conf.Entitlements != "" && conf.Entitlements != "none" {
			signCmd = exec.Command("ldid", "-S"+conf.Entitlements, conf.Output)
		} else {
			signCmd = exec.Command("ldid", "-S", conf.Output)
		}
		signCmd.CombinedOutput()
	}
	utils.DrawProgress("Ready!", 100)
}

func EnsureWrapper(sdkPath string) string {
	err := os.MkdirAll(config.GiosDir, 0755)
	if err != nil {
		fmt.Printf("Error creating .gios directory: %v\n", err)
		os.Exit(1)
	}

	wrapperPath := filepath.Join(config.GiosDir, "arm-clang.sh")
	content := `#!/bin/bash
if [ -z "$GIOS_SDK_PATH" ]; then
    echo "GIOS_SDK_PATH is not configured."
    exit 1
fi
exec clang -target armv7-apple-ios5.0 -marm -march=armv7-a -mfpu=vfpv3-d16 \
     -isysroot "$GIOS_SDK_PATH" \
     -Wno-unused-command-line-argument \
     -Wno-incompatible-sysroot \
     -Wno-error=incompatible-sysroot \
     -fno-asynchronous-unwind-tables \
     "$@"
`
	err = ioutil.WriteFile(wrapperPath, []byte(content), 0755)
	if err != nil {
		fmt.Printf("Error writing wrapper: %v\n", err)
		os.Exit(1)
	}
	return wrapperPath
}

func EnsureShims(sdkPath, wrapperPath string) string {
	libDir := filepath.Join(config.GiosDir, "lib")
	os.MkdirAll(libDir, 0755)

	shimC := filepath.Join(libDir, "shims.c")
	shimO := filepath.Join(libDir, "shims.o")
	shimA := filepath.Join(libDir, "libgios_libc.a")

	content := `#include <stddef.h>
#include <dirent.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/stat.h>
#include <errno.h>

void *memmove(void *dest, const void *src, size_t n) {
    unsigned char *d = (unsigned char *)dest;
    const unsigned char *s = (const unsigned char *)src;
    if (d < s) {
        while (n--) *d++ = *s++;
    } else {
        d += n; s += n;
        while (n--) *--d = *--s;
    }
    return dest;
}

void *memcpy(void *dest, const void *src, size_t n) {
    return memmove(dest, src, n);
}

void *memset(void *s, int c, size_t n) {
    unsigned char *p = (unsigned char *)s;
    while (n--) *p++ = (unsigned char)c;
    return s;
}

int memcmp(const void *s1, const void *s2, size_t n) {
    const unsigned char *p1 = (const unsigned char *)s1;
    const unsigned char *p2 = (const unsigned char *)s2;
    while (n--) {
        if (*p1 != *p2) return (int)(*p1 - *p2);
        p1++; p2++;
    }
    return 0;
}

void memset_pattern16(void *b, const void *pattern16, size_t len) {
    unsigned char *dest = (unsigned char *)b;
    const unsigned char *pat = (const unsigned char *)pattern16;
    while (len >= 16) {
        for(int i=0; i<16; i++) dest[i] = pat[i];
        dest += 16; len -= 16;
    }
    for(size_t i=0; i<len; i++) dest[i] = pat[i];
}

void *__memcpy_chk(void *dest, const void *src, size_t len, size_t destlen) { return memcpy(dest, src, len); }
void *__memset_chk(void *dest, int c, size_t len, size_t destlen) { return memset(dest, c, len); }
void *__memmove_chk(void *dest, const void *src, size_t len, size_t destlen) { return memmove(dest, src, len); }

DIR *fdopendir(int fd) {
    char path[1024];
    if (fcntl(fd, F_GETPATH, path) != -1) {
        return opendir(path);
    }
    errno = ENOSYS;
    return NULL;
}
`
	ioutil.WriteFile(shimC, []byte(content), 0644)
	
	cmd := exec.Command(wrapperPath, "-Os", "-c", shimC, "-o", shimO)
	cmd.Env = append(os.Environ(), "GIOS_SDK_PATH="+sdkPath)
	cmd.Run()
	
	exec.Command("ar", "rcs", shimA, shimO).Run()
	return libDir
}
