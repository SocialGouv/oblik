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

func updateDeployment(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	deploymentName := targetRef.Name

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error fetching deployment: %s", err.Error())
		return
	}

	updateContainerResources(deployment.Spec.Template.Spec.Containers, vpa, vcfg)

	patchData, err := createPatch(deployment, "apps/v1", "Deployment")
	if err != nil {
		klog.Errorf("Error creating patch: %s", err.Error())
		return
	}

	force := true
	_, err = clientset.AppsV1().Deployments(namespace).Patch(context.TODO(), deploymentName, types.ApplyPatchType, patchData, metav1.PatchOptions{
		FieldManager: "vpa-operator",
		Force:        &force, // Force the apply to take ownership of the fields
	})
	if err != nil {
		klog.Errorf("Error applying patch to deployment: %s", err.Error())
	}
}

func updateStatefulSet(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	statefulSetName := targetRef.Name

	statefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error fetching stateful set: %s", err.Error())
		return
	}

	updateContainerResources(statefulSet.Spec.Template.Spec.Containers, vpa, vcfg)

	patchData, err := createPatch(statefulSet, "apps/v1", "StatefulSet")
	if err != nil {
		klog.Errorf("Error creating patch: %s", err.Error())
		return
	}

	_, err = clientset.AppsV1().StatefulSets(namespace).Patch(context.TODO(), statefulSetName, types.ApplyPatchType, patchData, metav1.PatchOptions{
		FieldManager: "oblik-operator",
	})
	if err != nil {
		klog.Errorf("Error applying patch to stateful set: %s", err.Error())
	}
}

func updateContainerResources(containers []corev1.Container, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) {
	for _, container := range containers {

		if vcfg.RequestCPUApplyMode == ApplyModeEnforce {
			var newCPURequests resource.Quantity
			for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
				if containerRecommendation.ContainerName == container.Name {
					newCPURequests = *containerRecommendation.Target.Cpu()
					break
				}
			}
			klog.Infof("Setting CPU requests to %s for %s container: %s", newCPURequests.String(), vcfg.Key, container.Name)
			container.Resources.Requests[corev1.ResourceCPU] = newCPURequests
		}
		if vcfg.LimitCPUApplyMode == ApplyModeEnforce {
			newCPULimits := calculateNewLimitValue(container.Resources.Requests[corev1.ResourceCPU], vcfg.LimitCPUCalculatorAlgo, vcfg.LimitCPUCalculatorValue)
			klog.Infof("Setting CPU limits to %s for %s container: %s", newCPULimits.String(), vcfg.Key, container.Name)
			container.Resources.Limits[corev1.ResourceCPU] = newCPULimits
		}

		if vcfg.RequestMemoryApplyMode == ApplyModeEnforce {
			var newMemoryRequests resource.Quantity
			for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
				if containerRecommendation.ContainerName == container.Name {
					newMemoryRequests = *containerRecommendation.Target.Memory()
					break
				}
			}
			klog.Infof("Setting Memory requests to %s for %s container: %s", newMemoryRequests.String(), vcfg.Key, container.Name)
			container.Resources.Requests[corev1.ResourceMemory] = newMemoryRequests
		}
		if vcfg.LimitMemoryApplyMode == ApplyModeEnforce {
			newMemoryLimits := calculateNewLimitValue(container.Resources.Requests[corev1.ResourceMemory], vcfg.LimitMemoryCalculatorAlgo, vcfg.LimitMemoryCalculatorValue)
			klog.Infof("Setting Memory limits to %s for %s container: %s", newMemoryLimits.String(), vcfg.Key, container.Name)
			container.Resources.Limits[corev1.ResourceMemory] = newMemoryLimits
		}

	}
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
