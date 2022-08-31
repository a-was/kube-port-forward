package kube

import (
	"context"
	"fmt"

	"github.com/main-kube/util/safe"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var Endpoints = safe.NewSortedMap(map[string]*Endpoint{}, func(data []string, i, j int) bool {
	return data[i] < data[j]
})

type Endpoint struct {
	Name      string
	Namespace string
	KubePort  int
	HostPort  int
	Addr      string
}

func discover() {
	services, err := Client.API.CoreV1().Services("").List(Client.CTX, metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}
	for _, service := range services.Items {
		if service.Annotations["dev-port"] != "" {
			endpoint, err := Client.API.CoreV1().Endpoints(service.Namespace).Get(context.TODO(), service.Name, metav1.GetOptions{})
			if err != nil {
				fmt.Println(err)
			}

			Endpoints.Set(service.Name, &Endpoint{
				Name:      service.Name,
				Namespace: service.Namespace,
				KubePort:  int(service.Spec.Ports[0].Port),
				HostPort:  int(service.Spec.Ports[0].Port),
				Addr:      endpoint.Subsets[0].Addresses[0].IP,
			})
		}
	}
}

func (end *Endpoint) CreateService() error {
	if err := end.createService(); err != nil {
		return err
	}
	if err := end.updateEndpoint(); err != nil {
		return err
	}
	Endpoints.Set(end.Name, end)
	return nil
}

func DeleteEndpoint(name string) {
	end, ok := Endpoints.GetFull(name)
	if !ok {
		return
	}
	if err := Client.API.CoreV1().Services(end.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		log.Error(err)
		return
	}
	Endpoints.Delete(name)
}

func (end *Endpoint) createService() error {
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      end.Name,
			Namespace: end.Namespace,
			Annotations: map[string]string{
				"dev-port": "true",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     end.Name,
					Protocol: "TCP",
					Port:     int32(end.KubePort),
				},
			},
			ClusterIP: "None",
		},
	}
	// create service
	if _, err := Client.API.CoreV1().Services(end.Namespace).Create(context.TODO(), service, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (end *Endpoint) updateEndpoint() error {
	endpoint := &v1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Endpoints",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      end.Name,
			Namespace: end.Namespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: end.Addr,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     end.Name,
						Port:     int32(end.HostPort),
						Protocol: "TCP",
					},
				},
			},
		},
	}
	if _, err := Client.API.CoreV1().Endpoints(end.Namespace).Update(context.TODO(), endpoint, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func (end Endpoint) CheckServiceExists() bool {
	if _, ok := Endpoints.GetFull(end.Name); ok {
		log.Info("Service already exists: ", ok)
		return true
	}
	for element := range Endpoints.Iter() {
		if element.Value.HostPort == end.HostPort {
			return true
		}

	}
	return false
}
