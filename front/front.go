package front

import (
	"fmt"
	"os"
	"time"

	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/main-kube/util/slice"
	"go.uber.org/zap"
)

var (
	docStyle      = lipgloss.NewStyle().Margin(0, 0)
	focusedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	noStyle       = lipgloss.NewStyle()
	cursorStyle   = focusedStyle.Copy()
	blurredStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))

	log       = zap.S()
	areaWidth int
	flog      *os.File
)

type statusMessage struct {
	text string
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list        list.Model
	inputs      []textinput.Model
	view        int8
	focusIndex  int
	podPortfill int
	selectedPod *kube.Pod

	notify chan any
}

func Start() {
	f, err := tea.LogToFile("teaLog", "xD")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	items := []list.Item{}
	m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0), notify: make(chan any, 10)}
	m.list.KeyMap = initKeyMap()
	go kube.UpdateMap(m.notify)
	go testConnections()
	ti := textinput.New()
	ti.CharLimit = 6
	ti.Width = 20
	x := spinner.MiniDot
	m.list.SetSpinner(x)
	m.list.StartSpinner()
	m.list.StatusMessageLifetime = time.Second * 10
	m.list.Title = "Pods"

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
	case 0, 3:
		return m.listView()
	case 1:
		return m.forwardView()
	case 2:
		return m.serviceAddView()
	}
	return "Something went wrong"
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// if m.list.FilterState() == list.Filtering {
		// 	break
		// }
		ms := msg.String()
		switch {
		case ms == tea.KeyPgUp.String(), ms == tea.KeyPgDown.String():
			switch m.view {
			case 0:
				m.view = 3
			case 3:
				m.view = 0
			}
			return m.Update(kube.MapUpdateMsg{})

		case ms == tea.KeyDelete.String() && m.view == 3:
			kube.DeleteService(m.list.SelectedItem().FilterValue())

		case ms == "ctrl+c":
			closeAllConns()
			return m, tea.Quit

		case ms == "enter" && m.view == 0:
			m.view = 1
			m = m.forwardInputs()
			// need to use findPod bc i don't know how to get desc from list.item
			m.selectedPod = findPod(m.list.SelectedItem().FilterValue())
			return m, nil

		case ms == tea.KeyCtrlLeft.String() && (m.view == 0 || m.view == 3):
			m.view = 2
			m = m.serviceInputs()
			return m, nil

		case slice.Contains([]string{"esc", "tab", "enter", "up", "down"}, ms) && m.view == 1:
			if ms == "esc" {
				return m.resetView()
			}
			if ms == "tab" {
				if m.podPortfill > len(m.selectedPod.Ports) || m.podPortfill >= len(m.selectedPod.Ports) {
					m.podPortfill = 0
				}
				if m.focusIndex >= len(m.inputs) {
					m.focusIndex = 0
				}
				m.inputs[m.focusIndex].SetValue(m.selectedPod.Ports[m.podPortfill])
				m.podPortfill++
			}
			return m.handleFocus(msg)

		case slice.Contains([]string{"esc", "tab", "enter", "up", "down"}, ms) && m.view == 2:
			if ms == "esc" {
				return m.resetView()
			}
			return m.handleFocus(msg)
		}

	case kube.MapUpdateMsg:
		if len(m.list.Items()) == 0 {
			m.list.StopSpinner()
		}
		var items []list.Item
		if m.view == 3 {
			items = createNewServiceList()
			m.list.SetItems(items)
			m.list.Title = "Services"
			return m, waitForActivity(m.notify)
		}

		items = createNewPodList()
		m.list.Title = "Pods"
		m.list.SetItems(items)
		return m, waitForActivity(m.notify)

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
	case 0:
		m.list, cmd = m.list.Update(msg)
	case 1, 2:
		cmd = m.updateInputs(msg)
	case 3:
		m.list, cmd = m.list.Update(msg)

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

func (m model) handleFocus(msg tea.KeyMsg) (model, tea.Cmd) {
	s := msg.String()

	// Did the user press enter while the submit button was focused?
	// If so, exit.
	if s == "enter" && m.focusIndex == len(m.inputs) {
		if !m.checkInputs() {
			cmd := m.list.NewStatusMessage("All fields have to be filled")
			return m, cmd
		}
		switch m.view {
		case 1:
			return m.setupForward()
		case 2:
			return m.setupEndpoint()
		}

	}

	// Cycle indexes
	if s == "up" || s == "shift+tab" {
		m.focusIndex--
	} else if s != "tab" || (s == "tab" && m.view != 1) {
		m.focusIndex++
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

	return m, tea.Batch(cmds...)
}

func (m model) resetView() (tea.Model, tea.Cmd) {
	m.view = 0
	m.focusIndex = 0
	return m, nil
}

func waitForActivity(sub chan any) tea.Cmd {
	return func() tea.Msg {
		msg := <-sub
		switch t := msg.(type) {
		case kube.MapUpdateMsg:
			return t
		case statusMessage:
			return t
		}
		return nil
	}
}

func closeAllConns() {
	for element := range kube.Map.Iter() {
		for pod := range element.Value.Iter() {
			if pod.Value.PodPortForwardA != nil {
				pod.Value.Close()
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
