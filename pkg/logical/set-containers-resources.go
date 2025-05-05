package logical

import (
	"github.com/SocialGouv/oblik/pkg/calculator"
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

func setContainerCpuRequest(container *corev1.Container, containerRequestRecommendation *TargetRecommendation, changes []reporting.Change, scfg *config.StrategyConfig) []reporting.Change {
	containerName := container.Name
	cpuRequest := *container.Resources.Requests.Cpu()

	// Check if a direct CPU request value is specified
	if scfg.GetRequestCpuValue(containerName) != nil {
		directCpuRequest, err := resource.ParseQuantity(*scfg.GetRequestCpuValue(containerName))
		if err == nil {
			newCPURequest := directCpuRequest
			if scfg.GetRequestCPUApplyMode(containerName) == config.ApplyModeEnforce && newCPURequest.Cmp(cpuRequest) != 0 {
				changes = append(changes, reporting.Change{
					Old:           cpuRequest,
					New:           newCPURequest,
					Type:          reporting.UpdateTypeCpuRequest,
					ContainerName: containerName,
				})
				container.Resources.Requests[corev1.ResourceCPU] = newCPURequest
			}
			return changes
		} else {
			klog.Warningf("Error parsing direct CPU request value: %s, error: %s", *scfg.GetRequestCpuValue(containerName), err.Error())
		}
	}

	// If no direct value is specified, use the VPA recommendation
	newCPURequest := *containerRequestRecommendation.Cpu
	if scfg.GetMinAllowedRecommendationCpu(containerName) != nil && newCPURequest.Cmp(*scfg.GetMinAllowedRecommendationCpu(containerName)) == -1 {
		newCPURequest = *scfg.GetMinAllowedRecommendationCpu(containerName)
	}
	if scfg.GetMaxAllowedRecommendationCpu(containerName) != nil && newCPURequest.Cmp(*scfg.GetMaxAllowedRecommendationCpu(containerName)) == 1 {
		newCPURequest = *scfg.GetMaxAllowedRecommendationCpu(containerName)
	}

	newCPURequest = calculator.CalculateResourceValue(newCPURequest, scfg.GetIncreaseRequestCpuAlgo(containerName), scfg.GetIncreaseRequestCpuValue(containerName), calculator.ResourceTypeCPU)

	if scfg.GetMinRequestCpu(containerName) != nil && newCPURequest.Cmp(*scfg.GetMinRequestCpu(containerName)) == -1 {
		newCPURequest = *scfg.GetMinRequestCpu(containerName)
	}
	if scfg.GetMaxRequestCpu(containerName) != nil && newCPURequest.Cmp(*scfg.GetMaxRequestCpu(containerName)) == 1 {
		newCPURequest = *scfg.GetMaxRequestCpu(containerName)
	}

	minDiffCpuRequest := calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceCPU], scfg.GetMinDiffCpuRequestAlgo(containerName), scfg.GetMinDiffCpuRequestValue(containerName), calculator.ResourceTypeCPU)
	if newCPURequest.Cmp(minDiffCpuRequest) == -1 {
		newCPURequest = cpuRequest
	}
	if scfg.GetRequestCpuScaleDirection(containerName) == config.ScaleDirectionDown && newCPURequest.Cmp(cpuRequest) == 1 {
		newCPURequest = cpuRequest
	}
	if scfg.GetRequestCpuScaleDirection(containerName) == config.ScaleDirectionUp && newCPURequest.Cmp(cpuRequest) == -1 {
		newCPURequest = cpuRequest
	}
	if scfg.GetRequestCPUApplyMode(containerName) == config.ApplyModeEnforce && newCPURequest.Cmp(cpuRequest) != 0 {
		changes = append(changes, reporting.Change{
			Old:           cpuRequest,
			New:           newCPURequest,
			Type:          reporting.UpdateTypeCpuRequest,
			ContainerName: containerName,
		})
		container.Resources.Requests[corev1.ResourceCPU] = newCPURequest
	}
	return changes
}

