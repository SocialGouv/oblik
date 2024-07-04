package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

func setUnprovidedDefaultRecommandations(containers []corev1.Container, recommandations []TargetRecommandation, vpaResource *vpa.VerticalPodAutoscaler, vcfg *config.VpaWorkloadCfg) []TargetRecommandation {
	for _, container := range containers {
		containerName := container.Name
		var found bool
		for _, containerRecommendation := range recommandations {
			if containerRecommendation.ContainerName != containerName {
				continue
			}
			found = true
			break
		}
		if !found {
			containerRecommandation := TargetRecommandation{
				ContainerName: containerName,
			}
			switch vcfg.GetUnprovidedApplyDefaultRequestCPUSource(containerName) {
			case config.UnprovidedApplyDefaultModeMinAllowed:
				minCpu := findContainerPolicy(vpaResource, containerName).MinAllowed.Cpu()
				if vcfg.GetMinRequestCpu(containerName) != nil && (minCpu == nil || minCpu.Cmp(*vcfg.GetMinRequestCpu(containerName)) == -1) {
					minCpu = vcfg.GetMinRequestCpu(containerName)
				}
				containerRecommandation.Cpu = minCpu
			case config.UnprovidedApplyDefaultModeMaxAllowed:
				maxCpu := findContainerPolicy(vpaResource, containerName).MaxAllowed.Cpu()
				if vcfg.GetMaxRequestCpu(containerName) != nil && (maxCpu == nil || maxCpu.Cmp(*vcfg.GetMaxRequestCpu(containerName)) == 1) {
					maxCpu = vcfg.GetMaxRequestCpu(containerName)
				}
				containerRecommandation.Cpu = maxCpu
			case config.UnprovidedApplyDefaultModeValue:
				cpu, err := resource.ParseQuantity(vcfg.GetUnprovidedApplyDefaultRequestCPUValue(containerName))
				if err != nil {
					klog.Warningf("Set unprovided CPU resources, value parsing error: %s", err.Error())
					break
				}
				containerRecommandation.Cpu = &cpu
			}
			switch vcfg.GetUnprovidedApplyDefaultRequestMemorySource(containerName) {
			case config.UnprovidedApplyDefaultModeMinAllowed:
				minMemory := findContainerPolicy(vpaResource, containerName).MinAllowed.Memory()
				if vcfg.GetMinRequestMemory(containerName) != nil && (minMemory == nil || minMemory.Cmp(*vcfg.GetMinRequestMemory(containerName)) == -1) {
					minMemory = vcfg.GetMinRequestMemory(containerName)
				}
				containerRecommandation.Memory = minMemory
			case config.UnprovidedApplyDefaultModeMaxAllowed:
				maxMemory := findContainerPolicy(vpaResource, containerName).MaxAllowed.Memory()
				if vcfg.GetMaxRequestMemory(containerName) != nil && (maxMemory == nil || maxMemory.Cmp(*vcfg.GetMaxRequestMemory(containerName)) == 1) {
					maxMemory = vcfg.GetMaxRequestMemory(containerName)
				}
				containerRecommandation.Memory = maxMemory
			case config.UnprovidedApplyDefaultModeValue:
				memory, err := resource.ParseQuantity(vcfg.GetUnprovidedApplyDefaultRequestMemoryValue(containerName))
				if err != nil {
					klog.Warningf("Set unprovided Memory resources, value parsing error: %s", err.Error())
					break
				}
				containerRecommandation.Memory = &memory
			}

			recommandations = append(recommandations, containerRecommandation)
		}
	}
	return recommandations
}

func findContainerPolicy(vpaResource *vpa.VerticalPodAutoscaler, containerName string) *vpa.ContainerResourcePolicy {
	for _, containerPolicy := range vpaResource.Spec.ResourcePolicy.ContainerPolicies {
		if containerPolicy.ContainerName == containerName || containerPolicy.ContainerName == "*" {
			return &containerPolicy
		}
	}
	return nil
}
