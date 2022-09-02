package kube

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/main-kube/util/safe"
	"github.com/main-kube/util/slice"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// pod map key=namespace
	Map = safe.NewSortedMap(map[string]*safe.SortedMap[string, *Pod]{}, func(data []string, i, j int) bool {
		return data[i] < data[j]
	})
)

type Pod struct {
	PFs       []*PortForwardA
	Name      string
	Namespace string
	OwnerName string
	Status    string
	IP        string
	Ports     []string
}
type MapUpdateMsg struct{}

func newPodMap() *safe.SortedMap[string, *Pod] {
	return safe.NewSortedMap(map[string]*Pod{}, func(data []string, i, j int) bool {
		return data[i] < data[j]
	})
}

func UpdateMap(notify chan any) {
	wg := sync.WaitGroup{}
	for range time.Tick(1 * time.Second) {
		if Client == nil {
			continue
		}

		// delete nonexistent namespaces
		go cleanMap(namespaces)
		for _, namespace := range namespaces {
			wg.Add(1)
			go func(namespace string) {
				defer wg.Done()
				go addPods(namespace, notify)
			}(namespace)
		}
		wg.Wait()
	}
}

func cleanMap(ns []string) {
	if Map.Len() > 0 {
		for _, key := range slice.Diff(Map.Keys(), ns) {
			Map.Delete(key)
		}
	}
}

func addPods(nsName string, notify chan any) {
	podlist, err := Client.API.CoreV1().Pods(nsName).List(Client.CTX, v1.ListOptions{})
	if err != nil {
		log.Error(err)
	}
	podMap, ok := Map.GetFull(nsName)
	if !ok {
		podMap = newPodMap()
	}
	nameList := make([]string, 0, len(podlist.Items))
	for _, p := range podlist.Items {
		if p.OwnerReferences == nil {
			continue
		}
		nameList = append(nameList, p.Name)
		pod, ok := podMap.GetFull(p.Name)
		if ok {
			if string(p.Status.Phase) != pod.Status {
				pod.Status = getPodStatus(p)
				podMap.Set(p.Name, pod)
			}
			continue
		}
		podMap.Set(p.Name, &Pod{
			Name:      p.Name,
			Namespace: nsName,
			Status:    getPodStatus(p),
			Ports:     fillPorts(p),
			IP:        p.Status.PodIP,
			OwnerName: p.OwnerReferences[0].Name,
		})
	}
	for _, element := range slice.Diff(nameList, podMap.Keys()) {
		el := podMap.Get(element)
		pfs := []*PortForwardA{}
		if el != nil && len(el.PFs) > 0 {
			for _, pf := range el.PFs {
				pfs = append(pfs, pf.Copy())
			}
		}
		podMap.Delete(element)
		go tryReForward(pfs)
	}

	Map.Set(nsName, podMap)
	notify <- MapUpdateMsg{}
}

func getPodStatus(p corev1.Pod) string {
	for _, cond := range p.Status.Conditions {
		if cond.Type == corev1.ContainersReady {
			if cond.Status == corev1.ConditionFalse {
				for _, st := range p.Status.ContainerStatuses {
					switch {
					case st.State.Terminated != nil:
						return st.State.Terminated.Reason
					case st.State.Waiting != nil:
						return st.State.Waiting.Reason
					}

				}
				return "Not Ready"
			}
			return "Ready"
		}
	}
	return ""
}

func tryReForward(pfs []*PortForwardA) {
	if len(pfs) == 0 {
		return
	}
	ns := pfs[0].Namespace
	owner := pfs[0].OwnerName
	var pod *Pod
	var service *Service
	// find pod
	for p := range Map.Get(ns).Iter() {
		if strings.HasPrefix(p.Value.Name, owner) {
			pod = p.Value
			break
		}
	}
	for serv := range Services.Get(ns).Iter() {
		if pod == nil {
			return
		}
		if strings.HasPrefix(pod.Name, serv.Value.Name) {
			service = serv.Value
			break
		}
	}
	for _, pf := range pfs {
		if pod == nil {
			return
		}
		pf.Name = pod.Name
		go pf.Forward()
		pod.PFs = append(pod.PFs, pf)
		service.cleanServicePFs()
		service.PFs = append(service.PFs, pf)
	}
}

func fillPorts(p corev1.Pod) (ports []string) {
	for _, c := range p.Spec.Containers {
		for _, port := range c.Ports {
			ports = append(ports, strconv.Itoa(int(port.ContainerPort)))
		}
	}
	return
}
