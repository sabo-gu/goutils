package k8s

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/DoOR-Team/goutils/log"
)

func InitK8sClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Error("InClusterConfig Failed:", err, ", try OutClusterConfig")
		config, err = OutCluserConfig()
		if err != nil {
			log.Error("OutCluserConfig Failed:", err)
			return nil
		}
		log.Info("OutCluserConfig Successful")
	} else {
		log.Info("InClusterConfig Successful")
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("NewForConfig Failed:", err)
		return nil
	}
	defer func() {
		if err := recover(); err != nil {
			log.Error("panic error:", err)
			// clientset = nil
		}
	}()
	return clientset
}

func OutCluserConfig() (*rest.Config, error) {
	var kubeoutconfig *string
	if home := homeDir(); home != "" {
		configAddr := filepath.Join(home, ".kube", "config")
		kubeoutconfig = &configAddr
	}

	// use the current context in kubeoutconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeoutconfig)
	return config, err
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