func setContainerCpuLimit(container *corev1.Container, containerRequestRecommendation *TargetRecommendation, containerLimitRecommendation *TargetRecommendation, changes []reporting.Change, scfg *config.StrategyConfig) []reporting.Change {
	containerName := container.Name
	cpuLimit := *container.Resources.Limits.Cpu()

	// Check if a direct CPU limit value is specified
	if scfg.GetLimitCpuValue(containerName) != nil {
		directCpuLimit, err := resource.ParseQuantity(*scfg.GetLimitCpuValue(containerName))
		if err == nil {
			newCPULimit := directCpuLimit
			if scfg.GetLimitCPUApplyMode(containerName) == config.ApplyModeEnforce && newCPULimit.Cmp(cpuLimit) != 0 {
				changes = append(changes, reporting.Change{
					Old:           cpuLimit,
					New:           newCPULimit,
					Type:          reporting.UpdateTypeCpuLimit,
					ContainerName: containerName,
				})
				container.Resources.Limits[corev1.ResourceCPU] = newCPULimit
			}
			return changes
		} else {
			klog.Warningf("Error parsing direct CPU limit value: %s, error: %s", *scfg.GetLimitCpuValue(containerName), err.Error())
		}
	}

	// If no direct value is specified, use the VPA recommendation or calculator
	var newCPULimit resource.Quantity
	if scfg.GetLimitCpuApplyTarget(containerName) == config.LimitApplyTargetAuto {
		newCPULimit = calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceCPU], scfg.GetLimitCPUCalculatorAlgo(containerName), scfg.GetLimitCPUCalculatorValue(containerName), calculator.ResourceTypeCPU)
	} else {
		newCPULimit = *containerLimitRecommendation.Cpu
	}

	if scfg.GetMinLimitCpu(containerName) != nil && newCPULimit.Cmp(*scfg.GetMinLimitCpu(containerName)) == -1 {
		newCPULimit = *scfg.GetMinLimitCpu(containerName)
	}
	if scfg.GetMaxLimitCpu(containerName) != nil && newCPULimit.Cmp(*scfg.GetMaxLimitCpu(containerName)) == 1 {
		newCPULimit = *scfg.GetMaxLimitCpu(containerName)
	}

	if newCPULimit.Cmp(container.Resources.Requests[corev1.ResourceCPU]) == -1 {
		newCPULimit = container.Resources.Requests[corev1.ResourceCPU]
	}

	minDiffCpuLimit := calculator.CalculateResourceValue(container.Resources.Limits[corev1.ResourceCPU], scfg.GetMinDiffCpuLimitAlgo(containerName), scfg.GetMinDiffCpuLimitValue(containerName), calculator.ResourceTypeCPU)
	if newCPULimit.Cmp(minDiffCpuLimit) == -1 {
		newCPULimit = cpuLimit
	}
	if scfg.GetLimitCpuScaleDirection(containerName) == config.ScaleDirectionDown && newCPULimit.Cmp(cpuLimit) == 1 {
		newCPULimit = cpuLimit
	}
	if scfg.GetLimitCpuScaleDirection(containerName) == config.ScaleDirectionUp && newCPULimit.Cmp(cpuLimit) == -1 {
		newCPULimit = cpuLimit
	}
	if scfg.GetLimitCPUApplyMode(containerName) == config.ApplyModeEnforce && newCPULimit.Cmp(cpuLimit) != 0 {
		changes = append(changes, reporting.Change{
			Old:           cpuLimit,
			New:           newCPULimit,
			Type:          reporting.UpdateTypeCpuLimit,
			ContainerName: containerName,
		})
		container.Resources.Limits[corev1.ResourceCPU] = newCPULimit
	}
	return changes
}

