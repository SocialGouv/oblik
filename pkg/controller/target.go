package controller

import (
	"context"
	"encoding/json"
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	jsonpatch "github.com/evanphx/json-patch/v5"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/dynamic"
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

var FieldManager = "oblik-operator"

func updateDeployment(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VpaWorkloadCfg) (*[]Update, error) {
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
		FieldManager: FieldManager,
		Force:        &force, // Force the apply to take ownership of the fields
	})
	if err != nil {
		return nil, fmt.Errorf("Error applying patch to deployment: %s", err.Error())
	}
	return &updates, nil
}

func updateCronJob(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VpaWorkloadCfg) (*[]Update, error) {
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
		FieldManager: FieldManager,
		Force:        &force, // Force the apply to take ownership of the fields
	})
	if err != nil {
		return nil, fmt.Errorf("Error applying patch to deployment: %s", err.Error())
	}
	return &updates, nil
}

func updateStatefulSet(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VpaWorkloadCfg) (*[]Update, error) {
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

	force := true
	_, err = clientset.AppsV1().StatefulSets(namespace).Patch(context.TODO(), statefulSetName, types.ApplyPatchType, patchData, metav1.PatchOptions{
		FieldManager: FieldManager,
		Force:        &force, // Force the apply to take ownership of the fields
	})
	if err != nil {
		return nil, fmt.Errorf("Error applying patch to statefulset: %s", err.Error())
	}
	return &updates, nil
}

func updateCluster(dynamicClient *dynamic.DynamicClient, vpa *vpa.VerticalPodAutoscaler, vcfg *VpaWorkloadCfg) (*[]Update, error) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	clusterName := targetRef.Name

	gvr := schema.GroupVersionResource{
		Group:    "postgresql.cnpg.io",
		Version:  "v1",
		Resource: "clusters",
	}

	clusterResource, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error fetching cluster: %s", err.Error())
	}

	originalClusterJSON, err := json.Marshal(clusterResource)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling original cluster: %s", err.Error())
	}

	var cluster cnpgv1.Cluster
	err = json.Unmarshal(originalClusterJSON, &cluster)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling cluster: %s", err.Error())
	}

	containers := []corev1.Container{
		corev1.Container{
			Name:      "postgres",
			Resources: cluster.Spec.Resources,
		},
	}
	klog.Infof("debug cnpg cluster.Spec.Resources: %v", cluster.Spec.Resources)
	updates := updateContainerResources(containers, vpa, vcfg)
	cluster.Spec.Resources = containers[0].Resources

	updatedClusterJSON, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling updated cluster: %s", err.Error())
	}

	patchBytes, err := jsonpatch.CreateMergePatch(originalClusterJSON, updatedClusterJSON)
	if err != nil {
		return nil, fmt.Errorf("Error creating patch: %s", err.Error())
	}

	klog.Infof("debug cnpg originalClusterJSON: %s", originalClusterJSON)
	klog.Infof("debug cnpg updatedClusterJSON: %s", updatedClusterJSON)
	klog.Infof("debug cnpg patch: %s", string(patchBytes))

	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Patch(context.TODO(), clusterName, types.MergePatchType, patchBytes, metav1.PatchOptions{
		FieldManager: FieldManager,
	})
	if err != nil {
		return nil, fmt.Errorf("Error applying patch to cluster: %s", err.Error())
	}
	return &updates, nil
}

func findContainerPolicy(vpaResource *vpa.VerticalPodAutoscaler, containerName string) *vpa.ContainerResourcePolicy {
	for _, containerPolicy := range vpaResource.Spec.ResourcePolicy.ContainerPolicies {
		if containerPolicy.ContainerName == containerName || containerPolicy.ContainerName == "*" {
			return &containerPolicy
		}
	}
	return nil
}

func getTargetRecommandations(vpaResource *vpa.VerticalPodAutoscaler) []TargetRecommandation {
	recommandations := []TargetRecommandation{}
	if vpaResource.Status.Recommendation != nil {
		for _, containerRecommendation := range vpaResource.Status.Recommendation.ContainerRecommendations {
			recommandations = append(recommandations, TargetRecommandation{
				Cpu:           containerRecommendation.Target.Cpu(),
				Memory:        containerRecommendation.Target.Memory(),
				ContainerName: containerRecommendation.ContainerName,
			})
		}
	}
	return recommandations
}

