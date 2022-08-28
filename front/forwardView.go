package front

import (
	"fmt"
	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) forwardView() string {
	var b strings.Builder
	b.WriteString("Pod Ports: ")
	for _, port := range m.selectedPod.Ports {
		b.WriteString(fmt.Sprintf("%s ", port))
	}
	b.WriteString("\n\n")

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

	return b.String()
}

func (m model) forwardInputs() model {
	m.inputs = make([]textinput.Model, 2)
	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.CursorStyle = cursorStyle
		t.CharLimit = 5

		switch i {
		case 0:
			t.Placeholder = "PodPort"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "LocalPort"
		}

		m.inputs[i] = t
	}

	return m

}

func (m model) setupForward() (model, tea.Cmd) {
	m.view = 0
	var cmd tea.Cmd
	if m.inputs[0].Value() == "" || m.inputs[1].Value() == "" {
		cmd = m.list.NewStatusMessage("One of ports was empty")
		return m, cmd
	}
	pp, err := strconv.Atoi((m.inputs[0].Value()))

	if err != nil {
		cmd = m.list.NewStatusMessage(err.Error())
		return m, cmd
	}

	lp, err := strconv.Atoi((m.inputs[1].Value()))
	if err != nil {
		cmd = m.list.NewStatusMessage(err.Error())
		return m, cmd
	}
	if m.selectedPod.PodPortForwardA != nil {
		m.selectedPod.Close()
	}

	m.selectedPod.PodPortForwardA = &kube.PodPortForwardA{
		PodPort:   pp,
		LocalPort: lp,
	}
	go func() {
		go m.selectedPod.Forward(flog)
		m.selectedPod.Ready()
		log.Info("Ports ready")
	}()
	m.notify <- kube.MapUpdateMsg{}
	return m, nil
}
