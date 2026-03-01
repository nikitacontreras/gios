package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func runLogs() {
	conf := loadConfig()
	if !conf.Deploy.USB && conf.Deploy.IP == "" {
		fmt.Printf("%sError: Target Device IP not set in gios.json.%s\n", ColorRed, ColorReset)
		return
	}

	targetDisp := conf.Deploy.IP
	if conf.Deploy.USB {
		targetDisp = "USB Device"
		if !ensureUSBTunnel(conf) {
			return
		}
	}

	verbose := false
	for _, arg := range os.Args {
		if arg == "--verbose" || arg == "-v" {
			verbose = true
			break
		}
	}

	fmt.Printf("%s[gios]%s Searching for active log agent on %s...\n", ColorCyan, ColorReset, targetDisp)

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
			fmt.Printf("[gios] Trying: %-35s ", ColorBold+logCmd+ColorReset)
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
				fmt.Printf("%s[SSH FAIL]%s\n", ColorRed, ColorReset)
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
				fmt.Printf("%sDONE!%s\n", ColorGreen, ColorReset)
			} else {
				fmt.Printf("%s[FOUND]%s\n", ColorGreen, ColorReset)
			}
			fmt.Println("--------------------------------------------------")
			printLogLine(line, conf.Name)
			for l := range outChan {
				printLogLine(l, conf.Name)
			}
			break L
		case <-errChan:
			if verbose {
				fmt.Printf("%s[MISSING]%s\n", ColorRed, ColorReset)
			}
			cmd.Process.Kill()
			cmd.Wait()
			time.Sleep(200 * time.Millisecond)
			continue
		case <-time.After(3 * time.Second):
			// Likely a valid but idle 'tail -f'
			success = true
			if !verbose {
				fmt.Printf("%sCONNECTED%s\n", ColorGreen, ColorReset)
			} else {
				fmt.Printf("%s[READY]%s (Idle)\n", ColorGreen, ColorReset)
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
		fmt.Printf("\n%s[!] Error: No log agent could be established.%s\n", ColorRed, ColorReset)
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
		fmt.Printf("%s%s%s\n", ColorBold+ColorYellow, line, ColorReset)
	} else if strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "fail") {
		fmt.Printf("%s%s%s\n", ColorRed, line, ColorReset)
	} else if strings.Contains(lowerLine, "warn") {
		fmt.Printf("%s%s%s\n", ColorYellow, line, ColorReset)
	} else {
		fmt.Println(line)
	}
}
