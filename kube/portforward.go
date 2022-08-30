package kube

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/main-kube/util/slice"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwardA struct {
	KubeClient *ClientS

	Name        string
	ServiceName string
	Namespace   string
	LocalPort   int
	// KubePort is the target port for the pod
	KubePort  int
	Resource  string
	Condition bool
	OwnerName string

	// Steams configures where to write or read input from
	streams genericclioptions.IOStreams
	// stopCh is the channel used to manage the port forward lifecycle
	stopCh chan struct{}
	// readyCh communicates when the tunnel is ready to receive traffic
	readyCh chan struct{}

	Notify chan any
}

var out *os.File

func init() {
	// out, _ = os.OpenFile("/tmp/ibtwpfp-portforward-log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	out, _ = os.OpenFile("logk", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	os.Stderr = out
}

func (pf *PortForwardA) Forward() {

	pf.KubeClient = Client
	pf.stopCh = make(chan struct{})
	pf.readyCh = make(chan struct{})
	pf.streams = genericclioptions.IOStreams{
		Out:    out,
		ErrOut: out,
		In:     os.Stdin,
	}
	if pf.Resource == "services" {
		pf.Resource = "pods"
		if err := pf.getFirstPod(); err != nil {
			pf.Notify <- err
			log.Error(err)
			return
		}
	}

	url := pf.KubeClient.API.RESTClient().Post().Resource(pf.Resource).Namespace(pf.Namespace).Name(pf.Name).SubResource("portforward").Prefix("/api/v1").URL()
	transport, upgrader, err := spdy.RoundTripperFor(pf.KubeClient.Config)
	if err != nil {
		pf.Notify <- err
		log.Error(err)
		return
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", pf.LocalPort, pf.KubePort)}, pf.stopCh, pf.readyCh, pf.streams.Out, pf.streams.ErrOut)
	if err != nil {
		pf.Notify <- err
		log.Error(err)
		return
	}

	if err := fw.ForwardPorts(); err != nil {
		pf.Notify <- err
		log.Error(err)
		if strings.Contains(err.Error(), "pod not found") {
			go pf.Forward()
		}
		return
	}

}

func (pf *PortForwardA) Close() {
	if pf == nil {
		return
	}
	select {
	case pf.stopCh <- struct{}{}:
	default:
	}
	slice.Remove(&Map.Get(pf.Namespace).Get(pf.Name).PFs, pf)
	if Services.Get(pf.Namespace).Get(pf.ServiceName) != nil {
		slice.Remove(&Services.Get(pf.Namespace).Get(pf.ServiceName).PFs, pf)
	}
}

func (pf *PortForwardA) Ready() {
	<-pf.readyCh
}

func (pf *PortForwardA) getFirstPod() error {
	serv, err := Client.API.CoreV1().Services(pf.Namespace).Get(context.TODO(), pf.Name, v1.GetOptions{})
	if err != nil {
		return err
	}
	var selector string
	for k, v := range serv.Spec.Selector {
		selector = k + "=" + v
		break
	}

	pods, err := Client.API.CoreV1().Pods(pf.Namespace).List(Client.CTX, v1.ListOptions{
		LabelSelector: selector,
		Limit:         1,
	})
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("Service has no pods")
	}
	pod := pods.Items[0]
	pf.Name = pod.Name
	pf.OwnerName = pod.ObjectMeta.OwnerReferences[0].Name
	podm := Map.Get(pf.Namespace).Get(pf.Name)
	podm.PFs = append(podm.PFs, pf)
	return nil
}

func (pf *PortForwardA) Copy() *PortForwardA {
	return &PortForwardA{
		Name:        pf.Name,
		ServiceName: pf.ServiceName,
		Namespace:   pf.Namespace,
		LocalPort:   pf.LocalPort,
		KubePort:    pf.KubePort,
		Resource:    pf.Resource,
		Condition:   false,
		OwnerName:   pf.OwnerName,
		Notify:      pf.Notify,
	}
}
