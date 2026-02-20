package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Session struct {
	client   *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
}

func Connect(h Host, password string, skipKeyIfNotDeployed bool) (*Session, error) {
	config, err := buildSSHConfig(h, password, skipKeyIfNotDeployed)
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", h.Hostname, h.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm-256color"
	}

	if err := session.RequestPty(term, 80, 24, modes); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to request pty: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get stdin: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get stderr: %w", err)
	}

	if err := session.Shell(); err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	return &Session{
		client:   client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

func (s *Session) Resize(width, height int) error {
	return s.session.WindowChange(height, width)
}

func (s *Session) Write(data []byte) (int, error) {
	return s.stdin.Write(data)
}

func (s *Session) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

func (s *Session) ReadError(p []byte) (int, error) {
	return s.stderr.Read(p)
}

func (s *Session) Close() error {
	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.session != nil {
		s.session.Close()
	}
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

func buildSSHConfig(h Host, password string, skipKeyIfNotDeployed bool) (*ssh.ClientConfig, error) {
	home, _ := os.UserHomeDir()
	expandPath := func(p string) string {
		if strings.HasPrefix(p, "~") {
			return filepath.Join(home, p[1:])
		}
		return p
	}

	var authMethods []ssh.AuthMethod

	// If key is deployed, prioritize it; otherwise try password first
	keyAdded := false
	if h.IdentityFile != "" && !skipKeyIfNotDeployed {
		keyPath := expandPath(h.IdentityFile)
		key, err := os.ReadFile(keyPath)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				keyAdded = true
			}
		}
	}

	// Add password as fallback if key wasn't added, or as primary if no key
	if password != "" {
		if keyAdded {
			// Key first, then password as fallback
			authMethods = append(authMethods, ssh.Password(password))
		} else {
			// Password first if no key
			authMethods = append([]ssh.AuthMethod{ssh.Password(password)}, authMethods...)
		}
	}

	if len(authMethods) == 0 {
		keyPath := filepath.Join(home, ".ssh", "id_rsa")
		if key, err := os.ReadFile(keyPath); err == nil {
			if signer, err := ssh.ParsePrivateKey(key); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	if len(authMethods) == 0 {
		keyPath := filepath.Join(home, ".ssh", "id_ed25519")
		if key, err := os.ReadFile(keyPath); err == nil {
			if signer, err := ssh.ParsePrivateKey(key); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method available")
	}

	config := &ssh.ClientConfig{
		User:            h.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	return config, nil
}

type Host struct {
	Name         string
	Hostname     string
	User         string
	Port         int
	IdentityFile string
}
