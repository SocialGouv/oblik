package controller

import (
	"fmt"
	"log"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Run() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building Kubernetes clientset: %s", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building dynamic client: %s", err)
	}

	for {
		if err := watchOpa(clientset, dynamicClient); err != nil {
			fmt.Printf("Error watching OPA: %v\n", err)
			time.Sleep(10 * time.Second) // Wait before retrying
			continue
		}
	}

}
