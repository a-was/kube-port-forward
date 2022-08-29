package front

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"

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
		return m.error(err.Error())
	}

	// check port is already forwarded
	if m.checkPorts(pp) {
		return m.error("Port already forwarded")
	}

	if !checkLocalPort(strconv.Itoa(lp)) {
		return m.error("Local port is taken")
	}

	m.view = 0
	pf := &kube.PodPortForwardA{
		PodPort:   pp,
		LocalPort: lp,
	}

	go func() {
		m.selectedPod.PFs = append(m.selectedPod.PFs, pf)
		go m.selectedPod.Forward(pf)
		pf.Ready()
		log.Info("Ports ready")
	}()
	m.notify <- kube.MapUpdateMsg{}
	return m, nil
}

func (m model) checkPorts(pp int) bool {
	for _, pf := range m.selectedPod.PFs {
		if pf.PodPort == pp {
			return true
		}
	}
	return false
}

func checkLocalPort(lp string) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", lp), timeout)
	if err != nil {
		return true
	}
	if conn != nil {
		defer conn.Close()
		return false
	}
	return false
}
