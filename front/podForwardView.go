package front

import (
	"fmt"
	"strings"

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
