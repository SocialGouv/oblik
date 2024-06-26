package config

import (
	"fmt"

	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

type VpaContainerCfg struct {
	Key           string
	ContainerName string
	*LoadCfg
}

func createVpaContainerCfg(vpaResource *vpa.VerticalPodAutoscaler, containerName string) *VpaContainerCfg {
	key := fmt.Sprintf("%s/%s", vpaResource.Namespace, vpaResource.Name)
	cfg := &VpaContainerCfg{
		Key:           key,
		ContainerName: containerName,
		LoadCfg: &LoadCfg{
			Key: key,
		},
	}
	loadVpaCommonCfg(cfg.LoadCfg, vpaResource, containerName)
	return cfg
}
