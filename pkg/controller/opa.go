package controller

import (
	"context"
	"fmt"
	"log"
	"sync"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type OblikPodAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OblikPodAutoscalerSpec `json:"spec"`

	Key string
}

func (o *OblikPodAutoscaler) DeepCopyObject() runtime.Object {
	var copied OblikPodAutoscaler = *o
	return &copied
}

type OblikPodAutoscalerSpec struct {
	TargetRef        TargetRef         `json:"targetRef"`
	HPA              HPAConfiguration  `json:"hpa"`
	VPA              VPAConfiguration  `json:"vpa"`
	CursorMode       string            `json:"cursorMode"`
	PodCursor        ResourceCursor    `json:"podCursors,omitempty"`
	ContainerCursors []ContainerCursor `json:"containerCursors,omitempty"`
	BaseResource     BaseResource      `json:"baseResource,omitempty"`
	MinReplicas      int32             `json:"minReplicas"`
	DefaultLimit     bool              `json:"defaultLimit"`
	EnforceLimit     bool              `json:"enforceLimit"`
	LimitRatio       LimitRatio        `json:"limitRatio"`
}

type BaseResource struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type ResourceCursor struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type ContainerCursor struct {
	ContainerName string `json:"containerName"`
	CPU           string `json:"cpu"`
	Memory        string `json:"memory"`
}

type LimitRatio struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

type Decision struct {
	SwitchToHPA bool
	SwitchToVPA bool
}

// Watching OPA
func watchOpa(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface) {
	oblikPodAutoscalerGVR := schema.GroupVersionResource{
		Group:    "socialgouv.io",
		Version:  "v1",
		Resource: "oblikpodautoscalers",
	}

	watcher, err := dynamicClient.Resource(oblikPodAutoscalerGVR).Namespace("").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to start watcher: %s", err)
	}

	ch := watcher.ResultChan()

	for event := range ch {
		switch event.Type {
		case watch.Added, watch.Modified, watch.Deleted:
			opa, err := convertToOblikPodAutoscaler(event.Object.(*unstructured.Unstructured))
			if err != nil {
				log.Fatalf("Error converting unstructured to OblikPodAutoscaler: %s", err)
			}
			handleOpaEvent(event.Type, clientset, dynamicClient, opa)
		}
	}
}

func handleOpaEvent(eventType watch.EventType, clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) {
	fmt.Printf("Event Type: %s, OblikPodAutoscaler Name: %s\n", eventType, opa.Name)
	switch eventType {
	case watch.Added, watch.Modified:
		handleOpaApply(clientset, dynamicClient, opa)
	case watch.Deleted:
		handleOpaDelete(clientset, dynamicClient, opa)
	}
}

func convertToOblikPodAutoscaler(u *unstructured.Unstructured) (*OblikPodAutoscaler, error) {
	var opa OblikPodAutoscaler
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), &opa)
	if err != nil {
		return nil, err
	}

	opa.Key = fmt.Sprintf("%s/%s", opa.Namespace, opa.Name)

	group, version := splitAPIVersion(opa.Spec.TargetRef.APIVersion)
	if group == "" && version != "" {
		group = "core"
	}
	resource := convertKindToResource(opa.Spec.TargetRef.Kind)
	opa.Spec.TargetRef.Group = group
	opa.Spec.TargetRef.Version = version
	opa.Spec.TargetRef.Resource = resource

	opa.Spec.TargetRef.Namespace = opa.Namespace

	return &opa, nil
}

// Handling OPA
func handleOpaApply(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) {
	upsertVPA(dynamicClient, opa, "Off")
	startWatchingVpa(clientset, dynamicClient, opa)
}

func handleOpaDelete(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) {
	stopWatchingVpa(opa.Key)

	// Delete HPA
	err := clientset.AutoscalingV1().HorizontalPodAutoscalers(opa.Namespace).Delete(context.TODO(), opa.Name, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("Failed to delete HPA: %s", err)
	} else {
		log.Println("HPA successfully deleted:", opa.Name)
	}

	// Delete VPA
	vpaGVR := schema.GroupVersionResource{Group: "autoscaling.k8s.io", Version: "v1", Resource: "verticalpodautoscalers"}
	err = dynamicClient.Resource(vpaGVR).Namespace(opa.Namespace).Delete(context.TODO(), opa.Name, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("Failed to delete VPA: %s", err)
	} else {
		log.Println("VPA successfully deleted:", opa.Name)
	}
}

// Watching VPA
var watchers = map[string]watch.Interface{}
var mu sync.Mutex

func startWatchingVpa(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) error {
	mu.Lock()
	defer mu.Unlock()

	vpaGVR := schema.GroupVersionResource{
		Group:    "autoscaling.k8s.io",
		Version:  "v1",
		Resource: "verticalpodautoscalers",
	}

	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", opa.Name),
	}

	watcher, err := dynamicClient.Resource(vpaGVR).Namespace(opa.Namespace).Watch(context.TODO(), listOptions)
	if err != nil {
		log.Fatalf("Failed to set up watcher for VPA: %s", err)
		return err
	}

	watchers[opa.Key] = watcher

	go func() {
		for event := range watcher.ResultChan() {
			vpa, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				log.Println("Error parsing VPA object")
				continue
			}
			fmt.Printf("Detected %s on VPA: %s\n", event.Type, vpa.GetName())
			switch event.Type {
			case watch.Added, watch.Modified:
				err := handleVPARecommendation(clientset, dynamicClient, opa, vpa)
				if err != nil {
					log.Printf("Failed to handle vpa recommendation for %s: %s", opa.Key, err)
				}
			}
		}
	}()

	return nil
}

