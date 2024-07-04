package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	corev1 "k8s.io/api/core/v1"
)

func applyRecommandationsToContainers(containers []corev1.Container, requestRecommandations []TargetRecommandation, limitRecommandations []TargetRecommandation, vcfg *config.VpaWorkloadCfg) *reporting.UpdateResult {
	changes := []reporting.Change{}
	update := reporting.UpdateResult{
		Key: vcfg.Key,
	}

	for index, container := range containers {
		var containerRequestRecommendation *TargetRecommandation
		var containerLimitRecommendation *TargetRecommandation
		containerName := container.Name
		for _, cr := range requestRecommandations {
			if cr.ContainerName == containerName {
				containerRequestRecommendation = &cr
				break
			}
		}
		for _, cr := range limitRecommandations {
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
			setContainerCpuRequest(containerRef, containerRequestRecommendation, changes, vcfg)
			setContainerCpuLimit(containerRef, containerRequestRecommendation, containerLimitRecommendation, changes, vcfg)
		}

		if containerRequestRecommendation.Memory != nil {
			setContainerMemoryRequest(containerRef, containerRequestRecommendation, changes, vcfg)
			setContainerMemoryLimit(containerRef, containerRequestRecommendation, containerLimitRecommendation, changes, vcfg)

		}
		containers[index] = *containerRef
	}
	update.Changes = changes
	return &update
}
