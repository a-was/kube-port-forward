package kube

import (
	"fmt"
	"net/http"
	"os"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PodPortForwardA struct {
	KubeClient *ClientS

	LocalPort int
	// PodPort is the target port for the pod
	PodPort   int
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

func (pod *Pod) Forward(pf *PodPortForwardA) error {
	pf.KubeClient = Client
	pf.stopCh = make(chan struct{}, 1)
	pf.readyCh = make(chan struct{})
	pf.streams = genericclioptions.IOStreams{
		Out:    out,
		ErrOut: out,
		In:     os.Stdin,
	}

	url := pf.KubeClient.API.RESTClient().Post().Resource("pods").Namespace(pod.Namespace).Name(pod.Name).SubResource("portforward").Prefix("/api/v1").URL()
	transport, upgrader, err := spdy.RoundTripperFor(pf.KubeClient.Config)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", pf.LocalPort, pf.PodPort)}, pf.stopCh, pf.readyCh, pf.streams.Out, pf.streams.ErrOut)
	if err != nil {
		return err
	}

	// if pod.PodPortForwardA != nil {
	// 	pod.PodPortForwardA.Close()
	// }

	if err := fw.ForwardPorts(); err != nil {
		pf = nil
		return err
	}
	return nil
}

func (req *PodPortForwardA) Close() {
	if req == nil {
		return
	}
	close(req.stopCh)
}

// TODO fix this
func (req *PodPortForwardA) Ready() {
	<-req.readyCh
}