func stopWatchingVpa(opaKey string) {
	mu.Lock()
	defer mu.Unlock()

	watcher, exists := watchers[opaKey]
	if exists {
		watcher.Stop()
		delete(watchers, opaKey)
		log.Printf("Stopped watching VPA for OPA with opaKey %s", opaKey)
	} else {
		log.Printf("No active watcher found for VPA with opaKey %s", opaKey)
	}
}

// Handling VPA
func handleVPARecommendation(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler, vpa *unstructured.Unstructured) error {
	containerRecommendations, err := convertVPARecommendations(vpa)
	if err != nil {
		fmt.Printf("Error processing VPA recommendations: %s\n", err)
		return err
	}

	targetRef := opa.Spec.TargetRef
	targetObj, err := getTargetResource(dynamicClient, targetRef)
	if err != nil {
		return err
	}

	if err := enforceMinReplicas(dynamicClient, targetObj, targetRef, opa.Spec.MinReplicas); err != nil {
		return err
	}

	if err := enforceResourceLimits(dynamicClient, targetObj, targetRef, opa.Spec.LimitRatio, opa.Spec.DefaultLimit, opa.Spec.EnforceLimit, containerRecommendations); err != nil {
		return err
	}

	currentReplicas, err := getCurrentReplicas(dynamicClient, targetObj, targetRef)
	if err != nil {
		return err
	}

	decision := determineAction(opa, containerRecommendations, currentReplicas)

	if decision.SwitchToHPA {
		if err := switchToHPA(clientset, dynamicClient, opa); err != nil {
			return err
		}
	} else if decision.SwitchToVPA {
		if err := switchToVPA(clientset, dynamicClient, opa); err != nil {
			return err
		}
	}
	return nil
}

func determineAction(opa *OblikPodAutoscaler, recommendations []ContainerRecommendation, currentReplicas int32) Decision {

	cursors := opa.Spec.ContainerCursors
	cursorMode := opa.Spec.CursorMode
	minReplicas := opa.Spec.MinReplicas
	baseResource := opa.Spec.BaseResource

	// Calculate the replica ratio
	var replicaRatio float64
	if minReplicas > 0 {
		replicaRatio = float64(currentReplicas) / float64(minReplicas)
	} else {
		replicaRatio = 1 // Avoid division by zero
	}

	totalCPU := resource.NewQuantity(0, resource.DecimalSI)
	totalMemory := resource.NewQuantity(0, resource.DecimalSI)

	hpaCount := 0
	for _, cursor := range cursors {
		for _, recommendation := range recommendations {
			if recommendation.ContainerName == cursor.ContainerName {
				// Adjust resources by subtracting base, multiplying by the ratio, and then adding the base back
				cpuAdjusted := adjustResource(recommendation.CPU, baseResource.CPU, replicaRatio)
				memAdjusted := adjustResource(recommendation.Memory, baseResource.Memory, replicaRatio)

				// Check if the adjusted recommendations exceed the cursors
				cpuExceeds := exceedsThreshold(cpuAdjusted, cursor.CPU)
				memExceeds := exceedsThreshold(memAdjusted, cursor.Memory)

				if cpuExceeds || memExceeds {
					hpaCount++
				}

				recCPUQty, _ := resource.ParseQuantity(recommendation.CPU)
				recMemoryQty, _ := resource.ParseQuantity(recommendation.Memory)
				totalCPU.Add(recCPUQty)
				totalMemory.Add(recMemoryQty)
				break
			}
		}
	}

	podCursor := opa.Spec.PodCursor
	if podCursor.CPU != "" || podCursor.Memory != "" {
		podCurCPUQty, _ := resource.ParseQuantity(podCursor.CPU)
		podCurMemoryQty, _ := resource.ParseQuantity(podCursor.Memory)
		if totalCPU.Cmp(podCurCPUQty) > 0 || totalMemory.Cmp(podCurMemoryQty) > 0 {
			return Decision{SwitchToHPA: true}
		}
	}

	// Decide based on cursorMode
	switch cursorMode {
	case "any":
		if hpaCount > 0 {
			return Decision{SwitchToHPA: true}
		}
	case "all":
		if hpaCount == len(cursors) {
			return Decision{SwitchToHPA: true}
		}
	}

	// Default to switching to VPA if no conditions are met for HPA
	return Decision{SwitchToVPA: true}
}

func switchToHPA(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) error {
	log.Printf("Switch VPA to HPA based on VPA recommendations for OPA %s", opa.Key)
	if err := disableVPA(dynamicClient, opa); err != nil {
		log.Printf("Failed to disable VPA: %v", err)
		return err
	}
	if err := enableHPA(clientset, opa); err != nil {
		log.Printf("Failed to enable HPA: %v", err)
		return err
	}
	return nil
}

func switchToVPA(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) error {
	log.Printf("Switch HPA to VPA based on VPA recommendations for OPA %s", opa.Key)
	if err := disableHPA(clientset, opa); err != nil {
		log.Printf("Failed to disable HPA: %v", err)
		return err
	}
	if err := enableVPA(dynamicClient, opa); err != nil {
		log.Printf("Failed to enable VPA: %v", err)
		return err
	}
	return nil
}
