package config

import (
	"path/filepath"

	"github.com/main-kube/util/env"
	"k8s.io/client-go/util/homedir"
)

type config struct {
	KUBECONFIG_FOLDERS []string
}

var Config config

func init() {
	Config = config{
		KUBECONFIG_FOLDERS: env.Get("KUBECONFIG_FOLDERS", make([]string, 3)),
	}
	Config.KUBECONFIG_FOLDERS = append(Config.KUBECONFIG_FOLDERS, []string{filepath.Join(homedir.HomeDir(), ".k3d"), filepath.Join(homedir.HomeDir(), ".kube")}...)
	Config.KUBECONFIG_FOLDERS = append(Config.KUBECONFIG_FOLDERS, "/etc/rancher/k3s")
}
