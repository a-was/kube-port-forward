package front

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handlePodsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tea.KeyPgUp.String(), tea.KeyPgDown.String():
		m.view = serviceView
		return m.Update(kube.MapUpdateMsg{})
	case tea.KeyCtrlLeft.String():
		m.view = serviceAddView
		m = m.serviceInputs()
		return m.render()
	case tea.KeyEnter.String():
		m.view = forwardView
		m = m.forwardInputs()
		// need to use findPod bc i don't know how to get desc from list.item
		m.selectedPod = findPod(m.list.SelectedItem().FilterValue())
		return m.render()
	}
	return m, nil
}

func (m model) handleServiceView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tea.KeyPgUp.String(), tea.KeyPgDown.String():
		m.view = podsView
		return m.Update(kube.MapUpdateMsg{})

	case tea.KeyCtrlLeft.String():
		m.view = serviceAddView
		m = m.serviceInputs()
		return m.render()

	case tea.KeyDelete.String():
		kube.DeleteService(m.list.SelectedItem().FilterValue())
	}
	return m.render()
}

func (m model) handleUpdateList() (tea.Model, tea.Cmd) {
	if len(m.list.Items()) == 0 {
		m.list.StopSpinner()
	}
	var items []list.Item
	if m.view == 3 {
		items = createNewServiceList()
		m.list.SetItems(items)
		m.list.Title = "Services"
		return m, waitForActivity(m.notify)
	}

	items = createNewPodList()
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

func createNewServiceList() (items []list.Item) {
	items = make([]list.Item, 0, 30)
	var name, desc string
	for element := range kube.Services.Iter() {
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
func createNewPodList() (items []list.Item) {
	items = make([]list.Item, 0, 30)
	var name, desc string
	for element := range kube.Map.Iter() {
		for pod := range element.Value.Iter() {
			name = pod.Value.Name
			desc = prettyDesc(pod.Value)

			it := item{
				title: name,
				desc:  desc,
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
			desc += fmt.Sprintf(" | %d -> %d %s", pf.PodPort, pf.LocalPort, c)
		}
	}
	width += areaWidth - len(desc) - len(pod.Status) - 6
	for i := 0; i < width; i++ {
		desc += " "
	}
	desc += pod.Status
	return
}
func connectionStatus(pf *kube.PodPortForwardA) (string, int) {
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
func ping(p *kube.PodPortForwardA) {
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
