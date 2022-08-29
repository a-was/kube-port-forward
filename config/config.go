package config

import (
	"path/filepath"

	"github.com/main-kube/util/env"
	"k8s.io/client-go/util/homedir"
)

var (
	KUBECONFIG_FOLDERS []string

	DNS_PORT = env.Get("DNS_PORT", "5353")
)

func init() {
	KUBECONFIG_FOLDERS = env.Get("KUBECONFIG_FOLDERS", make([]string, 3))
	KUBECONFIG_FOLDERS = append(KUBECONFIG_FOLDERS, []string{filepath.Join(homedir.HomeDir(), ".k3d"), filepath.Join(homedir.HomeDir(), ".kube")}...)
	KUBECONFIG_FOLDERS = append(KUBECONFIG_FOLDERS, "/etc/rancher/k3s")
}
