package deploy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/danielpaulus/go-ios/ios/syslog"
	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/utils"
)

func RunLogs() {
	// Attempt to load config, but don't die if it fails and we just want --syslog
	conf, err := config.LoadConfigSafe()
	
	isSyslog := false
	for _, arg := range os.Args {
		if arg == "--syslog" {
			isSyslog = true
		}
	}

	if err != nil && !isSyslog {
		fmt.Printf("%sError: gios.json not found. Run 'gios init' first or use --syslog for USB-only logs.%s\n", utils.ColorRed, utils.ColorReset)
		return
	}

	if !isSyslog && !conf.Deploy.USB && conf.Deploy.IP == "" {
		fmt.Printf("%sError: Target Device IP not set in gios.json.%s\n", utils.ColorRed, utils.ColorReset)
		return
	}

	targetDisp := "Unknown"
	projectName := ""
	if err == nil {
		targetDisp = conf.Deploy.IP
		projectName = conf.Name
		if conf.Deploy.USB {
			targetDisp = "USB Device"
			if !EnsureUSBTunnelNative() {
				return
			}
		}
	}

	verbose := false
	for _, arg := range os.Args {
		if arg == "--verbose" || arg == "-v" {
			verbose = true
			break
		}
	}

	if isSyslog {
		fmt.Printf("%s[gios]%s Streaming native syslog via USB...\n", utils.ColorCyan, utils.ColorReset)
		device, err := GetFirstDevice()
		if err != nil {
			fmt.Printf("%s[!] Error:%s %v\n", utils.ColorRed, utils.ColorReset, err)
			return
		}
		
		conn, err := syslog.New(device)
		if err != nil {
			fmt.Printf("%s[!] Error:%s Failed to connect to syslog: %v\n", utils.ColorRed, utils.ColorReset, err)
			return
		}
		defer conn.Close()

		for {
			line, err := conn.ReadLogMessage()
			if err != nil {
				break
			}
			printLogLine(line, projectName)
		}
		return
	}

	fmt.Printf("%s[gios]%s Searching for active log agent on %s...\n", utils.ColorCyan, utils.ColorReset, targetDisp)

	// List of commands for logging.
	logCommands := []string{
		"tail -f /var/log/" + conf.PackageID + ".log",
		"tail -f /var/log/syslog",
		"tail -f /var/log/messages",
		"tail -f /var/log/webdebug.log",
		"log show --last 1m --follow --level debug",
		"tail -f /var/log/notifyd.log",
	}

	success := false
L:
	for i, logCmd := range logCommands {
		if verbose {
			fmt.Printf("[gios] Trying: %-35s ", utils.ColorBold+logCmd+utils.ColorReset)
		} else {
			fmt.Printf("\r[gios] Checking log agents: [%d/%d] ", i+1, len(logCommands))
		}
		
		sshArgs := conf.GetSSHArgs(logCmd)
		// Insert BatchMode and ConnectTimeout for the diagnostic phase
		sshArgs = append([]string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=3"}, sshArgs...)

		cmd := exec.Command("ssh", sshArgs...)
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			if verbose {
				fmt.Printf("%s[SSH FAIL]%s\n", utils.ColorRed, utils.ColorReset)
			}
			continue
		}

		outChan := make(chan string)
		errChan := make(chan bool)

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				outChan <- scanner.Text()
			}
			errChan <- true
		}()

		go func() {
			sc := bufio.NewScanner(stderr)
			for sc.Scan() {
				txt := strings.ToLower(sc.Text())
				if strings.Contains(txt, "no such file") || strings.Contains(txt, "not found") || strings.Contains(txt, "unrecognized") {
					errChan <- true
					return
				}
			}
		}()

		select {
		case line := <-outChan:
			success = true
			if !verbose {
				fmt.Printf("%sDONE!%s\n", utils.ColorGreen, utils.ColorReset)
			} else {
				fmt.Printf("%s[FOUND]%s\n", utils.ColorGreen, utils.ColorReset)
			}
			fmt.Println("--------------------------------------------------")
			printLogLine(line, conf.Name)
			for l := range outChan {
				printLogLine(l, conf.Name)
			}
			break L
		case <-errChan:
			if verbose {
				fmt.Printf("%s[MISSING]%s\n", utils.ColorRed, utils.ColorReset)
			}
			cmd.Process.Kill()
			cmd.Wait()
			time.Sleep(200 * time.Millisecond)
			continue
		case <-time.After(3 * time.Second):
			// Likely a valid but idle 'tail -f'
			success = true
			if !verbose {
				fmt.Printf("%sCONNECTED%s\n", utils.ColorGreen, utils.ColorReset)
			} else {
				fmt.Printf("%s[READY]%s (Idle)\n", utils.ColorGreen, utils.ColorReset)
			}
			fmt.Println("--------------------------------------------------")
			fmt.Println("(Waiting for new log entries...)")
			for l := range outChan {
				printLogLine(l, conf.Name)
			}
			break L
		}
	}

	if !success {
		if !verbose { fmt.Println() }
		fmt.Printf("\n%s[!] Error: No log agent could be established.%s\n", utils.ColorRed, utils.ColorReset)
		fmt.Println("This iPad doesn't seem to have a syslog daemon active.")
		fmt.Println("\nTo fix this:")
		fmt.Println(" 1. Open Cydia/Sileo on your iPad.")
		fmt.Println(" 2. Search for and install 'syslogd' (by deVbug).")
		fmt.Println(" 3. Restart the device or run 'launchctl load /Library/LaunchDaemons/com.apple.syslogd.plist'.")
	}
}

func printLogLine(line, projectName string) {
	lowerLine := strings.ToLower(line)
	if projectName != "" && strings.Contains(lowerLine, strings.ToLower(projectName)) {
		fmt.Printf("%s%s%s\n", utils.ColorBold+utils.ColorYellow, line, utils.ColorReset)
	} else if strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "fail") {
		fmt.Printf("%s%s%s\n", utils.ColorRed, line, utils.ColorReset)
	} else if strings.Contains(lowerLine, "warn") {
		fmt.Printf("%s%s%s\n", utils.ColorYellow, line, utils.ColorReset)
	} else {
		fmt.Println(line)
	}
}
