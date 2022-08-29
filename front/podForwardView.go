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

func (m model) handlePodForwardView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tea.KeyTab.String():
		if m.podPortfill > len(m.selectedPod.Ports) || m.podPortfill >= len(m.selectedPod.Ports) {
			m.podPortfill = 0
		}
		if m.focusIndex >= len(m.inputs) {
			m.focusIndex = 0
		}
		m.inputs[m.focusIndex].SetValue(m.selectedPod.Ports[m.podPortfill])
		m.podPortfill++
		return m.handleFocus(msg)

	case tea.KeyEscape.String():
		return m.resetView()

	case tea.KeyEnter.String(), tea.KeyUp.String(), tea.KeyDown.String():
		return m.handleFocus(msg)

	}
	return m, nil
}

func (m model) podForwardView() string {
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
	fmt.Fprintf(&b, "\n%s\n", m.forwardError)

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
			t.Placeholder = "ResourcePort"
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

func (m model) setupForward() (tea.Model, tea.Cmd) {

	if m.inputs[0].Value() == "" || m.inputs[1].Value() == "" {
		log.Info()
		return m.fpError("One of ports was empty")
	}
	pp, err := strconv.Atoi((m.inputs[0].Value()))

	if err != nil {
		return m.fpError(err.Error())
	}

	lp, err := strconv.Atoi((m.inputs[1].Value()))
	if err != nil {
		return m.fpError(err.Error())
	}
	// check port is already forwarded
	if m.checkPorts(pp) {
		return m.fpError("Port already forwarded")
	}
	// if !checkLocalPort(strconv.Itoa(lp)) {
	// 	return m.fpError("Local port is taken")
	// }

	var pf *kube.PortForwardA
	switch m.view {
	case podForwardView:
		pf = &kube.PortForwardA{
			Name:      m.selectedPod.Name,
			Namespace: m.selectedPod.Namespace,
			Resource:  "pods",
			KubePort:  pp,
			LocalPort: lp,
		}
		m.selectedPod.PFs = append(m.selectedPod.PFs, pf)
	case serviceForwardView:
		pf = &kube.PortForwardA{
			Name:      m.selectedService.Name,
			Namespace: m.selectedService.Namespace,
			Resource:  "services",
			KubePort:  pp,
			LocalPort: lp,
		}
		m.selectedService.PFs = append(m.selectedService.PFs, pf)
	}
	log.Info(m.selectedService.Name)
	log.Info(m.selectedService.Namespace)

	go func() {
		go pf.Forward()
		// pf.Ready()
		// log.Info("Ports ready")
	}()
	m.view = m.lastView
	m.forwardError = ""
	m.notify <- kube.MapUpdateMsg{}
	return m.render()
}

func (m model) checkPorts(pp int) bool {
	switch m.view {
	case podForwardView:
		for _, pf := range m.selectedPod.PFs {
			if pf.KubePort == pp {
				return true
			}
		}
	case serviceForwardView:
		for _, pf := range m.selectedService.PFs {
			if pf.KubePort == pp {
				return true
			}
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

func (m model) fpError(msg string) (tea.Model, tea.Cmd) {

	m.forwardError = errColour + msg
	return m.render()
}
