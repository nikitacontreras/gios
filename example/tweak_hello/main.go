package main

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>

// CFUserNotification definitions (manually declared for SDKs missing the header)
typedef CFOptionFlags CFUserNotificationLevel;
enum {
    kCFUserNotificationPlainAlertLevel = 0,
    kCFUserNotificationNoteAlertLevel = 1,
    kCFUserNotificationCautionAlertLevel = 2,
    kCFUserNotificationStopAlertLevel = 3
};

extern SInt32 CFUserNotificationDisplayAlert(
    CFTimeInterval timeout,
    CFOptionFlags flags,
    CFURLRef iconURL,
    CFURLRef soundURL,
    CFURLRef localizationURL,
    CFStringRef alertHeader,
    CFStringRef alertMessage,
    CFStringRef defaultButtonTitle,
    CFStringRef alternateButtonTitle,
    CFStringRef otherButtonTitle,
    CFOptionFlags *responseFlags
);

extern void TweakEntry();

__attribute__((constructor)) static void init_tweak() {
    TweakEntry();
}

static void show_alert(const char* title, const char* message) {
    CFStringRef titleRef = CFStringCreateWithCString(NULL, title, kCFStringEncodingUTF8);
    CFStringRef messageRef = CFStringCreateWithCString(NULL, message, kCFStringEncodingUTF8);
    
    CFUserNotificationDisplayAlert(
        0, 
        kCFUserNotificationPlainAlertLevel, 
        NULL, 
        NULL, 
        NULL, 
        titleRef, 
        messageRef, 
        CFSTR("Cool!"), 
        NULL, 
        NULL, 
        NULL
    );
    
    CFRelease(titleRef);
    CFRelease(messageRef);
}
*/
import "C"
import (
	"fmt"
	"time"
)

//export TweakEntry
func TweakEntry() {
	// Use a goroutine to wait for UI to be ready
	go func() {
		fmt.Println("[Gios/Tweak] Tweak loaded via C Constructor!")
		fmt.Println("[Gios/Tweak] Waiting 5s for UI...")
		time.Sleep(5 * time.Second)
		fmt.Println("[Gios/Tweak] Showing Alert now!")
		C.show_alert(C.CString("GIOS Tweak"), C.CString("Injected into SpringBoard successfully from Go! 🚀"))
	}()
}

func init() {}

func main() {}