func setUnprovidedDefaultRecommandations(containers []corev1.Container, recommandations []TargetRecommandation, vpaResource *vpa.VerticalPodAutoscaler, vcfg *VpaWorkloadCfg) []TargetRecommandation {
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
			case UnprovidedApplyDefaultModeMinAllowed:
				minCpu := findContainerPolicy(vpaResource, containerName).MinAllowed.Cpu()
				if vcfg.GetMinRequestCpu(containerName) != nil && (minCpu == nil || minCpu.Cmp(*vcfg.GetMinRequestCpu(containerName)) == -1) {
					minCpu = vcfg.GetMinRequestCpu(containerName)
				}
				containerRecommandation.Cpu = minCpu
			case UnprovidedApplyDefaultModeMaxAllowed:
				maxCpu := findContainerPolicy(vpaResource, containerName).MaxAllowed.Cpu()
				if vcfg.GetMaxRequestCpu(containerName) != nil && (maxCpu == nil || maxCpu.Cmp(*vcfg.GetMaxRequestCpu(containerName)) == 1) {
					maxCpu = vcfg.GetMaxRequestCpu(containerName)
				}
				containerRecommandation.Cpu = maxCpu
			case UnprovidedApplyDefaultModeValue:
				cpu, err := resource.ParseQuantity(vcfg.GetUnprovidedApplyDefaultRequestCPUValue(containerName))
				if err != nil {
					klog.Warningf("Set unprovided CPU resources, value parsing error: %s", err.Error())
					break
				}
				containerRecommandation.Cpu = &cpu
			}
			switch vcfg.GetUnprovidedApplyDefaultRequestMemorySource(containerName) {
			case UnprovidedApplyDefaultModeMinAllowed:
				minMemory := findContainerPolicy(vpaResource, containerName).MinAllowed.Memory()
				if vcfg.GetMinRequestMemory(containerName) != nil && (minMemory == nil || minMemory.Cmp(*vcfg.GetMinRequestMemory(containerName)) == -1) {
					minMemory = vcfg.GetMinRequestMemory(containerName)
				}
				containerRecommandation.Memory = minMemory
			case UnprovidedApplyDefaultModeMaxAllowed:
				maxMemory := findContainerPolicy(vpaResource, containerName).MaxAllowed.Memory()
				if vcfg.GetMaxRequestMemory(containerName) != nil && (maxMemory == nil || maxMemory.Cmp(*vcfg.GetMaxRequestMemory(containerName)) == 1) {
					maxMemory = vcfg.GetMaxRequestMemory(containerName)
				}
				containerRecommandation.Memory = maxMemory
			case UnprovidedApplyDefaultModeValue:
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

func applyRecommandationsToContainers(containers []corev1.Container, recommandations []TargetRecommandation, vcfg *VpaWorkloadCfg) []Update {
	updates := []Update{}

	for index, container := range containers {
		containerName := container.Name
		for _, containerRecommendation := range recommandations {
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
				newCPURequest := calculateNewResourceValue(*containerRecommendation.Cpu, vcfg.GetIncreaseRequestCpuAlgo(containerName), vcfg.GetIncreaseRequestCpuValue(containerName))
				if vcfg.GetMinRequestCpu(containerName) != nil && newCPURequest.Cmp(*vcfg.GetMinRequestCpu(containerName)) == -1 {
					newCPURequest = *vcfg.GetMinRequestCpu(containerName)
				}
				if vcfg.GetMaxRequestCpu(containerName) != nil && newCPURequest.Cmp(*vcfg.GetMaxRequestCpu(containerName)) == 1 {
					newCPURequest = *vcfg.GetMaxRequestCpu(containerName)
				}
				cpuRequest := *container.Resources.Requests.Cpu()
				if vcfg.GetRequestCPUApplyMode(containerName) == ApplyModeEnforce && newCPURequest.String() != cpuRequest.String() {
					updates = append(updates, Update{
						Old:           cpuRequest,
						New:           newCPURequest,
						Type:          UpdateTypeCpuRequest,
						ContainerName: containerName,
					})
					container.Resources.Requests[corev1.ResourceCPU] = newCPURequest
				}

				newCPULimit := calculateNewResourceValue(newCPURequest, vcfg.GetLimitCPUCalculatorAlgo(containerName), vcfg.GetLimitCPUCalculatorValue(containerName))
				cpuLimit := *container.Resources.Limits.Cpu()
				if vcfg.GetMinLimitCpu(containerName) != nil && newCPULimit.Cmp(*vcfg.GetMinLimitCpu(containerName)) == -1 {
					newCPULimit = *vcfg.GetMinLimitCpu(containerName)
				}
				if vcfg.GetMaxLimitCpu(containerName) != nil && newCPULimit.Cmp(*vcfg.GetMaxLimitCpu(containerName)) == 1 {
					newCPULimit = *vcfg.GetMaxLimitCpu(containerName)
				}
				if vcfg.GetLimitCPUApplyMode(containerName) == ApplyModeEnforce && newCPULimit.String() != cpuLimit.String() {
					updates = append(updates, Update{
						Old:           cpuLimit,
						New:           newCPULimit,
						Type:          UpdateTypeCpuLimit,
						ContainerName: containerName,
					})
					container.Resources.Limits[corev1.ResourceCPU] = newCPULimit
				}
			}

			if containerRecommendation.Memory != nil {
				newMemoryRequest := calculateNewResourceValue(*containerRecommendation.Memory, vcfg.GetIncreaseRequestMemoryAlgo(containerName), vcfg.GetIncreaseRequestMemoryValue(containerName))
				if vcfg.GetMinRequestMemory(containerName) != nil && newMemoryRequest.Cmp(*vcfg.GetMinRequestMemory(containerName)) == -1 {
					newMemoryRequest = *vcfg.GetMinRequestMemory(containerName)
				}
				if vcfg.GetMaxRequestMemory(containerName) != nil && newMemoryRequest.Cmp(*vcfg.GetMaxRequestMemory(containerName)) == 1 {
					newMemoryRequest = *vcfg.GetMaxRequestMemory(containerName)
				}
				memoryRequest := *container.Resources.Requests.Memory()
				if vcfg.GetRequestMemoryApplyMode(containerName) == ApplyModeEnforce && newMemoryRequest.String() != memoryRequest.String() {
					updates = append(updates, Update{
						Old:           memoryRequest,
						New:           newMemoryRequest,
						Type:          UpdateTypeMemoryRequest,
						ContainerName: containerName,
					})
					container.Resources.Requests[corev1.ResourceMemory] = newMemoryRequest
				}

				newMemoryLimit := calculateNewResourceValue(newMemoryRequest, vcfg.GetLimitMemoryCalculatorAlgo(containerName), vcfg.GetLimitMemoryCalculatorValue(containerName))
				memoryLimit := *container.Resources.Limits.Memory()
				if vcfg.GetMinLimitMemory(containerName) != nil && newMemoryLimit.Cmp(*vcfg.GetMinLimitMemory(containerName)) == -1 {
					newMemoryLimit = *vcfg.GetMinLimitMemory(containerName)
				}
				if vcfg.GetMaxLimitMemory(containerName) != nil && newMemoryLimit.Cmp(*vcfg.GetMaxLimitMemory(containerName)) == 1 {
					newMemoryLimit = *vcfg.GetMaxLimitMemory(containerName)
				}
				if vcfg.GetLimitMemoryApplyMode(containerName) == ApplyModeEnforce && newMemoryLimit.String() != memoryLimit.String() {
					updates = append(updates, Update{
						Old:           memoryLimit,
						New:           newMemoryLimit,
						Type:          UpdateTypeMemoryLimit,
						ContainerName: containerName,
					})
					container.Resources.Limits[corev1.ResourceMemory] = newMemoryLimit
				}
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

func updateContainerResources(containers []corev1.Container, vpaResource *vpa.VerticalPodAutoscaler, vcfg *VpaWorkloadCfg) []Update {
	recommandations := getTargetRecommandations(vpaResource)
	recommandations = setUnprovidedDefaultRecommandations(containers, recommandations, vpaResource, vcfg)
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
