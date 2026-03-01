package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type DaemonRequest struct {
	Command string `json:"command"` // "exec" or "upload"
	Payload string `json:"payload"` // command string or local path
	Remote  string `json:"remote"`  // remote path for upload
}

type DaemonResponse struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

func runDaemon() {
	conf := loadConfig()
	if (len(os.Args) >= 3 && strings.ToLower(os.Args[2]) == "usb") || conf.Deploy.USB {
		conf.Deploy.USB = true
		conf.Deploy.IP = "127.0.0.1"
	}

	targetDisp := conf.Deploy.IP
	if conf.Deploy.USB {
		targetDisp = "USB Device (127.0.0.1:2222)"
		ensureUSBTunnel(conf)
	}

	fmt.Printf("[gios] Starting native SSH daemon for %s...\n", targetDisp)

	client, err := NewSSHClient(conf)
	if err != nil {
		fmt.Printf("[!] Daemon failed to connect: %v\n", err)
		return
	}
	defer client.Close()

	socketPath := filepath.Join(giosDir, "gios.sock")
	os.Remove(socketPath)

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Printf("[!] Daemon failed to listen on socket %s: %v\n", socketPath, err)
		return
	}
	defer l.Close()
	os.Chmod(socketPath, 0700)

	fmt.Println("[+] Daemon ready. Listening for CLI commands on " + socketPath)
	fmt.Println("    (This is a background service. Use 'gios run' or 'gios logs' in another terminal)")

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go handleDaemonConn(conn, client)
	}
}

func handleDaemonConn(conn net.Conn, client *SSHClient) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	var req DaemonRequest
	if err := decoder.Decode(&req); err != nil {
		return
	}

	var resp DaemonResponse
	switch req.Command {
	case "exec":
		out, err := client.Run(req.Payload)
		resp.Output = out
		if err != nil {
			resp.Error = err.Error()
		}
	case "upload":
		err := client.Upload(req.Payload, req.Remote)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Output = "Upload successful"
		}
	}

	json.NewEncoder(conn).Encode(resp)
}

func callDaemon(req DaemonRequest) (*DaemonResponse, error) {
	socketPath := filepath.Join(giosDir, "gios.sock")
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, err
	}

	var resp DaemonResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
