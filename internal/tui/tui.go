package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shravan20/vecna/internal/config"
	"github.com/shravan20/vecna/internal/ssh"
)

type View int

const (
	ViewHome View = iota
	ViewAddHost
	ViewSSH
)

type Model struct {
	view         View
	width        int
	height       int
	keys         KeyMap
	cursor       int
	hosts        []config.Host
	inputs       []textinput.Model
	inputFocus   int
	err          error
	sshSession   *ssh.Session
	sshOutput    strings.Builder
	sshHost      *config.Host
	connecting   bool
	toast        string
	toastTimer   int
	toastSuccess bool
	sshLog       []string
	showPassword bool
}

func New() Model {
	return Model{
		view:  ViewHome,
		keys:  DefaultKeyMap(),
		hosts: config.GetHosts(),
	}
}

func (m *Model) initAddHostInputs() {
	m.inputs = make([]textinput.Model, 6)

	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].Prompt = ""
		m.inputs[i].CharLimit = 256
		m.inputs[i].Width = 40
		if i == 4 {
			m.inputs[i].EchoMode = textinput.EchoPassword
			m.inputs[i].EchoCharacter = '•'
		}
	}

	m.inputs[0].Placeholder = "e.g. prod-server"
	m.inputs[1].Placeholder = "192.168.1.100 or host.example.com"
	m.inputs[2].Placeholder = "root"
	m.inputs[3].Placeholder = "22"
	m.inputs[3].CharLimit = 5
	m.inputs[4].Placeholder = "password (for first-time key setup, optional)"
	m.inputs[5].Placeholder = "y/n (auto-generate SSH key?)"

	m.inputs[0].Focus()
	m.inputFocus = 0
}

func (m Model) Init() tea.Cmd {
	if m.connecting && m.sshHost != nil {
		return m.connectSSH()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.view {
		case ViewHome:
			return m.updateHome(msg)
		case ViewAddHost:
			return m.updateAddHost(msg)
		case ViewSSH:
			return m.updateSSH(msg)
		}

	case sshOutputMsg:
		if string(msg) != "" {
			m.sshOutput.WriteString(string(msg))
		}
		if m.sshSession != nil {
			return m, m.readSSHOutput()
		}
		return m, nil

	case sshErrorMsg:
		errorMsg := string(msg)
		m.sshLog = append(m.sshLog, fmt.Sprintf("✕ Error: %s", errorMsg))
		m.toast = errorMsg
		m.toastSuccess = false
		m.toastTimer = 50
		m.connecting = false
		m.view = ViewHome
		if m.sshSession != nil {
			m.sshSession.Close()
			m.sshSession = nil
		}
		return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })

	case sshConnectedMsg:
		m.connecting = false
		m.sshSession = msg.session
		m.sshLog = append(m.sshLog, "✓ Connected")
		if m.width > 0 && m.height > 0 {
			m.sshSession.Resize(m.width, m.height-2)
		}
		return m, m.readSSHOutput()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.sshSession != nil {
			m.sshSession.Resize(msg.Width, msg.Height-2)
		}

	case tea.MouseMsg:
		// mouse support ready

	case tickMsg:
		if m.toastTimer > 0 {
			m.toastTimer--
			if m.toastTimer == 0 {
				m.toast = ""
				m.toastSuccess = false
			} else {
				return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
			}
		}
	}

	return m, nil
}

func (m Model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.hosts)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Add):
		m.view = ViewAddHost
		m.initAddHostInputs()
		return m, m.inputs[0].Focus()

	case key.Matches(msg, m.keys.Delete):
		if len(m.hosts) > 0 && m.cursor < len(m.hosts) {
			config.RemoveHost(m.cursor)
			m.hosts = config.GetHosts()
			if m.cursor >= len(m.hosts) && m.cursor > 0 {
				m.cursor--
			}
		}

	case key.Matches(msg, m.keys.Connect), key.Matches(msg, m.keys.Enter):
		if len(m.hosts) > 0 && m.cursor < len(m.hosts) {
			h := m.hosts[m.cursor]
			m.sshHost = &h
			m.connecting = true
			m.view = ViewSSH
			m.sshOutput.Reset()
			m.sshLog = []string{"→ Connecting..."}
			return m, m.connectSSH()
		}
	}

	return m, nil
}

