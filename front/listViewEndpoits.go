package front

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"
)

func (m model) handleEndpointView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tea.KeyPgUp.String():
		m.view = servicesView
		m.lastView = m.view
		m.resetCursor()
		return m.Update(kube.MapUpdateMsg{})

	case tea.KeyPgDown.String():
		m.view = podsView
		m.lastView = m.view
		m.resetCursor()
		return m.Update(kube.MapUpdateMsg{})

	case "alt+[2~":
		m.view = endpointAddView
		m = m.endpointInputs()
		return m.render()

	case tea.KeyDelete.String():
		if m.list.SelectedItem() == nil {
			m.notify <- statusMessage{"Nothing to delete"}
			return m.render()
		}
		kube.DeleteEndpoint(m.list.SelectedItem().FilterValue())
	}
	return m, nil
}

func createNewEndpointList() (items []list.Item) {
	items = make([]list.Item, 0, 30)
	var name, desc string
	for element := range kube.Endpoints.Iter() {
		name = element.Value.Name
		desc = fmt.Sprintf("KubePort: %d, HostPort: %d, url: %s.%s:%[2]d", element.Value.KubePort, element.Value.HostPort, element.Value.Name, element.Value.Namespace)

		it := item{
			title: name,
			desc:  desc,
		}
		items = append(items, it)
	}
	return
}
