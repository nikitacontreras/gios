package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
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
	fmt.Printf("[gios] Starting native SSH daemon for %s...\n", conf.Deploy.IP)

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
		fmt.Printf("[!] Daemon failed to listen: %v\n", err)
		return
	}
	defer l.Close()
	os.Chmod(socketPath, 0700)

	fmt.Println("[+] Daemon ready. Listening for commands...")

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