func (m Model) updateAddHost(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+p":
		m.showPassword = !m.showPassword
		if m.showPassword {
			m.inputs[4].EchoMode = textinput.EchoNormal
		} else {
			m.inputs[4].EchoMode = textinput.EchoPassword
			m.inputs[4].EchoCharacter = '•'
		}
		return m, nil

	case "esc":
		m.view = ViewHome
		m.inputs = nil
		m.showPassword = false
		return m, nil

	case "enter":
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		if m.inputs[0].Value() != "" && m.inputs[1].Value() != "" {
			m.saveHost()
			if m.toastTimer > 0 {
				return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
			}
		}
		return m, nil

	case "tab":
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		return m, nil

	case "shift+tab":
		if m.inputFocus > 0 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus--
			return m, m.inputs[m.inputFocus].Focus()
		}
		return m, nil

	case "up":
		if m.inputFocus > 0 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus--
			return m, m.inputs[m.inputFocus].Focus()
		}
		return m, nil

	case "down":
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		return m, nil
	}

	// Let textinput handle all other keys (including 'k', 'j', etc.)
	var cmd tea.Cmd
	m.inputs[m.inputFocus], cmd = m.inputs[m.inputFocus].Update(msg)
	return m, cmd
}

func (m *Model) saveHost() {
	name := m.inputs[0].Value()
	hostname := m.inputs[1].Value()

	if name == "" || hostname == "" {
		m.toast = "Name and Host are required"
		m.toastSuccess = false
		m.toastTimer = 50
		return
	}

	port := 22
	if p := m.inputs[3].Value(); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	user := m.inputs[2].Value()
	if user == "" {
		user = "root"
	}

	password := m.inputs[4].Value()
	autoGenKey := strings.ToLower(m.inputs[5].Value()) == "y" || strings.ToLower(m.inputs[5].Value()) == "yes"

	var identityFile string
	if autoGenKey {
		privatePath, _, err := ssh.GenerateKeyPair(name)
		if err != nil {
			m.toast = fmt.Sprintf("Failed to generate key: %v", err)
			m.toastSuccess = false
			m.toastTimer = 50
			return
		}
		identityFile = privatePath
	}

	sshHost := ssh.Host{
		Name:         name,
		Hostname:     hostname,
		User:         user,
		Port:         port,
		IdentityFile: identityFile,
	}

	if password == "" && identityFile == "" {
		m.toast = "Password or existing key required for validation"
		m.toastSuccess = false
		m.toastTimer = 50
		return
	}

	m.toast = "→ Validating connection..."
	m.toastSuccess = false
	m.toastTimer = 100

	if err := ssh.ValidateConnection(sshHost, password); err != nil {
		m.toast = fmt.Sprintf("Validation failed: %v", err)
		m.toastSuccess = false
		m.toastTimer = 50
		return
	}

	var encryptedPassword string
	if password != "" {
		var err error
		encryptedPassword, err = config.EncryptPassword(password)
		if err != nil {
			m.toast = fmt.Sprintf("Failed to encrypt password: %v", err)
			m.toastSuccess = false
			m.toastTimer = 50
			return
		}
	}

	keyDeployed := false
	if autoGenKey && password != "" && identityFile != "" {
		m.toast = "→ Deploying SSH key..."
		m.toastSuccess = false
		m.toastTimer = 100
		publicKeyPath := identityFile + ".pub"
		if err := ssh.DeployPublicKey(sshHost, password, publicKeyPath); err == nil {
			keyDeployed = true
			m.toast = "✓ SSH key deployed successfully"
			m.toastSuccess = true
			m.toastTimer = 30
		} else {
			m.toast = fmt.Sprintf("Key deployment failed: %v", err)
			m.toastSuccess = false
			m.toastTimer = 50
			return
		}
	}

	h := config.Host{
		Name:            name,
		Hostname:        hostname,
		User:            user,
		Port:            port,
		IdentityFile:    identityFile,
		Password:        encryptedPassword,
		KeyDeployed:     keyDeployed,
		AutoGenerateKey: autoGenKey,
	}

	config.AddHost(h)
	if err := config.Save(); err != nil {
		m.toast = fmt.Sprintf("Failed to save: %v", err)
		m.toastSuccess = false
		m.toastTimer = 50
		return
	}

	m.hosts = config.GetHosts()
	if keyDeployed {
		m.toast = fmt.Sprintf("Host '%s' ready", name)
	} else {
		m.toast = fmt.Sprintf("Host '%s' added", name)
	}
	m.toastSuccess = true
	m.toastTimer = 30
	m.view = ViewHome
	m.inputs = nil
	m.showPassword = false
}

