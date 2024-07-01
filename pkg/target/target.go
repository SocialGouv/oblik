package target

import (
	"encoding/json"
	"fmt"

	"github.com/SocialGouv/oblik/pkg/calculator"
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

var FieldManager = "oblik-operator"

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

func applyRecommandationsToContainers(containers []corev1.Container, requestRecommandations []TargetRecommandation, limitRecommandations []TargetRecommandation, vcfg *config.VpaWorkloadCfg) *reporting.UpdateResult {
	changes := []reporting.Change{}
	update := reporting.UpdateResult{
		Key: vcfg.Key,
	}

	for index, container := range containers {
		containerName := container.Name
		for _, containerRecommendation := range requestRecommandations {
			if containerRecommendation.ContainerName != containerName {
				continue
			}

			if container.Resources.Requests == nil {
				container.Resources.Requests = corev1.ResourceList{}
			}
			if container.Resources.Limits == nil {
				container.Resources.Limits = corev1.ResourceList{}
			}

			if containerRecommendation.Cpu != nil {
				cpuRequest := *container.Resources.Requests.Cpu()
				newCPURequest := calculator.CalculateResourceValue(*containerRecommendation.Cpu, vcfg.GetIncreaseRequestCpuAlgo(containerName), vcfg.GetIncreaseRequestCpuValue(containerName))
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

				cpuLimit := *container.Resources.Limits.Cpu()

				var newCPULimit resource.Quantity
				if vcfg.GetLimitCpuApplyTarget(containerName) == config.LimitApplyTargetAuto {
					newCPULimit = calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceCPU], vcfg.GetLimitCPUCalculatorAlgo(containerName), vcfg.GetLimitCPUCalculatorValue(containerName))
				} else {
					for _, limiContainerRecommendation := range limitRecommandations {
						if limiContainerRecommendation.ContainerName != containerName {
							continue
						}
						newCPULimit = *limiContainerRecommendation.Cpu
					}
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
			}

			if containerRecommendation.Memory != nil {
				memoryRequest := *container.Resources.Requests.Memory()
				var newMemoryRequest resource.Quantity
				if vcfg.GetMemoryLimitFromCpuEnabled(containerName) {
					memoryFromCpu := calculator.CalculateCpuToMemory(container.Resources.Requests[corev1.ResourceCPU])
					newMemoryRequest = calculator.CalculateResourceValue(memoryFromCpu, vcfg.GetMemoryRequestFromCpuAlgo(containerName), vcfg.GetMemoryRequestFromCpuValue(containerName))
				} else {
					newMemoryRequest = calculator.CalculateResourceValue(*containerRecommendation.Memory, vcfg.GetIncreaseRequestMemoryAlgo(containerName), vcfg.GetIncreaseRequestMemoryValue(containerName))
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

				memoryLimit := *container.Resources.Limits.Memory()
				var newMemoryLimit resource.Quantity
				if vcfg.GetMemoryLimitFromCpuEnabled(containerName) {
					memoryFromCpu := calculator.CalculateCpuToMemory(container.Resources.Limits[corev1.ResourceCPU])
					newMemoryLimit = calculator.CalculateResourceValue(memoryFromCpu, vcfg.GetMemoryLimitFromCpuAlgo(containerName), vcfg.GetMemoryLimitFromCpuValue(containerName))
				} else {
					if vcfg.GetLimitMemoryApplyTarget(containerName) == config.LimitApplyTargetAuto {
						newMemoryLimit = calculator.CalculateResourceValue(container.Resources.Requests[corev1.ResourceMemory], vcfg.GetLimitMemoryCalculatorAlgo(containerName), vcfg.GetLimitMemoryCalculatorValue(containerName))
					} else {
						for _, limiContainerRecommendation := range limitRecommandations {
							if limiContainerRecommendation.ContainerName != containerName {
								continue
							}
							newMemoryLimit = *limiContainerRecommendation.Memory
						}
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
			}

			containers[index] = container
			break
		}
	}
	update.Changes = changes
	return &update
}

func updateContainerResources(containers []corev1.Container, vpaResource *vpa.VerticalPodAutoscaler, vcfg *config.VpaWorkloadCfg) *reporting.UpdateResult {
	requestRecommandations := getRequestTargetRecommandations(vpaResource, vcfg)
	requestRecommandations = setUnprovidedDefaultRecommandations(containers, requestRecommandations, vpaResource, vcfg)

	limitRecommandations := getLimitTargetRecommandations(vpaResource, vcfg)
	limitRecommandations = setUnprovidedDefaultRecommandations(containers, limitRecommandations, vpaResource, vcfg)

	update := applyRecommandationsToContainers(containers, requestRecommandations, limitRecommandations, vcfg)
	return update
}

func createPatch(obj interface{}, apiVersion, kind string) ([]byte, error) {
	var patchedObj interface{}
	switch t := obj.(type) {
	case *appsv1.Deployment:
		patchedObj = t.DeepCopy()
		patchedObj.(*appsv1.Deployment).APIVersion = apiVersion
		patchedObj.(*appsv1.Deployment).Kind = kind
		patchedObj.(*appsv1.Deployment).ObjectMeta.ManagedFields = nil
	case *appsv1.StatefulSet:
		patchedObj = t.DeepCopy()
		patchedObj.(*appsv1.StatefulSet).APIVersion = apiVersion
		patchedObj.(*appsv1.StatefulSet).Kind = kind
		patchedObj.(*appsv1.StatefulSet).ObjectMeta.ManagedFields = nil
	case *appsv1.DaemonSet:
		patchedObj = t.DeepCopy()
		patchedObj.(*appsv1.DaemonSet).APIVersion = apiVersion
		patchedObj.(*appsv1.DaemonSet).Kind = kind
		patchedObj.(*appsv1.DaemonSet).ObjectMeta.ManagedFields = nil
	case *batchv1.CronJob:
		patchedObj = t.DeepCopy()
		patchedObj.(*batchv1.CronJob).APIVersion = apiVersion
		patchedObj.(*batchv1.CronJob).Kind = kind
		patchedObj.(*batchv1.CronJob).ObjectMeta.ManagedFields = nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	jsonData, err := json.Marshal(patchedObj)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}

func findContainerPolicy(vpaResource *vpa.VerticalPodAutoscaler, containerName string) *vpa.ContainerResourcePolicy {
	for _, containerPolicy := range vpaResource.Spec.ResourcePolicy.ContainerPolicies {
		if containerPolicy.ContainerName == containerName || containerPolicy.ContainerName == "*" {
			return &containerPolicy
		}
	}
	return nil
}
