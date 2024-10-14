package client

import (
	"github.com/SocialGouv/oblik/pkg/config"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type KubeClients struct {
	Clientset     *kubernetes.Clientset
	DynamicClient *dynamic.DynamicClient
	VpaClientset  *vpaclientset.Clientset
	RestConfig    *rest.Config
}

func NewKubeClients() *KubeClients {

	conf, err := config.LoadKubeConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		klog.Fatalf("Error creating Kubernetes client: %s", err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(conf)
	if err != nil {
		klog.Fatalf("Error creating dynamic client: %s", err.Error())
	}

	vpaClientset, err := vpaclientset.NewForConfig(conf)
	if err != nil {
		klog.Fatalf("Error creating VPA client: %s", err.Error())
	}

	kubeClients := &KubeClients{
		Clientset:     clientset,
		DynamicClient: dynamicClient,
		VpaClientset:  vpaClientset,
		RestConfig:    conf,
	}

	return kubeClients
}
