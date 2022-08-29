package front

import (
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"
)

func (m model) handlePodsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tea.KeyPgUp.String(), tea.KeyPgDown.String():
		m.view = servicesView
		m.lastView = m.view
		return m.Update(kube.MapUpdateMsg{})
	case tea.KeyDelete.String():
		m.selectedPod = findPod(m.list.SelectedItem().FilterValue())
		m.selectedService = nil
		return m.toDelete()

	// case tea.KeyCtrlLeft.String():
	// 	m.view = serviceAddView
	// 	m = m.endpointInputs()
	// 	return m.render()
	case tea.KeyEnter.String():
		m.view = podForwardView
		m = m.forwardInputs()
		// need to use findPod bc i don't know how to get desc from list.item
		m.selectedPod = findPod(m.list.SelectedItem().FilterValue())
		return m.render()
	}
	return m, nil
}

func createNewPodList() (items []list.Item) {
	items = make([]list.Item, 0, 30)
	var name string
	for element := range kube.Map.Iter() {
		for pod := range element.Value.Iter() {
			name = pod.Value.Name
			it := item{
				title: name,
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
	for i := 0; i < width; i++ {
		desc += " "
	}
	desc += pod.Status
	return
}

func connectionStatus(pf *kube.PortForwardA) (string, int) {
	if pf.Condition {
		return "✔️ ", 5
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

func testConnections() {
	for range time.Tick(time.Second) {
		for element := range kube.Map.Iter() {
			for pod := range element.Value.Iter() {
				for _, pf := range pod.Value.PFs {
					if pf != nil {
						go ping(pf)
					}
				}
			}
		}
	}
}

func ping(p *kube.PortForwardA) {
	if p == nil {
		return
	}
	_, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d", p.LocalPort))
	if err != nil {
		log.Info(err)
		p.Condition = false
		return
	}
	p.Condition = true
}
