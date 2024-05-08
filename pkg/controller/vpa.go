package controller

import (
	"context"
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type VPAConfiguration struct {
	Mode           string            `json:"mode"`
	ResourcePolicy VPAResourcePolicy `json:"resourcePolicy"`
}

type VPAResourcePolicy struct {
	ContainerPolicies []ContainerPolicy `json:"containerPolicies"`
}

type ContainerPolicy struct {
	ContainerName string            `json:"containerName"`
	MinAllowed    map[string]string `json:"minAllowed"`
	MaxAllowed    map[string]string `json:"maxAllowed"`
}

type ContainerRecommendation struct {
	ContainerName string
	CPU           string // assumed to be in millicores
	Memory        string // assumed to be in bytes
}

func upsertVPA(dynamicClient dynamic.Interface, opa *OblikPodAutoscaler, mode string) error {
	vpaGVR := schema.GroupVersionResource{
		Group:    "autoscaling.k8s.io",
		Version:  "v1",
		Resource: "verticalpodautoscalers",
	}

	vpaName := opa.Name
	namespace := opa.Namespace

	// Define the desired state of the VPA object
	vpaObject := &unstructured.Unstructured{
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
			},
		},
	}

	// Check if the VPA already exists
	currentResource, err := dynamicClient.Resource(vpaGVR).Namespace(namespace).Get(context.TODO(), vpaName, metav1.GetOptions{})
	if err != nil {
		// If VPA does not exist, create it
		if errors.IsNotFound(err) {
			_, err := dynamicClient.Resource(vpaGVR).Namespace(namespace).Create(context.TODO(), vpaObject, metav1.CreateOptions{})
			if err != nil {
				log.Printf("Failed to create VPA: %v", err)
				return err
			}
			log.Println("Created VPA successfully")
			return nil
		}
		log.Printf("Failed to get VPA: %v", err)
		return err
	}

	resourceVersion, _, err := unstructured.NestedString(currentResource.Object, "metadata", "resourceVersion")
	if err != nil {
		return fmt.Errorf("failed to get resourceVersion from current resource: %v", err)
	}

	// Set the resource version on the updated resource
	unstructured.SetNestedField(vpaObject.Object, resourceVersion, "metadata", "resourceVersion")

	// If VPA exists, update it
	_, err = dynamicClient.Resource(vpaGVR).Namespace(namespace).Update(context.TODO(), vpaObject, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update VPA: %v", err)
		return err
	}
	log.Println("Updated VPA successfully")
	return nil
}

func enableVPA(dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) error {
	return upsertVPA(dynamicClient, opa, "On")
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
		return nil, fmt.Errorf("no recommendations found in VPA")
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