func setContainerMemoryRequest(container *corev1.Container, containerRequestRecommendation *TargetRecommendation, changes []reporting.Change, scfg *config.StrategyConfig) []reporting.Change {
	containerName := container.Name
	memoryRequest := *container.Resources.Requests.Memory()

	// Check if a direct memory request value is specified
	if scfg.GetRequestMemoryValue(containerName) != nil {
		directMemoryRequest, err := resource.ParseQuantity(*scfg.GetRequestMemoryValue(containerName))
		if err == nil {
			newMemoryRequest := directMemoryRequest
			if scfg.GetRequestMemoryApplyMode(containerName) == config.ApplyModeEnforce && newMemoryRequest.Cmp(memoryRequest) != 0 {
				changes = append(changes, reporting.Change{
					Old:           memoryRequest,
					New:           newMemoryRequest,
					Type:          reporting.UpdateTypeMemoryRequest,
					ContainerName: containerName,
				})
				container.Resources.Requests[corev1.ResourceMemory] = newMemoryRequest
			}
			return changes
		} else {
			klog.Warningf("Error parsing direct memory request value: %s, error: %s", *scfg.GetRequestMemoryValue(containerName), err.Error())
		}
	}

	// If no direct value is specified, use the VPA recommendation or calculator
	var newMemoryRequest resource.Quantity
	if scfg.GetMemoryRequestFromCpuEnabled(containerName) {
		memoryFromCpu := calculator.CalculateCpuToMemory(container.Resources.Requests[corev1.ResourceCPU])
		newMemoryRequest = calculator.CalculateResourceValue(memoryFromCpu, scfg.GetMemoryRequestFromCpuAlgo(containerName), scfg.GetMemoryRequestFromCpuValue(containerName), calculator.ResourceTypeMemory)
	} else {
		newMemoryRequest = *containerRequestRecommendation.Memory
		if scfg.GetMinAllowedRecommendationMemory(containerName) != nil && newMemoryRequest.Cmp(*scfg.GetMinAllowedRecommendationMemory(containerName)) == -1 {
			newMemoryRequest = *scfg.GetMinAllowedRecommendationMemory(containerName)
		}
		if scfg.GetMaxAllowedRecommendationMemory(containerName) != nil && newMemoryRequest.Cmp(*scfg.GetMaxAllowedRecommendationMemory(containerName)) == 1 {
			newMemoryRequest = *scfg.GetMaxAllowedRecommendationMemory(containerName)
		}
		newMemoryRequest = calculator.CalculateResourceValue(newMemoryRequest, scfg.GetIncreaseRequestMemoryAlgo(containerName), scfg.GetIncreaseRequestMemoryValue(containerName), calculator.ResourceTypeMemory)
	}
	if scfg.GetMinRequestMemory(containerName) != nil && newMemoryRequest.Cmp(*scfg.GetMinRequestMemory(containerName)) == -1 {
		newMemoryRequest = *scfg.GetMinRequestMemory(containerName)
	}
	if scfg.GetMaxRequestMemory(containerName) != nil && newMemoryRequest.Cmp(*scfg.GetMaxRequestMemory(containerName)) == 1 {
		newMemoryRequest = *scfg.GetMaxRequestMemory(containerName)
	}
	minDiffMemoryRequest := calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceMemory], scfg.GetMinDiffMemoryRequestAlgo(containerName), scfg.GetMinDiffMemoryRequestValue(containerName), calculator.ResourceTypeMemory)
	if newMemoryRequest.Cmp(minDiffMemoryRequest) == -1 {
		newMemoryRequest = memoryRequest
	}
	if scfg.GetRequestMemoryScaleDirection(containerName) == config.ScaleDirectionDown && newMemoryRequest.Cmp(memoryRequest) == 1 {
		newMemoryRequest = memoryRequest
	}
	if scfg.GetRequestMemoryScaleDirection(containerName) == config.ScaleDirectionUp && newMemoryRequest.Cmp(memoryRequest) == -1 {
		newMemoryRequest = memoryRequest
	}
	if scfg.GetRequestMemoryApplyMode(containerName) == config.ApplyModeEnforce && newMemoryRequest.Cmp(memoryRequest) != 0 {
		changes = append(changes, reporting.Change{
			Old:           memoryRequest,
			New:           newMemoryRequest,
			Type:          reporting.UpdateTypeMemoryRequest,
			ContainerName: containerName,
		})
		container.Resources.Requests[corev1.ResourceMemory] = newMemoryRequest
	}
	return changes
}

