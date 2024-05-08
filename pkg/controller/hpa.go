package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type HPAConfiguration struct {
	MinReplicas                    int32 `json:"minReplicas"`
	MaxReplicas                    int32 `json:"maxReplicas"`
	TargetCPUUtilizationPercentage int32 `json:"targetCPUUtilizationPercentage"`
}

func enableHPA(clientset *kubernetes.Clientset, opa *OblikPodAutoscaler) error {
	// Deserialize HPA spec from OPA spec
	hpaSpec := &autoscalingv1.HorizontalPodAutoscalerSpec{}
	hpaSpecData, _ := json.Marshal(opa.Spec.HPA) // Assume opa.Spec.HPA holds the full HPA spec as raw JSON
	json.Unmarshal(hpaSpecData, &hpaSpec)

	hpa := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opa.Name,
			Namespace: opa.Namespace,
		},
		Spec: *hpaSpec,
	}

	var isCreate bool
	_, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(opa.Namespace).Get(context.TODO(), opa.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			isCreate = true
		} else {
			log.Printf("Error checking if HPA exists for %s: %v", opa.Name, err)
			return err
		}
	}

	if isCreate {
		_, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(opa.Namespace).Create(context.TODO(), hpa, metav1.CreateOptions{})
		if err != nil {
			log.Printf("Failed to create HPA: %s", err)
			return err
		}
	} else {
		_, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(opa.Namespace).Update(context.TODO(), hpa, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to update HPA: %s", err)
			return err
		}
	}
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
