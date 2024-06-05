package controller

import (
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type UpdateType int

const (
	UpdateTypeCpuRequest UpdateType = iota
	UpdateTypeMemoryRequest
	UpdateTypeCpuLimit
	UpdateTypeMemoryLimit
)

type Update struct {
	Old           resource.Quantity
	New           resource.Quantity
	Type          UpdateType
	ContainerName string
}

type TargetRecommandation struct {
	Cpu           *resource.Quantity
	Memory        *resource.Quantity
	ContainerName string
}

func updateDeployment(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) (*[]Update, error) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	deploymentName := targetRef.Name
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error fetching deployment: %s", err.Error())
	}

	updates := updateContainerResources(deployment.Spec.Template.Spec.Containers, vpa, vcfg)

	patchData, err := createPatch(deployment, "apps/v1", "Deployment")
	if err != nil {
		return nil, fmt.Errorf("Error creating patch: %s", err.Error())
	}

	force := true
	_, err = clientset.AppsV1().Deployments(namespace).Patch(context.TODO(), deploymentName, types.ApplyPatchType, patchData, metav1.PatchOptions{
		FieldManager: "oblik-operator",
		Force:        &force, // Force the apply to take ownership of the fields
	})
	if err != nil {
		klog.Errorf("Error applying patch to deployment: %s", err.Error())
	}
	return &updates, nil
}

func updateCronJob(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) (*[]Update, error) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	cronjobName := targetRef.Name

	cronjob, err := clientset.BatchV1().CronJobs(namespace).Get(context.TODO(), cronjobName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error fetching cronjob: %s", err.Error())
	}

	updates := updateContainerResources(cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers, vpa, vcfg)

	patchData, err := createPatch(cronjob, "batch/v1", "CronJob")
	if err != nil {
		return nil, fmt.Errorf("Error creating patch: %s", err.Error())

	}

	force := true
	_, err = clientset.BatchV1().CronJobs(namespace).Patch(context.TODO(), cronjobName, types.ApplyPatchType, patchData, metav1.PatchOptions{
		FieldManager: "oblik-operator",
		Force:        &force, // Force the apply to take ownership of the fields
	})
	if err != nil {
		return nil, fmt.Errorf("Error applying patch to deployment: %s", err.Error())
	}
	return &updates, nil
}

func updateStatefulSet(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) (*[]Update, error) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	statefulSetName := targetRef.Name

	statefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error fetching stateful set: %s", err.Error())
	}

	updates := updateContainerResources(statefulSet.Spec.Template.Spec.Containers, vpa, vcfg)

	patchData, err := createPatch(statefulSet, "apps/v1", "StatefulSet")
	if err != nil {
		return nil, fmt.Errorf("Error creating patch: %s", err.Error())
	}

	_, err = clientset.AppsV1().StatefulSets(namespace).Patch(context.TODO(), statefulSetName, types.ApplyPatchType, patchData, metav1.PatchOptions{
		FieldManager: "oblik-operator",
	})
	if err != nil {
		klog.Errorf("Error applying patch to statefulset: %s", err.Error())
	}
	return &updates, nil
}

func findContainerPolicy(vpaResources *vpa.VerticalPodAutoscaler, containerName string) *vpa.ContainerResourcePolicy {
	for _, containerPolicy := range vpaResources.Spec.ResourcePolicy.ContainerPolicies {
		if containerPolicy.ContainerName == containerName || containerPolicy.ContainerName == "*" {
			return &containerPolicy
		}
	}
	return nil
}

func getTargetRecommandations(vpaResources *vpa.VerticalPodAutoscaler) []TargetRecommandation {
	recommandations := []TargetRecommandation{}
	for _, containerRecommendation := range vpaResources.Status.Recommendation.ContainerRecommendations {
		recommandations = append(recommandations, TargetRecommandation{
			Cpu:           containerRecommendation.Target.Cpu(),
			Memory:        containerRecommendation.Target.Memory(),
			ContainerName: containerRecommendation.ContainerName,
		})
	}
	return recommandations
}