func (m Model) View() string {
	switch m.view {
	case ViewAddHost:
		return m.viewAddHost()
	case ViewSSH:
		return m.viewSSH()
	default:
		return m.viewHome()
	}
}

type sshOutputMsg string
type sshErrorMsg string
type sshConnectedMsg struct {
	session *ssh.Session
}
type tickMsg struct{}


func (m Model) connectSSH() tea.Cmd {
	return func() tea.Msg {
		if m.sshHost == nil {
			return sshErrorMsg("no host selected")
		}

		host := m.sshHost
		h := ssh.Host{
			Name:         host.Name,
			Hostname:     host.Hostname,
			User:         host.User,
			Port:         host.Port,
			IdentityFile: host.IdentityFile,
		}

		var password string
		if host.Password != "" {
			decrypted, err := config.DecryptPassword(host.Password)
			if err != nil {
				return sshErrorMsg(fmt.Sprintf("failed to decrypt password: %v", err))
			}
			password = decrypted
		}

		if password == "" && host.IdentityFile == "" {
			return sshErrorMsg("no authentication method available (need password or key)")
		}

		skipKey := !host.KeyDeployed && host.IdentityFile != ""
		session, err := ssh.Connect(h, password, skipKey)
		if err != nil {
			return sshErrorMsg(err.Error())
		}

		return sshConnectedMsg{session: session}
	}
}


func (m Model) readSSHOutput() tea.Cmd {
	return func() tea.Msg {
		if m.sshSession == nil {
			return nil
		}

		buf := make([]byte, 4096)
		n, err := m.sshSession.Read(buf)
		if err == io.EOF {
			return sshErrorMsg("connection closed")
		}
		if err != nil {
			return sshErrorMsg(err.Error())
		}

		if n > 0 {
			return sshOutputMsg(string(buf[:n]))
		}
		return nil
	}
}

func (m Model) updateSSH(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		if m.sshSession != nil {
			m.sshSession.Close()
			m.sshSession = nil
		}
		m.view = ViewHome
		m.sshHost = nil
		m.sshOutput.Reset()
		m.sshLog = nil
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		if m.sshSession != nil {
			m.sshSession.Close()
			m.sshSession = nil
		}
		return m, tea.Quit
	}

	if m.sshSession != nil {
		var data []byte
		if msg.Type == tea.KeyRunes {
			data = []byte(msg.String())
		} else {
			switch msg.String() {
			case "enter":
				data = []byte("\r")
			case "backspace":
				data = []byte("\x7f")
			case "tab":
				data = []byte("\t")
			case "space":
				data = []byte(" ")
			case "up":
				data = []byte("\x1b[A")
			case "down":
				data = []byte("\x1b[B")
			case "right":
				data = []byte("\x1b[C")
			case "left":
				data = []byte("\x1b[D")
			case "ctrl+c":
				data = []byte("\x03")
			case "ctrl+d":
				data = []byte("\x04")
			case "esc":
				data = []byte("\x1b")
			default:
				data = []byte(msg.String())
			}
		}
		if len(data) > 0 {
			m.sshSession.Write(data)
			return m, m.readSSHOutput()
		}
	}

	return m, m.readSSHOutput()
}

func Run() error {
	m := New()
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
