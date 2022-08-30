package front

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleUpdateList() (tea.Model, tea.Cmd) {
	if len(m.list.Items()) == 0 {
		m.list.StopSpinner()
	}
	switch m.view {
	case endpointView:
		items := createNewEndpointList()
		m.list.SetItems(items)
		m.list.Title = "Endpoints"
		return m, waitForActivity(m.notify)
	case servicesView:
		items := createNewServiceList()
		m.list.SetItems(items)
		m.list.Title = "Services"
		return m, waitForActivity(m.notify)
	case deleteForwardView:
		items := m.createToDeleteList()
		m.list.SetItems(items)
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

func (m model) resetCursor() model {
	m.list.Paginator.Page = 0
	for m.list.Cursor() != 0 {
		m.list.CursorUp()
	}
	return m
}
