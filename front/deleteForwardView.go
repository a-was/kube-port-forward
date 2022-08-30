package front

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/config"
	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/dns"
	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/kube"
	"github.com/main-kube/util"
	"go.uber.org/zap"
)

func (m model) toDelete() (tea.Model, tea.Cmd) {
	switch m.view {
	case podsView:
		m.selectedPod = findPod(m.list.SelectedItem().FilterValue())
		m.list.Title = m.selectedPod.Name
		m.selectedService = nil
		if len(m.selectedPod.PFs) == 1 {
			pf := m.selectedPod.PFs[0]
			pf.Close()
			dns.Unregister(fmt.Sprintf(config.DNS_SERVICE_FMT, pf.Name, pf.Namespace))
			return m.render()
		}
	case servicesView:
		m.selectedService = findService(m.list.SelectedItem().FilterValue())
		m.list.Title = m.selectedService.Name
		m.selectedPod = nil
		if len(m.selectedService.PFs) == 1 {
			pf := m.selectedService.PFs[0]
			pf.Close()
			dns.Unregister(fmt.Sprintf(config.DNS_SERVICE_FMT, pf.Name, pf.Namespace))
			return m.render()
		}
	}
	m.lastView = m.view
	m.view = deleteForwardView
	return m.Update(kube.MapUpdateMsg{})
}

func (m model) handleDeleteForwardView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	log = zap.S()
	switch msg.String() {
	case tea.KeyEscape.String():
		return m.resetView()

	case tea.KeyDelete.String():
		kubePort := strings.TrimSpace(strings.Split(m.list.Items()[m.list.Index()].FilterValue(), "->")[0])
		m.stopPF(kubePort)
		return m.resetView()
	}
	return m, nil
}

func (m model) stopPF(kubeport string) {
	kp := util.Must(strconv.Atoi(kubeport))
	var PFs []*kube.PortForwardA
	switch m.lastView {
	case podsView:
		PFs = m.selectedPod.PFs
	case servicesView:
		PFs = m.selectedService.PFs
	}
	for _, pf := range PFs {
		if pf.KubePort == kp {
			go pf.Close()
		}
	}

}

func (m model) createToDeleteList() (items []list.Item) {
	switch m.lastView {
	case podsView:
		items = make([]list.Item, 0, len(m.selectedPod.PFs))
		for _, pf := range m.selectedPod.PFs {
			c, _ := connectionStatus(pf)
			items = append(items, item{title: fmt.Sprintf("%d -> %d %s |", pf.KubePort, pf.LocalPort, c), desc: "----------------"})
		}
	case servicesView:
		items = make([]list.Item, 0, len(m.selectedService.PFs))
		for _, pf := range m.selectedService.PFs {
			c, _ := connectionStatus(pf)
			items = append(items, item{title: fmt.Sprintf("%d -> %d %s", pf.KubePort, pf.LocalPort, c), desc: "----------------"})
		}
	}
	m.list.SetItems(items)
	return
}
