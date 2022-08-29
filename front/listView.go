package front

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleUpdateList() (tea.Model, tea.Cmd) {
	if len(m.list.Items()) == 0 {
		m.list.StopSpinner()
	}
	if m.view == endpointView {
		items := createNewEndpointList()
		m.list.SetItems(items)
		m.list.Title = "Endpoints"
		return m, waitForActivity(m.notify)
	}
	if m.view == serviceView {
		items := createNewServiceList()
		m.list.SetItems(items)
		m.list.Title = "Services"
		return m, waitForActivity(m.notify)
	}

	items := createNewPodList()
	m.list.Title = "Pods"
	m.list.SetItems(items)
	return m, waitForActivity(m.notify)
}

func (m model) listView() string {
	l := m.list.View()
	if strings.Contains(l, "No items found.") {
		l = strings.ReplaceAll(l, "No items found.", "Waiting for connection...")
	}
	return docStyle.Render(l)
}
