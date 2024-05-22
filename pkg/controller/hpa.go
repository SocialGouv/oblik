package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type HPAConfiguration struct {
	MinReplicas                    int32 `json:"minReplicas"`
	MaxReplicas                    int32 `json:"maxReplicas"`
	TargetCPUUtilizationPercentage int32 `json:"targetCPUUtilizationPercentage"`
}

func enableHPA(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) error {
	// Deserialize HPA spec from OPA spec
	hpaSpec := &autoscalingv1.HorizontalPodAutoscalerSpec{}
	hpaSpecData, _ := json.Marshal(opa.Spec.HPA) // Assume opa.Spec.HPA holds the full HPA spec as raw JSON
	json.Unmarshal(hpaSpecData, &hpaSpec)

	hpaSpec.ScaleTargetRef = autoscalingv1.CrossVersionObjectReference{
		APIVersion: opa.Spec.TargetRef.APIVersion,
		Kind:       opa.Spec.TargetRef.Kind,
		Name:       opa.Spec.TargetRef.Name,
	}

	// Define the desired state of the HPA object
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "autoscaling/v2",
			"kind":       "HorizontalPodAutoscaler",
			"metadata": map[string]interface{}{
				"name":      opa.Name,
				"namespace": opa.Namespace,
			},
			"spec": *hpaSpec,
		},
	}

	flattenAndClean(obj)

	applyHPA(dynamicClient, opa, obj)
	fmt.Println("HPA enabled for", opa.Name)
	return nil
}

func disableHPA(clientset *kubernetes.Clientset, opa *OblikPodAutoscaler) error {
	// Check if the HPA exists
	_, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(opa.Namespace).Get(context.TODO(), opa.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("HPA %s not found, nothing to disable", opa.Name)
			return nil
		}
		log.Printf("Error checking if HPA exists for %s: %v", opa.Name, err)
		return err
	}

	// Delete HPA if it exists
	deletePolicy := metav1.DeletePropagationForeground
	err = clientset.AutoscalingV1().HorizontalPodAutoscalers(opa.Namespace).Delete(context.TODO(), opa.Name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
		log.Printf("Failed to delete HPA for %s: %v", opa.Name, err)
		return err
	}

	log.Printf("Successfully disabled HPA for %s", opa.Name)
	return nil
}

func applyHPA(dynamicClient dynamic.Interface, opa *OblikPodAutoscaler, obj *unstructured.Unstructured) error {
	gvr := schema.GroupVersionResource{
		Group:    "autoscaling",
		Version:  "v1",
		Resource: "horizontalpodautoscalers",
	}

	// Convert the HPA specification to JSON for the patch
	data, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal HPA JSON: %s", err)
	}

	// Send a server-side apply request
	force := true
	_, err = dynamicClient.Resource(gvr).Namespace(opa.Namespace).Patch(context.Background(), opa.Name, types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: FieldManager,
		Force:        &force,
	})
	if err != nil {
		return fmt.Errorf("failed to apply HPA: %s", err)
	}
	return nil
}
