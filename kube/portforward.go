package kube

import (
	"context"
	"fmt"
	"net/http"
	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwardA struct {
	KubeClient *ClientS

	Name      string
	Namespace string
	LocalPort int
	// KubePort is the target port for the pod
	KubePort  int
	Resource  string
	Condition bool

	// Steams configures where to write or read input from
	streams genericclioptions.IOStreams
	// stopCh is the channel used to manage the port forward lifecycle
	stopCh chan struct{}
	// readyCh communicates when the tunnel is ready to receive traffic
	readyCh chan struct{}
}

var out *os.File

func init() {
	// out, _ = os.OpenFile("/tmp/ibtwpfp-portforward-log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	out, _ = os.OpenFile("logk", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	os.Stderr = out
}

func (pf *PortForwardA) Forward(notify chan any) {

	pf.KubeClient = Client
	pf.stopCh = make(chan struct{}, 1)
	pf.readyCh = make(chan struct{})
	pf.streams = genericclioptions.IOStreams{
		Out:    out,
		ErrOut: out,
		In:     os.Stdin,
	}
	if pf.Resource == "services" {
		pf.Resource = "pods"
		if err := pf.getFirstPod(); err != nil {
			notify <- err
			log.Error(err)
		}
	}

	url := pf.KubeClient.API.RESTClient().Post().Resource(pf.Resource).Namespace(pf.Namespace).Name(pf.Name).SubResource("portforward").Prefix("/api/v1").URL()
	transport, upgrader, err := spdy.RoundTripperFor(pf.KubeClient.Config)
	if err != nil {
		notify <- err
		log.Error(err)
		return
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", pf.LocalPort, pf.KubePort)}, pf.stopCh, pf.readyCh, pf.streams.Out, pf.streams.ErrOut)
	if err != nil {
		notify <- err
		log.Error(err)
		return
	}

	if err := fw.ForwardPorts(); err != nil {
		pf = nil
		notify <- err
		log.Error(err)
		return
	}

}

func (req *PortForwardA) Close() {
	if req == nil {
		return
	}
	close(req.stopCh)
}

// TODO fix this
func (req *PortForwardA) Ready() {
	<-req.readyCh
}

func (req *PortForwardA) getFirstPod() error {
	serv, err := Client.API.CoreV1().Services(req.Namespace).Get(context.TODO(), req.Name, v1.GetOptions{})
	if err != nil {
		return err
	}
	var selector string
	for k, v := range serv.Spec.Selector {
		selector = k + "=" + v
		break
	}

	pods, err := Client.API.CoreV1().Pods(req.Namespace).List(Client.CTX, v1.ListOptions{
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
	req.Name = pod.Name
	return nil
}