func setContainerMemoryLimit(container *corev1.Container, containerRequestRecommendation *TargetRecommendation, containerLimitRecommendation *TargetRecommendation, changes []reporting.Change, scfg *config.StrategyConfig) []reporting.Change {
	containerName := container.Name
	memoryLimit := *container.Resources.Limits.Memory()

	// Check if a direct memory limit value is specified
	if scfg.GetLimitMemoryValue(containerName) != nil {
		directMemoryLimit, err := resource.ParseQuantity(*scfg.GetLimitMemoryValue(containerName))
		if err == nil {
			newMemoryLimit := directMemoryLimit
			if scfg.GetLimitMemoryApplyMode(containerName) == config.ApplyModeEnforce && newMemoryLimit.Cmp(memoryLimit) != 0 {
				changes = append(changes, reporting.Change{
					Old:           memoryLimit,
					New:           newMemoryLimit,
					Type:          reporting.UpdateTypeMemoryLimit,
					ContainerName: containerName,
				})
				container.Resources.Limits[corev1.ResourceMemory] = newMemoryLimit
			}
			return changes
		} else {
			klog.Warningf("Error parsing direct memory limit value: %s, error: %s", *scfg.GetLimitMemoryValue(containerName), err.Error())
		}
	}

	// If no direct value is specified, use the VPA recommendation or calculator
	var newMemoryLimit resource.Quantity
	if scfg.GetMemoryLimitFromCpuEnabled(containerName) {
		memoryFromCpu := calculator.CalculateCpuToMemory(container.Resources.Limits[corev1.ResourceCPU])
		newMemoryLimit = calculator.CalculateResourceValue(memoryFromCpu, scfg.GetMemoryLimitFromCpuAlgo(containerName), scfg.GetMemoryLimitFromCpuValue(containerName), calculator.ResourceTypeMemory)
	} else {
		if scfg.GetLimitMemoryApplyTarget(containerName) == config.LimitApplyTargetAuto {
			newMemoryLimit = calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceMemory], scfg.GetLimitMemoryCalculatorAlgo(containerName), scfg.GetLimitMemoryCalculatorValue(containerName), calculator.ResourceTypeMemory)
		} else {
			newMemoryLimit = *containerLimitRecommendation.Memory
		}
	}
	if scfg.GetMinLimitMemory(containerName) != nil && newMemoryLimit.Cmp(*scfg.GetMinLimitMemory(containerName)) == -1 {
		newMemoryLimit = *scfg.GetMinLimitMemory(containerName)
	}
	if scfg.GetMaxLimitMemory(containerName) != nil && newMemoryLimit.Cmp(*scfg.GetMaxLimitMemory(containerName)) == 1 {
		newMemoryLimit = *scfg.GetMaxLimitMemory(containerName)
	}

	if newMemoryLimit.Cmp(container.Resources.Requests[corev1.ResourceMemory]) == -1 {
		newMemoryLimit = container.Resources.Requests[corev1.ResourceMemory]
	}

	minDiffMemoryLimit := calculator.CalculateResourceValue(container.Resources.Limits[corev1.ResourceMemory], scfg.GetMinDiffMemoryLimitAlgo(containerName), scfg.GetMinDiffMemoryLimitValue(containerName), calculator.ResourceTypeMemory)
	if newMemoryLimit.Cmp(minDiffMemoryLimit) == -1 {
		newMemoryLimit = memoryLimit
	}
	if scfg.GetLimitMemoryScaleDirection(containerName) == config.ScaleDirectionDown && newMemoryLimit.Cmp(memoryLimit) == 1 {
		newMemoryLimit = memoryLimit
	}
	if scfg.GetLimitMemoryScaleDirection(containerName) == config.ScaleDirectionUp && newMemoryLimit.Cmp(memoryLimit) == -1 {
		newMemoryLimit = memoryLimit
	}
	if scfg.GetLimitMemoryApplyMode(containerName) == config.ApplyModeEnforce && newMemoryLimit.Cmp(memoryLimit) != 0 {
		changes = append(changes, reporting.Change{
			Old:           memoryLimit,
			New:           newMemoryLimit,
			Type:          reporting.UpdateTypeMemoryLimit,
			ContainerName: containerName,
		})
		container.Resources.Limits[corev1.ResourceMemory] = newMemoryLimit
	}

	return changes
}