func setUnprovidedDefaultRecommandations(containers []corev1.Container, recommandations []TargetRecommandation, vpaResources *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) []TargetRecommandation {
	for _, container := range containers {
		var found bool
		for _, containerRecommendation := range recommandations {
			if containerRecommendation.ContainerName != container.Name {
				continue
			}
			found = true
			break
		}
		if !found {
			containerRecommandation := TargetRecommandation{
				ContainerName: container.Name,
			}
			switch vcfg.UnprovidedApplyDefaultRequestCPUSource {
			case UnprovidedApplyDefaultModeMaxAllowed:
				containerRecommandation.Cpu = findContainerPolicy(vpaResources, container.Name).MaxAllowed.Cpu()
			case UnprovidedApplyDefaultModeMinAllowed:
				containerRecommandation.Cpu = findContainerPolicy(vpaResources, container.Name).MinAllowed.Cpu()
			case UnprovidedApplyDefaultModeValue:
				cpu, err := resource.ParseQuantity(vcfg.UnprovidedApplyDefaultRequestCPUValue)
				if err != nil {
					klog.Warningf("Set unprovided CPU resources, value parsing error: %s", err.Error())
					break
				}
				containerRecommandation.Cpu = &cpu
			}
			switch vcfg.UnprovidedApplyDefaultRequestMemorySource {
			case UnprovidedApplyDefaultModeMaxAllowed:
				containerRecommandation.Memory = findContainerPolicy(vpaResources, container.Name).MaxAllowed.Memory()
			case UnprovidedApplyDefaultModeMinAllowed:
				containerRecommandation.Memory = findContainerPolicy(vpaResources, container.Name).MinAllowed.Memory()
			case UnprovidedApplyDefaultModeValue:
				memory, err := resource.ParseQuantity(vcfg.UnprovidedApplyDefaultRequestMemoryValue)
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

func applyRecommandationsToContainers(containers []corev1.Container, recommandations []TargetRecommandation, vcfg *VPAOblikConfig) []Update {
	updates := []Update{}
	for index, container := range containers {
		for _, containerRecommendation := range recommandations {
			if containerRecommendation.ContainerName != container.Name {
				continue
			}

			if container.Resources.Requests == nil {
				container.Resources.Requests = corev1.ResourceList{}
			}
			if container.Resources.Limits == nil {
				container.Resources.Limits = corev1.ResourceList{}
			}

			newCPURequests := calculateNewResourceValue(*containerRecommendation.Cpu, vcfg.IncreaseRequestCpuAlgo, vcfg.IncreaseRequestCpuValue)
			cpuRequest := *container.Resources.Requests.Cpu()
			if vcfg.RequestCPUApplyMode == ApplyModeEnforce && newCPURequests.String() != cpuRequest.String() {
				updates = append(updates, Update{
					Old:           cpuRequest,
					New:           newCPURequests,
					Type:          UpdateTypeCpuRequest,
					ContainerName: container.Name,
				})
				container.Resources.Requests[corev1.ResourceCPU] = newCPURequests
			}

			newCPULimit := calculateNewResourceValue(container.Resources.Requests[corev1.ResourceCPU], vcfg.LimitCPUCalculatorAlgo, vcfg.LimitCPUCalculatorValue)
			cpuLimit := *container.Resources.Limits.Cpu()
			if vcfg.LimitCPUApplyMode == ApplyModeEnforce && newCPULimit.String() != cpuLimit.String() {
				updates = append(updates, Update{
					Old:           cpuLimit,
					New:           newCPULimit,
					Type:          UpdateTypeCpuLimit,
					ContainerName: container.Name,
				})
				container.Resources.Limits[corev1.ResourceCPU] = newCPULimit
			}

			newMemoryRequest := calculateNewResourceValue(*containerRecommendation.Memory, vcfg.IncreaseRequestMemoryAlgo, vcfg.IncreaseRequestMemoryValue)
			memoryRequest := *container.Resources.Requests.Memory()
			if vcfg.RequestMemoryApplyMode == ApplyModeEnforce && newMemoryRequest.String() != memoryRequest.String() {
				updates = append(updates, Update{
					Old:           memoryRequest,
					New:           newMemoryRequest,
					Type:          UpdateTypeMemoryRequest,
					ContainerName: container.Name,
				})
				container.Resources.Requests[corev1.ResourceMemory] = newMemoryRequest
			}

			newMemoryLimit := calculateNewResourceValue(container.Resources.Requests[corev1.ResourceMemory], vcfg.LimitMemoryCalculatorAlgo, vcfg.LimitMemoryCalculatorValue)
			memoryLimit := *container.Resources.Limits.Memory()
			if vcfg.LimitMemoryApplyMode == ApplyModeEnforce && newMemoryLimit.String() != memoryLimit.String() {
				updates = append(updates, Update{
					Old:           memoryLimit,
					New:           newMemoryLimit,
					Type:          UpdateTypeMemoryLimit,
					ContainerName: container.Name,
				})
				container.Resources.Limits[corev1.ResourceMemory] = newMemoryLimit
			}
			containers[index] = container
			break
		}
	}
	return updates
}

func getUpdateTypeLabel(updateType UpdateType) string {
	switch updateType {
	case UpdateTypeCpuRequest:
		return "CPU request"
	case UpdateTypeMemoryRequest:
		return "Memory request"
	case UpdateTypeCpuLimit:
		return "CPU limit"
	case UpdateTypeMemoryLimit:
		return "Memory limit"
	}
	return ""
}

func updateContainerResources(containers []corev1.Container, vpaResources *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) []Update {
	recommandations := getTargetRecommandations(vpaResources)
	recommandations = setUnprovidedDefaultRecommandations(containers, recommandations, vpaResources, vcfg)
	updates := applyRecommandationsToContainers(containers, recommandations, vcfg)
	return updates
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
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	jsonData, err := json.Marshal(patchedObj)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}
