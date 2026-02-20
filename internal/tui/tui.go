package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shravan20/vecna/internal/config"
)

type View int

const (
	ViewHome View = iota
	ViewAddHost
)

type Model struct {
	view       View
	width      int
	height     int
	keys       KeyMap
	cursor     int
	hosts      []config.Host
	inputs     []textinput.Model
	inputFocus int
	err        error
}

func New() Model {
	return Model{
		view:  ViewHome,
		keys:  DefaultKeyMap(),
		hosts: config.GetHosts(),
	}
}

func (m *Model) initAddHostInputs() {
	m.inputs = make([]textinput.Model, 4)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "myserver"
	m.inputs[0].Focus()
	m.inputs[0].Prompt = "Name: "
	m.inputs[0].CharLimit = 64

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "192.168.1.100 or hostname.com"
	m.inputs[1].Prompt = "Host: "
	m.inputs[1].CharLimit = 256

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "root"
	m.inputs[2].Prompt = "User: "
	m.inputs[2].CharLimit = 64

	m.inputs[3] = textinput.New()
	m.inputs[3].Placeholder = "22"
	m.inputs[3].Prompt = "Port: "
	m.inputs[3].CharLimit = 5

	m.inputFocus = 0
}

func (m Model) Init() tea.Cmd {
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
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.MouseMsg:
		// mouse support ready
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
	}

	return m, nil
}

func (m Model) updateAddHost(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.view = ViewHome
		m.inputs = nil
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		m.saveHost()
		m.view = ViewHome
		m.inputs = nil
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.inputFocus > 0 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus--
			return m, m.inputs[m.inputFocus].Focus()
		}

	case key.Matches(msg, m.keys.Down):
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.inputFocus], cmd = m.inputs[m.inputFocus].Update(msg)
	return m, cmd
}

func (m *Model) saveHost() {
	port := 22
	if p := m.inputs[3].Value(); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	user := m.inputs[2].Value()
	if user == "" {
		user = "root"
	}

	h := config.Host{
		Name:     m.inputs[0].Value(),
		Hostname: m.inputs[1].Value(),
		User:     user,
		Port:     port,
	}

	config.AddHost(h)
	config.Save()
	m.hosts = config.GetHosts()
}

func (m Model) View() string {
	switch m.view {
	case ViewAddHost:
		return m.viewAddHost()
	default:
		return m.viewHome()
	}
}

func Run() error {
	p := tea.NewProgram(
		New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
