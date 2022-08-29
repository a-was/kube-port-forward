package kube

import (
	"strconv"
	"sync"
	"time"

	"github.com/main-kube/util/safe"
	"github.com/main-kube/util/slice"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Service struct {
	Name      string
	Namespace string
	Ports     []string
	PFs       []*PortForwardA
}

var Services = safe.NewSortedMap(map[string]*safe.SortedMap[string, *Service]{}, func(data []string, i, j int) bool {
	return data[i] < data[j]
})

func UpdateServiceMap(notify chan any) {
	wg := sync.WaitGroup{}
	for range time.Tick(1 * time.Second) {
		if Client == nil {
			continue
		}
		go cleanServiceMap(namespaces)
		for _, namespace := range namespaces {
			wg.Add(1)
			go func(namespace string) {
				defer wg.Done()
				go addServices(namespace, notify)
			}(namespace)
		}
		wg.Wait()
	}
}
func cleanServiceMap(ns []string) {
	if Services.Len() > 0 {
		for _, key := range slice.Diff(Services.Keys(), ns) {
			Services.Delete(key)
		}
	}
}
func addServices(namespace string, notify chan any) {
	serviceList, err := Client.API.CoreV1().Services(namespace).List(Client.CTX, v1.ListOptions{})
	if err != nil {
		log.Error(err)
	}
	serviceMap, ok := Services.GetFull(namespace)
	if !ok {
		serviceMap = newServiceMap()
	}
	nameList := make([]string, 0, len(serviceList.Items))
	for _, p := range serviceList.Items {
		nameList = append(nameList, p.Name)
		ok := serviceMap.Exists(p.Name)
		if ok {
			// if string(p.Status.Phase) != pod.Status {
			// 	pod.Status = string(p.Status.Phase)
			// 	serviceMap.Set(p.Name, pod)
			// }
			continue
		}
		serviceMap.Set(p.Name, &Service{
			Name:      p.Name,
			Namespace: namespace,
			Ports:     fillSerPorts(p),
		})
	}
	for _, element := range slice.Diff(nameList, serviceMap.Keys()) {
		serviceMap.Delete(element)
	}

	Services.Set(namespace, serviceMap)
	notify <- MapUpdateMsg{}
}

func fillSerPorts(ser corev1.Service) (ports []string) {
	for _, port := range ser.Spec.Ports {
		ports = append(ports, strconv.Itoa(int(port.Port)))
	}
	return
}

func newServiceMap() *safe.SortedMap[string, *Service] {
	return safe.NewSortedMap(map[string]*Service{}, func(data []string, i, j int) bool {
		return data[i] < data[j]
	})
}
