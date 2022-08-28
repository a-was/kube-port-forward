package front

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"

	"github.com/charmbracelet/bubbles/list"
)

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
	if pod.PodPortForwardA != nil && pod.LocalPort > 0 {
		desc += fmt.Sprintf(" | %d -> %d %s", pod.PodPort, pod.LocalPort, connectionStatus(pod))
	}
	width := areaWidth - len(desc) - len(pod.Status) - 6
	for i := 0; i < width; i++ {
		desc += " "
	}
	desc += pod.Status
	return
}
func connectionStatus(pod *kube.Pod) string {
	if pod.Condition {
		return "✔️"
	}
	return "❌"
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
				if pod.Value.PodPortForwardA != nil {
					go ping(pod.Value)
				}
			}
		}
	}
}
func ping(p *kube.Pod) {
	if p.PodPortForwardA == nil {
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
