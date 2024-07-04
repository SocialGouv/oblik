package logical

import (
	"github.com/SocialGouv/oblik/pkg/calculator"
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func setContainerCpuRequest(container *corev1.Container, containerRequestRecommendation *TargetRecommandation, changes []reporting.Change, vcfg *config.VpaWorkloadCfg) []reporting.Change {
	containerName := container.Name
	cpuRequest := *container.Resources.Requests.Cpu()
	newCPURequest := calculator.CalculateResourceValue(*containerRequestRecommendation.Cpu, vcfg.GetIncreaseRequestCpuAlgo(containerName), vcfg.GetIncreaseRequestCpuValue(containerName))
	if vcfg.GetMinRequestCpu(containerName) != nil && newCPURequest.Cmp(*vcfg.GetMinRequestCpu(containerName)) == -1 {
		newCPURequest = *vcfg.GetMinRequestCpu(containerName)
	}
	if vcfg.GetMaxRequestCpu(containerName) != nil && newCPURequest.Cmp(*vcfg.GetMaxRequestCpu(containerName)) == 1 {
		newCPURequest = *vcfg.GetMaxRequestCpu(containerName)
	}
	minDiffCpuRequest := calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceCPU], vcfg.GetMinDiffCpuRequestAlgo(containerName), vcfg.GetMinDiffCpuRequestValue(containerName))
	if newCPURequest.Cmp(minDiffCpuRequest) == -1 {
		newCPURequest = cpuRequest
	}
	if vcfg.GetRequestCpuScaleDirection(containerName) == config.ScaleDirectionDown && newCPURequest.Cmp(cpuRequest) == 1 {
		newCPURequest = cpuRequest
	}
	if vcfg.GetRequestCpuScaleDirection(containerName) == config.ScaleDirectionUp && newCPURequest.Cmp(cpuRequest) == -1 {
		newCPURequest = cpuRequest
	}
	if vcfg.GetRequestCPUApplyMode(containerName) == config.ApplyModeEnforce && newCPURequest.String() != cpuRequest.String() {
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

func setContainerCpuLimit(container *corev1.Container, containerRequestRecommendation *TargetRecommandation, containerLimitRecommendation *TargetRecommandation, changes []reporting.Change, vcfg *config.VpaWorkloadCfg) []reporting.Change {
	containerName := container.Name
	cpuLimit := *container.Resources.Limits.Cpu()

	var newCPULimit resource.Quantity
	if vcfg.GetLimitCpuApplyTarget(containerName) == config.LimitApplyTargetAuto {
		newCPULimit = calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceCPU], vcfg.GetLimitCPUCalculatorAlgo(containerName), vcfg.GetLimitCPUCalculatorValue(containerName))
	} else {
		newCPULimit = *containerLimitRecommendation.Cpu
	}

	if vcfg.GetMinLimitCpu(containerName) != nil && newCPULimit.Cmp(*vcfg.GetMinLimitCpu(containerName)) == -1 {
		newCPULimit = *vcfg.GetMinLimitCpu(containerName)
	}
	if vcfg.GetMaxLimitCpu(containerName) != nil && newCPULimit.Cmp(*vcfg.GetMaxLimitCpu(containerName)) == 1 {
		newCPULimit = *vcfg.GetMaxLimitCpu(containerName)
	}
	minDiffCpuLimit := calculator.CalculateResourceValue(container.Resources.Limits[corev1.ResourceCPU], vcfg.GetMinDiffCpuLimitAlgo(containerName), vcfg.GetMinDiffCpuLimitValue(containerName))
	if newCPULimit.Cmp(minDiffCpuLimit) == -1 {
		newCPULimit = cpuLimit
	}
	if vcfg.GetLimitCpuScaleDirection(containerName) == config.ScaleDirectionDown && newCPULimit.Cmp(cpuLimit) == 1 {
		newCPULimit = cpuLimit
	}
	if vcfg.GetLimitCpuScaleDirection(containerName) == config.ScaleDirectionUp && newCPULimit.Cmp(cpuLimit) == -1 {
		newCPULimit = cpuLimit
	}
	if vcfg.GetLimitCPUApplyMode(containerName) == config.ApplyModeEnforce && newCPULimit.String() != cpuLimit.String() {
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

func setContainerMemoryRequest(container *corev1.Container, containerRequestRecommendation *TargetRecommandation, changes []reporting.Change, vcfg *config.VpaWorkloadCfg) []reporting.Change {
	containerName := container.Name
	memoryRequest := *container.Resources.Requests.Memory()
	var newMemoryRequest resource.Quantity
	if vcfg.GetMemoryLimitFromCpuEnabled(containerName) {
		memoryFromCpu := calculator.CalculateCpuToMemory(container.Resources.Requests[corev1.ResourceCPU])
		newMemoryRequest = calculator.CalculateResourceValue(memoryFromCpu, vcfg.GetMemoryRequestFromCpuAlgo(containerName), vcfg.GetMemoryRequestFromCpuValue(containerName))
	} else {
		newMemoryRequest = calculator.CalculateResourceValue(*containerRequestRecommendation.Memory, vcfg.GetIncreaseRequestMemoryAlgo(containerName), vcfg.GetIncreaseRequestMemoryValue(containerName))
	}
	if vcfg.GetMinRequestMemory(containerName) != nil && newMemoryRequest.Cmp(*vcfg.GetMinRequestMemory(containerName)) == -1 {
		newMemoryRequest = *vcfg.GetMinRequestMemory(containerName)
	}
	if vcfg.GetMaxRequestMemory(containerName) != nil && newMemoryRequest.Cmp(*vcfg.GetMaxRequestMemory(containerName)) == 1 {
		newMemoryRequest = *vcfg.GetMaxRequestMemory(containerName)
	}
	minDiffMemoryRequest := calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceMemory], vcfg.GetMinDiffMemoryRequestAlgo(containerName), vcfg.GetMinDiffMemoryRequestValue(containerName))
	if newMemoryRequest.Cmp(minDiffMemoryRequest) == -1 {
		newMemoryRequest = memoryRequest
	}
	if vcfg.GetRequestMemoryScaleDirection(containerName) == config.ScaleDirectionDown && newMemoryRequest.Cmp(memoryRequest) == 1 {
		newMemoryRequest = memoryRequest
	}
	if vcfg.GetRequestMemoryScaleDirection(containerName) == config.ScaleDirectionUp && newMemoryRequest.Cmp(memoryRequest) == -1 {
		newMemoryRequest = memoryRequest
	}
	if vcfg.GetRequestMemoryApplyMode(containerName) == config.ApplyModeEnforce && newMemoryRequest.String() != memoryRequest.String() {
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

func setContainerMemoryLimit(container *corev1.Container, containerRequestRecommendation *TargetRecommandation, containerLimitRecommendation *TargetRecommandation, changes []reporting.Change, vcfg *config.VpaWorkloadCfg) []reporting.Change {
	containerName := container.Name
	memoryLimit := *container.Resources.Limits.Memory()
	var newMemoryLimit resource.Quantity
	if vcfg.GetMemoryLimitFromCpuEnabled(containerName) {
		memoryFromCpu := calculator.CalculateCpuToMemory(container.Resources.Limits[corev1.ResourceCPU])
		newMemoryLimit = calculator.CalculateResourceValue(memoryFromCpu, vcfg.GetMemoryLimitFromCpuAlgo(containerName), vcfg.GetMemoryLimitFromCpuValue(containerName))
	} else {
		if vcfg.GetLimitMemoryApplyTarget(containerName) == config.LimitApplyTargetAuto {
			newMemoryLimit = calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceMemory], vcfg.GetLimitMemoryCalculatorAlgo(containerName), vcfg.GetLimitMemoryCalculatorValue(containerName))
		} else {
			newMemoryLimit = *containerLimitRecommendation.Memory
		}
	}
	if vcfg.GetMinLimitMemory(containerName) != nil && newMemoryLimit.Cmp(*vcfg.GetMinLimitMemory(containerName)) == -1 {
		newMemoryLimit = *vcfg.GetMinLimitMemory(containerName)
	}
	if vcfg.GetMaxLimitMemory(containerName) != nil && newMemoryLimit.Cmp(*vcfg.GetMaxLimitMemory(containerName)) == 1 {
		newMemoryLimit = *vcfg.GetMaxLimitMemory(containerName)
	}
	minDiffMemoryLimit := calculator.CalculateResourceValue(container.Resources.Limits[corev1.ResourceMemory], vcfg.GetMinDiffMemoryLimitAlgo(containerName), vcfg.GetMinDiffMemoryLimitValue(containerName))
	if newMemoryLimit.Cmp(minDiffMemoryLimit) == -1 {
		newMemoryLimit = memoryLimit
	}
	if vcfg.GetLimitMemoryScaleDirection(containerName) == config.ScaleDirectionDown && newMemoryLimit.Cmp(memoryLimit) == 1 {
		newMemoryLimit = memoryLimit
	}
	if vcfg.GetLimitMemoryScaleDirection(containerName) == config.ScaleDirectionUp && newMemoryLimit.Cmp(memoryLimit) == -1 {
		newMemoryLimit = memoryLimit
	}
	if vcfg.GetLimitMemoryApplyMode(containerName) == config.ApplyModeEnforce && newMemoryLimit.String() != memoryLimit.String() {
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
