package config

import (
	"fmt"
)

type ContainerConfig struct {
	Key           string
	ContainerName string
	*LoadCfg
}

func createContainerConfig(annotable Annotatable, containerName string) *ContainerConfig {
	key := fmt.Sprintf("%s/%s", annotable.GetNamespace(), annotable.GetName())
	cfg := &ContainerConfig{
		Key:           key,
		ContainerName: containerName,
		LoadCfg: &LoadCfg{
			Key: key,
		},
	}
	loadAnnotableCommonCfg(cfg.LoadCfg, annotable, containerName)
	return cfg
}
