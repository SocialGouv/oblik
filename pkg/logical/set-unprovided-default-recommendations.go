package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

func SetUnprovidedDefaultRecommendations(containers []corev1.Container, recommendations []TargetRecommendation, scfg *config.StrategyConfig, vpaResource *vpa.VerticalPodAutoscaler) []TargetRecommendation {
	for _, container := range containers {
		containerName := container.Name
		var found bool
		for _, containerRecommendation := range recommendations {
			if containerRecommendation.ContainerName != containerName {
				continue
			}
			found = true
			break
		}
		if !found {
			containerRecommendation := TargetRecommendation{
				ContainerName: containerName,
			}
			switch scfg.GetUnprovidedApplyDefaultRequestCPUSource(containerName) {
			case config.UnprovidedApplyDefaultModeMinAllowed:
				minCpu := findContainerPolicy(vpaResource, containerName).MinAllowed.Cpu()
				if scfg.GetMinAllowedRecommendationCpu(containerName) != nil && (minCpu == nil || minCpu.Cmp(*scfg.GetMinAllowedRecommendationCpu(containerName)) == -1) {
					minCpu = scfg.GetMinAllowedRecommendationCpu(containerName)
				}
				containerRecommendation.Cpu = minCpu
			case config.UnprovidedApplyDefaultModeMaxAllowed:
				maxCpu := findContainerPolicy(vpaResource, containerName).MaxAllowed.Cpu()
				if scfg.GetMaxAllowedRecommendationCpu(containerName) != nil && (maxCpu == nil || maxCpu.Cmp(*scfg.GetMaxAllowedRecommendationCpu(containerName)) == 1) {
					maxCpu = scfg.GetMaxAllowedRecommendationCpu(containerName)
				}
				containerRecommendation.Cpu = maxCpu
			case config.UnprovidedApplyDefaultModeValue:
				value := scfg.GetUnprovidedApplyDefaultRequestCPUValue(containerName)
				cpu, err := resource.ParseQuantity(value)
				if err != nil {
					klog.Warningf("Set unprovided CPU resources, value parsing error: %s. Value was: %s", err.Error(), value)
					break
				}
				containerRecommendation.Cpu = &cpu
			}
			switch scfg.GetUnprovidedApplyDefaultRequestMemorySource(containerName) {
			case config.UnprovidedApplyDefaultModeMinAllowed:
				minMemory := findContainerPolicy(vpaResource, containerName).MinAllowed.Memory()
				if scfg.GetMinAllowedRecommendationMemory(containerName) != nil && (minMemory == nil || minMemory.Cmp(*scfg.GetMinAllowedRecommendationMemory(containerName)) == -1) {
					minMemory = scfg.GetMinAllowedRecommendationMemory(containerName)
				}
				containerRecommendation.Memory = minMemory
			case config.UnprovidedApplyDefaultModeMaxAllowed:
				maxMemory := findContainerPolicy(vpaResource, containerName).MaxAllowed.Memory()
				if scfg.GetMaxAllowedRecommendationMemory(containerName) != nil && (maxMemory == nil || maxMemory.Cmp(*scfg.GetMaxAllowedRecommendationMemory(containerName)) == 1) {
					maxMemory = scfg.GetMaxAllowedRecommendationMemory(containerName)
				}
				containerRecommendation.Memory = maxMemory
			case config.UnprovidedApplyDefaultModeValue:
				value := scfg.GetUnprovidedApplyDefaultRequestMemoryValue(containerName)
				memory, err := resource.ParseQuantity(value)
				if err != nil {
					klog.Warningf("Set unprovided Memory resources, value parsing error: %s. Value was: %s", err.Error(), value)
					break
				}
				containerRecommendation.Memory = &memory
			}

			recommendations = append(recommendations, containerRecommendation)
		}
	}
	return recommendations
}

func findContainerPolicy(vpaResource *vpa.VerticalPodAutoscaler, containerName string) *vpa.ContainerResourcePolicy {
	if vpaResource == nil {
		return nil
	}
	for _, containerPolicy := range vpaResource.Spec.ResourcePolicy.ContainerPolicies {
		if containerPolicy.ContainerName == containerName || containerPolicy.ContainerName == "*" {
			return &containerPolicy
		}
	}
	return nil
}
