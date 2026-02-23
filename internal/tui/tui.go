package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hinshun/vt10x"
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
	sshVT        vt10x.Terminal
	sshHost      *config.Host
	connecting   bool
	toast        string
	toastTimer   int
	toastSuccess bool
	showPassword bool
	animFrame    int
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
	animTick := tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return animTickMsg{} })
	if m.connecting && m.sshHost != nil {
		return tea.Batch(m.connectSSH(), animTick)
	}
	return animTick
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
		if len(msg) > 0 && m.sshVT != nil {
			m.sshVT.Write([]byte(msg))
		}
		if m.sshSession != nil {
			return m, m.readSSHOutput()
		}
		return m, nil

	case sshErrorMsg:
		errorMsg := string(msg)
		m.toast = errorMsg
		m.toastSuccess = false
		m.toastTimer = 50
		m.connecting = false
		m.view = ViewHome
		if m.sshSession != nil {
			m.sshSession.Close()
			m.sshSession = nil
		}
		m.sshVT = nil
		return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })

	case sshConnectedMsg:
		m.connecting = false
		m.sshSession = msg.session
		cols, rows := m.width, m.height-2
		if cols < 80 {
			cols = 80
		}
		if rows < 24 {
			rows = 24
		}
		m.sshVT = vt10x.New(vt10x.WithSize(cols, rows))
		if m.width > 0 && m.height > 0 {
			m.sshSession.Resize(cols, rows)
		}
		return m, m.readSSHOutput()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		termCols, termRows := msg.Width, msg.Height-2
		if m.sshSession != nil {
			m.sshSession.Resize(termCols, termRows)
		}
		if m.sshVT != nil {
			m.sshVT.Resize(termCols, termRows)
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

	case animTickMsg:
		needsAnim := m.connecting || m.width == 0
		if needsAnim {
			m.animFrame++
			return m, tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return animTickMsg{} })
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
			m.animFrame = 0
			m.view = ViewSSH
			m.sshVT = nil
			return m, tea.Batch(
				m.connectSSH(),
				tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return animTickMsg{} }),
			)
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
type animTickMsg struct{}


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

		type connResult struct {
			session *ssh.Session
			err     error
		}
		ch := make(chan connResult, 1)
		go func() {
			session, err := ssh.Connect(h, password, skipKey)
			ch <- connResult{session, err}
		}()

		select {
		case res := <-ch:
			if res.err != nil {
				return sshErrorMsg(res.err.Error())
			}
			return sshConnectedMsg{session: res.session}
		case <-time.After(15 * time.Second):
			return sshErrorMsg("connection timed out (15s)")
		}
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
	// Ctrl+] is the only way to disconnect (standard SSH escape)
	if msg.String() == "ctrl+]" {
		if m.sshSession != nil {
			m.sshSession.Close()
			m.sshSession = nil
		}
		m.connecting = false
		m.view = ViewHome
		m.sshHost = nil
		m.sshVT = nil
		return m, nil
	}

	if m.sshSession == nil {
		return m, nil
	}

	var data []byte
	if msg.Type == tea.KeyRunes {
		data = []byte(msg.String())
	} else {
		switch msg.Type {
		case tea.KeyEnter:
			data = []byte("\r")
		case tea.KeyBackspace:
			data = []byte("\x7f")
		case tea.KeyTab:
			data = []byte("\t")
		case tea.KeySpace:
			data = []byte(" ")
		case tea.KeyUp:
			data = []byte("\x1b[A")
		case tea.KeyDown:
			data = []byte("\x1b[B")
		case tea.KeyRight:
			data = []byte("\x1b[C")
		case tea.KeyLeft:
			data = []byte("\x1b[D")
		case tea.KeyEscape:
			data = []byte("\x1b")
		case tea.KeyDelete:
			data = []byte("\x1b[3~")
		case tea.KeyHome:
			data = []byte("\x1b[H")
		case tea.KeyEnd:
			data = []byte("\x1b[F")
		case tea.KeyPgUp:
			data = []byte("\x1b[5~")
		case tea.KeyPgDown:
			data = []byte("\x1b[6~")
		case tea.KeyInsert:
			data = []byte("\x1b[2~")
		case tea.KeyF1:
			data = []byte("\x1bOP")
		case tea.KeyF2:
			data = []byte("\x1bOQ")
		case tea.KeyF3:
			data = []byte("\x1bOR")
		case tea.KeyF4:
			data = []byte("\x1bOS")
		case tea.KeyF5:
			data = []byte("\x1b[15~")
		case tea.KeyF6:
			data = []byte("\x1b[17~")
		case tea.KeyF7:
			data = []byte("\x1b[18~")
		case tea.KeyF8:
			data = []byte("\x1b[19~")
		case tea.KeyF9:
			data = []byte("\x1b[20~")
		case tea.KeyF10:
			data = []byte("\x1b[21~")
		case tea.KeyF11:
			data = []byte("\x1b[23~")
		case tea.KeyF12:
			data = []byte("\x1b[24~")
		default:
			s := msg.String()
			switch s {
			case "ctrl+c":
				data = []byte("\x03")
			case "ctrl+d":
				data = []byte("\x04")
			case "ctrl+z":
				data = []byte("\x1a")
			case "ctrl+l":
				data = []byte("\x0c")
			case "ctrl+a":
				data = []byte("\x01")
			case "ctrl+e":
				data = []byte("\x05")
			case "ctrl+k":
				data = []byte("\x0b")
			case "ctrl+u":
				data = []byte("\x15")
			case "ctrl+w":
				data = []byte("\x17")
			case "ctrl+r":
				data = []byte("\x12")
			case "ctrl+p":
				data = []byte("\x10")
			case "ctrl+n":
				data = []byte("\x0e")
			default:
				if len(s) == 1 {
					data = []byte(s)
				}
			}
		}
	}

	if len(data) > 0 {
		m.sshSession.Write(data)
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
