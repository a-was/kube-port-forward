package front

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/main-kube/util"
)

func (m model) handleServiceAddView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tea.KeyEsc.String():
		return m.resetView()
	case tea.KeyTab.String(), tea.KeyEnter.String(), tea.KeyUp.String(), tea.KeyDown.String():
		return m.handleFocus(msg)
	}
	return m, nil
}

func (m model) serviceAddView() string {
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)
	fmt.Fprintf(&b, "\n%s\n", m.serviceAddError)

	return b.String()
}

func (m model) serviceInputs() model {
	m.inputs = make([]textinput.Model, 4)
	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.CursorStyle = cursorStyle
		t.CharLimit = 30

		switch i {
		case 0:
			t.Placeholder = "Name"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "Namespace"
		case 2:
			t.Placeholder = "HostPort"
			t.CharLimit = 5
		// case 3:
		// 	t.Placeholder = "KubePort"
		// 	t.CharLimit = 5
		case 3:
			t.Placeholder = "Address"
			t.SetValue(util.GetOutboundIP())
		}

		m.inputs[i] = t
	}

	return m

}

func (m model) setupEndpoint() (tea.Model, tea.Cmd) {

	hp, err := strconv.Atoi((m.inputs[2].Value()))
	if err != nil {
		return m.serviceError(err.Error())
	}

	// kp, err := strconv.Atoi((m.inputs[3].Value()))
	// if err != nil {
	// 	cmd = m.list.NewStatusMessage(err.Error())
	// 	return m, cmd
	// }

	end := kube.Endpoint{
		Name:      m.inputs[0].Value(),
		Namespace: m.inputs[1].Value(),
		HostPort:  hp,
		KubePort:  hp,
		Addr:      m.inputs[3].Value(),
	}
	if end.CheckServiceExists() {
		return m.serviceError("Service already exists")
	}

	m.view = 3
	m.serviceAddError = ""
	go func(m model) {
		if err := end.CreateService(); err != nil {
			m.notify <- statusMessage{err.Error()}
		}
		m.notify <- statusMessage{fmt.Sprintf("Service created: %s", end.Name)}
	}(m)
	return m, nil
}

func (m model) serviceError(msg string) (tea.Model, tea.Cmd) {
	m.serviceAddError = errColour + msg
	return m.Update(statusMessage{text: msg})
}
