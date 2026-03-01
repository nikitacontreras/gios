package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SSHClient represents a managed SSH connection
type SSHClient struct {
	client *ssh.Client
	config Config
}

// NewSSHClient creates a new SSH connection using the project configuration (key-based)
func NewSSHClient(c Config) (*SSHClient, error) {
	sshKeyPath := filepath.Join(giosDir, "id_rsa")
	key, err := ioutil.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	host := c.Deploy.IP
	port := "22"
	if c.Deploy.USB {
		host = "127.0.0.1"
		port = "2222"
	}

	sshConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Config: ssh.Config{
			KeyExchanges: []string{
				"diffie-hellman-group1-sha1",
				"diffie-hellman-group14-sha1",
				"diffie-hellman-group-exchange-sha1",
				"diffie-hellman-group-exchange-sha256",
			},
			Ciphers: []string{
				"aes128-ctr", "aes192-ctr", "aes256-ctr",
				"aes128-cbc", "3des-cbc",
			},
		},
	}
	sshConfig.HostKeyAlgorithms = []string{
		ssh.KeyAlgoRSA,
		ssh.KeyAlgoED25519,
		ssh.KeyAlgoECDSA256,
	}

	client, err := ssh.Dial("tcp", host+":"+port, sshConfig)
	if err != nil {
		return nil, err
	}

	return &SSHClient{client: client, config: c}, nil
}

// NewSSHClientWithPassword creates a new SSH connection using a password
func NewSSHClientWithPassword(c Config, password string) (*SSHClient, error) {
	host := c.Deploy.IP
	port := "22"
	if c.Deploy.USB {
		host = "127.0.0.1"
		port = "2222"
	}

	sshConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Config: ssh.Config{
			KeyExchanges: []string{
				"diffie-hellman-group1-sha1",
				"diffie-hellman-group14-sha1",
				"diffie-hellman-group-exchange-sha1",
				"diffie-hellman-group-exchange-sha256",
			},
			Ciphers: []string{
				"aes128-ctr", "aes192-ctr", "aes256-ctr",
				"aes128-cbc", "3des-cbc",
			},
		},
	}
	sshConfig.HostKeyAlgorithms = []string{
		ssh.KeyAlgoRSA,
		ssh.KeyAlgoED25519,
		ssh.KeyAlgoECDSA256,
	}

	client, err := ssh.Dial("tcp", host+":"+port, sshConfig)
	if err != nil {
		return nil, err
	}

	return &SSHClient{client: client, config: c}, nil
}

// InstallKey appends a public key to the remote authorized_keys file
func (s *SSHClient) InstallKey(pubKeyPath string) error {
	pubKey, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		return err
	}

	remoteKey := strings.TrimSpace(string(pubKey))
	cmd := fmt.Sprintf("mkdir -p ~/.ssh && touch ~/.ssh/authorized_keys && chmod 700 ~/.ssh && chmod 600 ~/.ssh/authorized_keys && grep -qF '%s' ~/.ssh/authorized_keys || echo '%s' >> ~/.ssh/authorized_keys", remoteKey, remoteKey)
	_, err = s.Run(cmd)
	return err
}

// Run executes a command on the remote device
func (s *SSHClient) Run(command string) (string, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	out, err := session.CombinedOutput(command)
	return string(out), err
}

// Stream executes a command and streams its output to the provided writers
func (s *SSHClient) Stream(command string, stdout, stderr io.Writer) error {
	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = stdout
	session.Stderr = stderr
	return session.Run(command)
}

// Upload transfers a local file to a remote path
func (s *SSHClient) Upload(localPath, remotePath string) error {
	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C%04o %d %s\n", stat.Mode()&0777, stat.Size(), filepath.Base(remotePath))
		io.Copy(w, file)
		fmt.Fprint(w, "\x00")
	}()

	return session.Run("scp -t " + remotePath)
}
// Download transfers a remote file to a local path using 'cat' for maximum compatibility
func (s *SSHClient) Download(remotePath, localPath string) error {
	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// Using cat is more reliable on legacy iOS than scp protocol handshakes
	session.Stdout = localFile
	err = session.Run("cat \"" + remotePath + "\"")
	return err
}

// Close terminates the connection
func (s *SSHClient) Close() {
	if s.client != nil {
		s.client.Close()
	}
}
