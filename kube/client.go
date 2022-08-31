package kube

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fr-str/itsy-bitsy-teenie-weenie-port-forwarder-programini/config"

	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	ctrl "sigs.k8s.io/controller-runtime"
	crd "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log        *zap.SugaredLogger
	Client     *ClientS
	namespaces []string
)

// ClientS ...
type ClientS struct {
	Config  *restclient.Config
	API     *kubernetes.Clientset
	CRD     crd.Client
	CTX     context.Context
	Metrics *metricsv.Clientset

	IngressOldVersion bool
}

func Connect(configName string) {
	log = zap.S()

	var err error
	Client, err = newClient(findConfig(configName))
	if err != nil {
		log.Fatal(err)
		return
	}
	go getNamespaces()
	go discover()
}

func findConfig(configName string) (kConfig []byte) {
	log = zap.S()
	var err error
	if configName[0] == '/' {
		kConfig, err = os.ReadFile(configName)
		if err != nil {
			log.Error(err)
		}
		return
	}
	if configName[0] == '~' {
		log.Debug(os.ExpandEnv(configName))
		kConfig, err = os.ReadFile(os.ExpandEnv(configName))
		if err != nil {
			log.Error(err)
		}
		return
	}
	for _, v := range config.KUBECONFIG_FOLDERS {
		if v == "" {
			continue
		}
		files, err := os.ReadDir(v)
		if err != nil {
			log.Error(err)
		}
		for _, filed := range files {
			if filed.IsDir() {
				continue
			}
			if strings.HasPrefix(filed.Name(), configName) {
				kConfig, err = os.ReadFile(filepath.Join(v, filed.Name()))
				if err != nil {
					log.Error(err)
				}
				return
			}

		}

	}
	fmt.Printf("Config '%s' not found in specified folders %v", configName, config.KUBECONFIG_FOLDERS)
	os.Exit(1)
	return
}

func newClient(config []byte) (client *ClientS, err error) {

	kube := new(ClientS)

	if len(config) == 0 {
		kube.Config, err = ctrl.GetConfig()

	} else {
		kube.Config, err = clientcmd.RESTConfigFromKubeConfig(config)
	}

	if err != nil {
		return nil, err
	}

	kube.Config.RateLimiter = nil
	kube.Config.QPS = 1000
	kube.Config.Burst = 2000

	kube.API, err = kubernetes.NewForConfig(kube.Config)
	if err != nil {
		return nil, err
	}

	kube.CRD, err = crd.New(kube.Config, crd.Options{})
	if err != nil {
		return nil, err
	}

	kube.Metrics, err = metricsv.NewForConfig(kube.Config)
	if err != nil {
		return nil, err
	}

	kube.CTX = context.TODO()

	return kube, nil
}

func getNamespaces() {
	for range time.Tick(time.Second) {

		ns, _ := Client.API.CoreV1().Namespaces().List(Client.CTX, v1.ListOptions{})
		nsl := make([]string, 0, len(ns.Items))
		for _, n := range ns.Items {
			nsl = append(nsl, n.Name)
		}
		namespaces = nsl
	}
}
