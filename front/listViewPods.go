package front

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"
)

func (m model) handlePodsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tea.KeyPgUp.String():
		m.view = endpointView
		m.lastView = m.view
		m = m.resetCursor()
		return m.Update(kube.MapUpdateMsg{})

	case tea.KeyPgDown.String():
		m.view = servicesView
		m.lastView = m.view
		m = m.resetCursor()
		return m.Update(kube.MapUpdateMsg{})

	case tea.KeyDelete.String():
		m.selectedPod = findPod(m.list.SelectedItem().FilterValue())
		m.selectedService = nil
		return m.toDelete()

	case tea.KeyEnter.String():
		// need to use findPod bc i don't know how to get desc from list.item
		m.selectedPod = findPod(m.list.SelectedItem().FilterValue())
		if m.selectedPod.Status != "Ready" {
			m.notify <- statusMessage{text: errColour + "Pod is not ready"}
			return m.render()
		}
		m = m.forwardInputs()
		m.view = podForwardView
		return m.render()
	}
	return m, nil
}

func createNewPodList() (items []list.Item) {
	items = make([]list.Item, 0, 30)
	// var name string
	for element := range kube.Map.Iter() {
		for pod := range element.Value.Iter() {
			// name = pod.Value.Name
			it := item{
				title: prettyTitle(pod.Value),
				desc:  prettyDesc(pod.Value),
			}
			items = append(items, it)
		}
	}
	return
}

func prettyDesc(pod *kube.Pod) (desc string) {
	desc = pod.Namespace
	var width int
	for _, pf := range pod.PFs {
		if pf != nil && pf.LocalPort > 0 {
			c, i := connectionStatus(pf)
			width += i
			desc += fmt.Sprintf(" | %d -> %d %s", pf.KubePort, pf.LocalPort, c)
		}
	}
	width += areaWidth - len(desc) - len(pod.Status) - 6
	// if strings.HasSuffix(pod.Status, "mReady") {
	// 	width += 13
	// } else {
	// 	width += 14
	// }
	for i := 0; i < width; i++ {
		desc += " "
	}
	switch pod.Status {
	case "Ready":
		desc += docstyleReady.Render(pod.Status)
	case "Error":
		desc += docstyleTerm.Render(pod.Status)
	default:
		desc += docstylePodErr.Render(pod.Status)
	}
	return
}

func prettyTitle(pod *kube.Pod) (title string) {
	title = pod.Name
	return
}

func connectionStatus(pf *kube.PortForwardA) (string, int) {
	if pf.Condition {
		return "✅ ", 1
	}
	return "❌", 1
}

func findPod(name string) *kube.Pod {
	for item := range kube.Map.Iter() {
		for v := range item.Value.Iter() {
			if v.Value.Name == name {
				return v.Value
			}
		}
	}
	return nil
}
