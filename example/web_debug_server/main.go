package main

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

// Forward declarations for CoreFoundation Alert symbols (missing in some SDKs)
typedef CFOptionFlags CFUserNotificationAlertLevel;
#define kCFUserNotificationNoteAlertLevel 1

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
) __attribute__((weak_import));

static void ShowNativeAlert(const char* title, const char* message) {
    if (CFUserNotificationDisplayAlert == NULL) return;

    CFStringRef cfTitle = CFStringCreateWithCString(NULL, title, kCFStringEncodingUTF8);
    CFStringRef cfMsg = CFStringCreateWithCString(NULL, message, kCFStringEncodingUTF8);

    // Arg 6 is Header, Arg 7 is Body Message.
    CFUserNotificationDisplayAlert(
        0, kCFUserNotificationNoteAlertLevel, NULL, NULL, NULL,
        cfTitle, cfMsg, CFSTR("OK"), NULL, NULL, NULL
    );

    if (cfTitle) CFRelease(cfTitle);
    if (cfMsg) CFRelease(cfMsg);
}
*/
import "C"

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unsafe"
)

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>GIOS | Dashboard</title>
    <style>
        body { 
            background-color: #0d0d10; 
            color: #e0e0e6; 
            font-family: "Helvetica Neue", Helvetica, Arial, sans-serif; 
            margin: 0; padding: 0;
            -webkit-text-size-adjust: none;
        }

        .container { padding: 20px; max-width: 960px; margin: 0 auto; }

        header { 
            border-bottom: 2px solid #2d2d35; 
            padding-bottom: 15px; 
            margin-bottom: 30px;
            overflow: hidden;
        }

        .logo { float: left; }
        .logo h1 { 
            margin: 0; font-size: 28px; font-weight: bold; 
            color: #00f2ff;
            background-image: -webkit-linear-gradient(top, #00f2ff, #7000ff);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .row { overflow: hidden; margin-bottom: 25px; clear: both; }
        .col { 
            width: 48%; 
            float: left; 
            margin-right: 2%;
            background: #16161a;
            -webkit-border-radius: 12px;
            -webkit-box-shadow: 0 4px 6px rgba(0,0,0,0.3);
            border: 1px solid #2d2d35;
        }
        .col.last { margin-right: 0; }
        .col.third { width: 31%; margin-right: 2%; }
        .col.third.last { width: 32%; margin-right: 0; }
        .col.full { width: 100%; margin-right: 0; }

        .card { padding: 15px; }
        .card h2 { 
            font-size: 11px; 
            color: #9494a0; 
            text-transform: uppercase; 
            margin: 0 0 12px 0;
            letter-spacing: 1.5px;
            font-weight: 800;
        }
        .card .val { font-size: 13px; font-weight: 600; word-wrap: break-word; color: #fff; }
        
        pre { 
            font-family: Courier, monospace; 
            font-size: 11px; 
            background: #000; 
            padding: 12px; 
            -webkit-border-radius: 8px; 
            overflow: auto;
            color: #00f2ff;
            border: 1px solid #1a1a1f;
            margin: 0;
        }

        /* Control Buttons & Inputs */
        .btn {
            display: inline-block;
            padding: 10px 15px;
            background: #25252b;
            border: 1px solid #3d3d45;
            color: #fff;
            text-decoration: none;
            font-size: 12px;
            font-weight: bold;
            -webkit-border-radius: 8px;
            margin-right: 5px;
            cursor: pointer;
        }
        .btn:hover { background: #35353b; border-color: #00f2ff; }
        .btn-primary { background: #00f2ff; color: #000; border: none; }
        .btn-danger { color: #ff3b30; border-color: #ff3b30; }

        input[type="text"] {
            background: #000;
            border: 1px solid #2d2d35;
            padding: 10px;
            color: #00f2ff;
            -webkit-border-radius: 8px;
            width: 70%;
            font-size: 13px;
            margin-bottom: 10px;
        }

        footer { text-align: center; color: #555; padding: 40px 0; font-size: 11px; }

        @media (max-width: 768px) {
            .col, .col.third { width: 100%; margin-right: 0; margin-bottom: 15px; }
            input[type="text"] { width: 100%; }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="logo">
                <h1>GIOS.</h1>
            </div>
        </header>

        <div class="row">
            <div class="col full">
                <div class="card">
                    <h2>Remote Alert System</h2>
                    <p style="font-size: 11px; color:#9494a0; margin-bottom:10px;">Push native popup notifications to the iPad screen.</p>
                    <form action="/alert" method="POST" style="margin:0">
                        <input type="text" name="msg" placeholder="Type message for iPad..." required>
                        <br>
                        <input type="submit" value="Push Alert to Screen" class="btn btn-primary">
                    </form>
                </div>
            </div>
        </div>

        <div class="row">
            <div class="col third">
                <div class="card">
                    <h2>Quick Actions</h2>
                    <form action="/action" method="POST" style="margin:0">
                        <input type="submit" name="cmd" value="Respring" class="btn">
                        <input type="submit" name="cmd" value="Reboot" class="btn btn-danger">
                    </form>
                </div>
            </div>
            <div class="col third">
                <div class="card">
                    <h2>System Uptime</h2>
                    <div class="val">{{.Uptime}}</div>
                </div>
            </div>
            <div class="col third last">
                <div class="card">
                    <h2>Kernel</h2>
                    <div class="val">{{.Kernel}}</div>
                </div>
            </div>
        </div>

        <div class="row">
            <div class="col full">
                <div class="card">
                    <h2>Hardware Usage (CPU & Processes)</h2>
                    <pre>{{.Processes}}</pre>
                </div>
            </div>
        </div>

        <div class="row">
            <div class="col">
                 <div class="card">
                    <h2>Storage (df -h)</h2>
                    <pre>{{.Disk}}</pre>
                </div>
            </div>
            <div class="col last">
                <div class="card">
                    <h2>Network Stats</h2>
                    <pre>{{.Network}}</pre>
                </div>
            </div>
        </div>

        <footer>
            Running on {{.Arch}} &bull; Gios Integrated &bull; {{.Time}}
        </footer>
    </div>
</body>
</html>
`

func logToFile(msg string) {
	f, _ := os.OpenFile("/var/log/webdebug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		fmt.Fprintf(f, "[%v] %s\n", time.Now().Format("15:04:05"), msg)
		f.Close()
	}
	fmt.Println(msg)
}

func execCmd(name string, arg ...string) string {
	out, err := exec.Command(name, arg...).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(out))
	}
	return string(out)
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Handle Alert sending
	if r.URL.Path == "/alert" && r.Method == "POST" {
		r.ParseForm()
		msg := r.FormValue("msg")
		logToFile("SENDING ALERT: " + msg)
		
		// DO NOT FREE STRINGS UNTIL CALL IS DONE.
		// Moving free inside goroutine to prevent memory being wiped before C reads it.
		go func(m string) {
			cTitle := C.CString("Remote Message")
			cMsg := C.CString(m)
			C.ShowNativeAlert(cTitle, cMsg)
			C.free(unsafe.Pointer(cTitle))
			C.free(unsafe.Pointer(cMsg))
		}(msg)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<script>window.location='/';</script>")
		return
	}

	// Handle Quick Actions
	if r.URL.Path == "/action" && r.Method == "POST" {
		r.ParseForm()
		cmd := r.FormValue("cmd")
		logToFile("ACTION: " + cmd)
		
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<script>window.location='/';</script>")
		
		go func() {
			time.Sleep(100 * time.Millisecond)
			if cmd == "Respring" {
				exec.Command("killall", "-9", "SpringBoard").Run()
			} else if cmd == "Reboot" {
				exec.Command("reboot").Run()
			}
		}()
		return
	}

	logToFile(fmt.Sprintf("Request from %s", r.RemoteAddr))

	res := htmlTemplate
	res = strings.Replace(res, "{{.Kernel}}", execCmd("uname", "-r"), 1)
	res = strings.Replace(res, "{{.Uptime}}", execCmd("uptime"), 1)
	res = strings.Replace(res, "{{.Disk}}", execCmd("df", "-h"), 1)
	res = strings.Replace(res, "{{.Network}}", execCmd("ifconfig"), 1)
	res = strings.Replace(res, "{{.Processes}}", execCmd("ps", "-arc", "-o", "%cpu,command", "-m", "10"), 1)
	res = strings.Replace(res, "{{.Time}}", time.Now().Format("15:04:05"), 1)
	res = strings.Replace(res, "{{.Arch}}", runtime.GOARCH, 1)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, res)
}

func main() {
	logToFile("Dashboard starting...")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
