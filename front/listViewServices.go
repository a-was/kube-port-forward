package front

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"
)

func (m model) handleServicesView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tea.KeyPgUp.String(), tea.KeyPgDown.String():
		m.view = podsView
		m.lastView = m.view
		return m.Update(kube.MapUpdateMsg{})

	case tea.KeyDelete.String():
		m.selectedService = findService(m.list.SelectedItem().FilterValue())
		m.selectedPod = nil
		return m.toDelete()

	case tea.KeyEnter.String():
		m.view = serviceForwardView
		m = m.forwardInputs()
		// need to use findPod bc i don't know how to get desc from list.item
		m.selectedService = findService(m.list.SelectedItem().FilterValue())
		return m.render()
	}
	return m, nil
}

func findService(name string) *kube.Service {
	for item := range kube.Services.Iter() {
		for v := range item.Value.Iter() {
			if v.Value.Name == name {
				return v.Value
			}
		}
	}
	return nil
}

func prettyServiceDesc(ser *kube.Service) (desc string) {
	desc = ser.Namespace
	for _, pf := range ser.PFs {
		if pf != nil && pf.LocalPort > 0 {
			c, _ := connectionStatus(pf)
			desc += fmt.Sprintf(" | %d -> %d %s", pf.KubePort, pf.LocalPort, c)
		}
	}
	return
}

func createNewServiceList() (items []list.Item) {
	items = make([]list.Item, 0, 30)
	var name string
	for element := range kube.Services.Iter() {
		for service := range element.Value.Iter() {
			name = service.Value.Name

			it := item{
				title: name,
				desc:  prettyServiceDesc(service.Value),
			}
			items = append(items, it)
		}
	}
	return
}
