package controller

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

type VPAConfiguration struct {
	Mode           string            `json:"mode"`
	ResourcePolicy VPAResourcePolicy `json:"resourcePolicy,omitempty"`
}

type VPAResourcePolicy struct {
	ContainerPolicies []ContainerPolicy `json:"containerPolicies,omitempty"`
}

type ContainerPolicy struct {
	ContainerName string            `json:"containerName"`
	MinAllowed    map[string]string `json:"minAllowed,omitempty"`
	MaxAllowed    map[string]string `json:"maxAllowed,omitempty"`
}

type ContainerRecommendation struct {
	ContainerName string
	CPU           string // assumed to be in millicores
	Memory        string // assumed to be in bytes
}

func upsertVPA(dynamicClient dynamic.Interface, opa *OblikPodAutoscaler, mode string) error {

	vpaName := opa.Name
	namespace := opa.Namespace

	// Define the desired state of the VPA object
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "autoscaling.k8s.io/v1",
			"kind":       "VerticalPodAutoscaler",
			"metadata": map[string]interface{}{
				"name":      vpaName,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"targetRef": map[string]interface{}{
					"apiVersion": opa.Spec.TargetRef.APIVersion,
					"kind":       opa.Spec.TargetRef.Kind,
					"name":       opa.Spec.TargetRef.Name,
				},
				"updatePolicy": map[string]interface{}{
					"updateMode": mode,
				},
				"resourcePolicy": opa.Spec.VPA.ResourcePolicy,
			},
		},
	}

	flattenAndClean(obj)

	if err := applyVPA(dynamicClient, opa, obj); err != nil {
		return err
	}

	return nil
}

func applyVPA(dynamicClient dynamic.Interface, opa *OblikPodAutoscaler, obj *unstructured.Unstructured) error {
	data, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal VPA JSON: %s", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "autoscaling.k8s.io",
		Version:  "v1",
		Resource: "verticalpodautoscalers",
	}

	// Send a server-side apply request
	force := true
	_, err = dynamicClient.Resource(gvr).Namespace(opa.Namespace).Patch(context.Background(), opa.Name, types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: FieldManager,
		Force:        &force,
	})
	if err != nil {
		return fmt.Errorf("failed to apply VPA: %s", err)
	}

	fmt.Println("VPA applied successfully.")
	return nil
}

func enableVPA(dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) error {
	return upsertVPA(dynamicClient, opa, "Auto")
}

func disableVPA(dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) error {
	return upsertVPA(dynamicClient, opa, "Off")
}

func convertVPARecommendations(vpa *unstructured.Unstructured) ([]ContainerRecommendation, error) {
	recommendations, found, err := unstructured.NestedSlice(vpa.Object, "status", "recommendation", "containerRecommendations")
	if err != nil {
		return nil, fmt.Errorf("error retrieving container recommendations: %v", err)
	}
	if !found {
		return nil, nil
	}

	var containerRecs []ContainerRecommendation
	for _, item := range recommendations {
		recMap, ok := item.(map[string]interface{})
		if !ok {
			continue // skip items that aren't maps
		}
		containerName, ok := recMap["containerName"].(string)
		if !ok {
			continue // skip if the container name is not a string
		}
		target := recMap["target"].(map[string]interface{})
		cpuQty, cpuOk := target["cpu"].(string)
		memQty, memOk := target["memory"].(string)
		if !cpuOk || !memOk {
			continue // skip if the CPU or memory data is not valid
		}

		containerRecs = append(containerRecs, ContainerRecommendation{
			ContainerName: containerName,
			CPU:           cpuQty,
			Memory:        memQty,
		})
	}

	return containerRecs, nil
}
