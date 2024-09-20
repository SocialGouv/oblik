package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	corev1 "k8s.io/api/core/v1"
)

func ApplyRecommendationsToContainers(containers []corev1.Container, requestRecommendations []TargetRecommendation, limitRecommendations []TargetRecommendation, scfg *config.StrategyConfig) *reporting.UpdateResult {
	changes := []reporting.Change{}
	update := reporting.UpdateResult{
		Key: scfg.Key,
	}

	for index, container := range containers {
		var containerRequestRecommendation *TargetRecommendation
		var containerLimitRecommendation *TargetRecommendation
		containerName := container.Name
		for _, cr := range requestRecommendations {
			if cr.ContainerName == containerName {
				containerRequestRecommendation = &cr
				break
			}
		}
		for _, cr := range limitRecommendations {
			if cr.ContainerName == containerName {
				containerLimitRecommendation = &cr
				break
			}
		}
		if containerRequestRecommendation == nil {
			continue
		}

		if container.Resources.Requests == nil {
			container.Resources.Requests = corev1.ResourceList{}
		}
		if container.Resources.Limits == nil {
			container.Resources.Limits = corev1.ResourceList{}
		}

		containerRef := &container

		if containerRequestRecommendation.Cpu != nil {
			changes = setContainerCpuRequest(containerRef, containerRequestRecommendation, changes, scfg)
			changes = setContainerCpuLimit(containerRef, containerRequestRecommendation, containerLimitRecommendation, changes, scfg)
		}

		if containerRequestRecommendation.Memory != nil {
			changes = setContainerMemoryRequest(containerRef, containerRequestRecommendation, changes, scfg)
			changes = setContainerMemoryLimit(containerRef, containerRequestRecommendation, containerLimitRecommendation, changes, scfg)

		}
		containers[index] = *containerRef
	}
	update.Changes = changes
	return &update
}
