package kube

import (
	"fmt"
	"net/http"
	"os"

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
	out, _ = os.OpenFile("/tmp/ibtwpfp-portforward-log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	os.Stderr = out
}

func (pf *PortForwardA) Forward() {

	pf.KubeClient = Client
	pf.stopCh = make(chan struct{}, 1)
	pf.readyCh = make(chan struct{})
	pf.streams = genericclioptions.IOStreams{
		Out:    out,
		ErrOut: out,
		In:     os.Stdin,
	}
	log.Info("PortForward")
	log.Info(pf.Resource)

	url := pf.KubeClient.API.RESTClient().Post().Resource(pf.Resource).Namespace(pf.Namespace).Name(pf.Name).SubResource("portforward").Prefix("/api/v1").URL()
	log.Info(url)
	transport, upgrader, err := spdy.RoundTripperFor(pf.KubeClient.Config)
	if err != nil {
		log.Error(err)
		fmt.Fprintln(out, err)
		return
	}
	fmt.Fprintln(out, "PortForward started")

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", pf.LocalPort, pf.KubePort)}, pf.stopCh, pf.readyCh, pf.streams.Out, pf.streams.ErrOut)
	if err != nil {
		fmt.Fprintln(out, err)
		log.Error(err)
		return
	}

	// if pod.PodPortForwardA != nil {
	// 	pod.PodPortForwardA.Close()
	// }

	if err := fw.ForwardPorts(); err != nil {
		pf = nil
		fmt.Fprintln(out, err)
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
