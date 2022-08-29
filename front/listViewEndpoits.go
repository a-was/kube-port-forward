package front

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"
)

func (m model) handleEndpointView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tea.KeyPgUp.String(), tea.KeyPgDown.String():
		m.view = podsView
		return m.Update(kube.MapUpdateMsg{})

	case tea.KeyCtrlLeft.String():
		m.view = endpointAddView
		m = m.endpointInputs()
		return m.render()

	case tea.KeyDelete.String():
		kube.DeleteService(m.list.SelectedItem().FilterValue())
	}
	return m.render()
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
		log.Info(it)
		items = append(items, it)

	}
	return
}
