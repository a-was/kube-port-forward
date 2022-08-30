package front

import (
	"fmt"
	"time"

	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

type view int8

const (
	podsView           view = 0
	podForwardView     view = 1
	endpointAddView    view = 2
	endpointView       view = 3
	servicesView       view = 4
	serviceForwardView view = 5
	deleteForwardView  view = 6
)

var (
	docstyleReady  = lipgloss.NewStyle().Foreground(lipgloss.Color("#33cc00")) /* .UnderlineSpaces(true).Underline(true) */
	docstylePodErr = lipgloss.NewStyle().Foreground(lipgloss.Color("#ac6c00")) /* .UnderlineSpaces(true).Underline(true) */
	docstyleTerm   = lipgloss.NewStyle().Foreground(lipgloss.Color("#60686c")) /* .UnderlineSpaces(true).Underline(true) */
	docStyle       = lipgloss.NewStyle().Margin(0, 0)
	focusedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	noStyle        = lipgloss.NewStyle()
	cursorStyle    = focusedStyle.Copy()
	blurredStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	focusedButton  = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton  = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
	errColour      = "\033[38:5:204m"

	log       *zap.SugaredLogger
	areaWidth int
)

type statusMessage struct {
	text string
}
type renderMsg struct{}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list            list.Model
	inputs          []textinput.Model
	focusIndex      int
	podPortfill     int
	servicePortfill int
	selectedPod     *kube.Pod
	selectedService *kube.Service

	view     view
	lastView view

	forwardError     string
	endpointAddError string

	notify chan any
}

func Start() {
	log = zap.S()
	f, err := tea.LogToFile("teaLog", "xD")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	items := []list.Item{}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedDesc.BorderBottom(true).BorderLeft(false)
	delegate.Styles.SelectedTitle.BorderLeft(false)

	m := model{list: list.New(items, delegate, 0, 0), notify: make(chan any, 10)}
	m.list.KeyMap = initKeyMap()
	go kube.UpdateMap(m.notify)
	go kube.UpdateServiceMap(m.notify)
	go testConnections()
	ti := textinput.New()
	ti.CharLimit = 6
	ti.Width = 20
	x := spinner.MiniDot
	m.list.SetSpinner(x)
	m.list.StartSpinner()
	m.list.StatusMessageLifetime = time.Second * 10
	m.list.Title = "Services"
	m.lastView, m.view = 4, 4

	p := tea.NewProgram(m, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		return
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		spinner.Tick,
		waitForActivity(m.notify), // wait for activity
	)
}
func (m model) View() string {
	switch m.view {
	case podsView, endpointView, servicesView, deleteForwardView:
		return m.listView()
	case podForwardView:
		return m.podForwardView()
	case serviceForwardView:
		return m.serviceForwardView()
	case endpointAddView:
		return m.endpointAddView()
	}
	return "Something went wrong"
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.view {
		case podsView:
			m, cmd := m.handlePodsView(msg)
			if cmd != nil {
				return m, cmd
			}

		case podForwardView:
			m, cmd := m.handlePodForwardView(msg)
			if cmd != nil {
				return m, cmd
			}
		case endpointAddView:
			m, cmd := m.handleEndpointAddView(msg)
			if cmd != nil {
				return m, cmd
			}
		case endpointView:
			m, cmd := m.handleEndpointView(msg)
			if cmd != nil {
				return m, cmd
			}
		case servicesView:
			m, cmd := m.handleServicesView(msg)
			if cmd != nil {
				return m, cmd
			}
		case serviceForwardView:
			m, cmd := m.handleServiceForwardView(msg)
			if cmd != nil {
				return m, cmd
			}
		case deleteForwardView:
			m, cmd := m.handleDeleteForwardView(msg)
			if cmd != nil {
				return m, cmd
			}
		}
		if msg.String() == "ctrl+c" {
			closeAllConns()
			return m, tea.Quit
		}

	case renderMsg:
		return m, nil
	case kube.MapUpdateMsg:
		return m.handleUpdateList()

	case statusMessage:
		// m.view = 0
		cmd := m.list.NewStatusMessage(msg.text)
		return m, cmd

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		areaWidth = msg.Width - h
		m.list.SetSize(areaWidth, msg.Height-v)

	}

	var cmd tea.Cmd
	switch m.view {
	case endpointView, podsView, servicesView, deleteForwardView:
		m.list, cmd = m.list.Update(msg)
	case podForwardView, endpointAddView, serviceForwardView:
		cmd = m.updateInputs(msg)
	}

	return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds = make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) handleFocus(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()

	// Did the user press enter while the submit button was focused?
	// If so, exit.
	if s == "enter" && m.focusIndex == len(m.inputs) {
		if !m.checkInputs() {
			return m.fpError("All fields have to be filled")
		}
		switch m.view {
		case podForwardView, serviceForwardView:
			return m.setupForward()
		case endpointAddView:
			return m.setupEndpoint()
		}

	}

	// Cycle indexes
	if s == "up" {
		m.focusIndex--
		m.podPortfill = 0
	} else if s != "tab" {
		m.focusIndex++
		m.podPortfill = 0
	}

	if m.focusIndex > len(m.inputs) {
		m.focusIndex = 0
	} else if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs)
	}

	cmds := make([]tea.Cmd, len(m.inputs))
	for i := 0; i <= len(m.inputs)-1; i++ {
		if i == m.focusIndex {
			// Set focused state
			cmds[i] = m.inputs[i].Focus()
			m.inputs[i].PromptStyle = focusedStyle
			m.inputs[i].TextStyle = focusedStyle
			continue
		}
		// Remove focused state
		m.inputs[i].Blur()
		m.inputs[i].PromptStyle = noStyle
		m.inputs[i].TextStyle = noStyle
	}
	cmds = append(cmds, func() tea.Msg { return renderMsg{} })
	return m, tea.Batch(cmds...)
}

func (m model) resetView() (tea.Model, tea.Cmd) {
	m.view = m.lastView
	m.focusIndex = 0
	return m.render()
}

func (m model) render() (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		return renderMsg{}
	}
}

func waitForActivity(sub chan any) tea.Cmd {
	return func() tea.Msg {
		msg := <-sub
		switch t := msg.(type) {
		case kube.MapUpdateMsg:
			return t
		case statusMessage:
			return t
		case error:
			return statusMessage{errColour + t.Error()}
		}
		return nil
	}
}

func closeAllConns() {
	for element := range kube.Map.Iter() {
		for pod := range element.Value.Iter() {
			for _, pf := range pod.Value.PFs {
				if pf != nil {
					pf.Close()
				}
			}
		}
	}
}

func (m model) checkInputs() bool {
	for _, inp := range m.inputs {
		if inp.Value() == "" {
			log.Debug(inp.Placeholder)
			return false
		}
	}
	return true
}

func initKeyMap() list.KeyMap {
	return list.KeyMap{
		CursorUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev page"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next page"),
		),
		GoToStart: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "go to start"),
		),
		GoToEnd: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "go to end"),
		),
		// Filtering.
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),
		CancelWhileFiltering: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		AcceptWhileFiltering: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "apply filter"),
		),
		ShowFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more"),
		),
		CloseFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "close help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(key.WithKeys("ctrl+c")),
	}
}
