package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

const MiB = 1024 * 1024

type OblikPodAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OblikPodAutoscalerSpec `json:"spec"`
}

func (o *OblikPodAutoscaler) DeepCopyObject() runtime.Object {
	var copied OblikPodAutoscaler = *o
	return &copied
}

type OblikPodAutoscalerSpec struct {
	CPUCursor    string           `json:"cpuCursor"`
	MemoryCursor string           `json:"memoryCursor"`
	TargetRef    TargetRef        `json:"targetRef"`
	HPA          HPAConfiguration `json:"hpa"`
	VPA          VPAConfiguration `json:"vpa"`
}

type HPAConfiguration struct {
	MinReplicas                    *int32 `json:"minReplicas"`
	MaxReplicas                    int32  `json:"maxReplicas"`
	TargetCPUUtilizationPercentage int32  `json:"targetCPUUtilizationPercentage"`
}

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

type TargetRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Group      string
	Version    string
	Resource    string
}

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building Kubernetes clientset: %s", err)
	}

	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building metrics clientset: %s", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building dynamic client: %s", err)
	}

	oblikPodAutoscalerGVR := schema.GroupVersionResource{
		Group:    "socialgouv.io",
		Version:  "v1",
		Resource: "oblikpodautoscalers",
	}

	watcher, err := dynamicClient.Resource(oblikPodAutoscalerGVR).Namespace("").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to start watcher: %s", err)
	}

	vpaGVR := schema.GroupVersionResource{Group: "autoscaling.k8s.io", Version: "v1", Resource: "verticalpodautoscalers"}

	ch := watcher.ResultChan()

	for event := range ch {
		switch event.Type {
		case watch.Added, watch.Modified, watch.Deleted:
			opa, err := convertToOblikPodAutoscaler(event.Object.(*unstructured.Unstructured))
			if err != nil {
				log.Fatalf("Error converting unstructured to OblikPodAutoscaler: %s", err)
			}
			handleEvent(event.Type, clientset, metricsClient, dynamicClient, vpaGVR, opa)
		}
	}
}

func convertKindToResource(kind string) string {
	// Convert to lowercase
	resource := strings.ToLower(kind)
	// Handle special cases
	switch resource {
	case "endpoints":
		return resource // "Endpoints" is already plural
	default:
		// This simple rule handles most cases
		if strings.HasSuffix(resource, "s") {
			return resource + "es" // For kinds ending in 's' like "Class", making it "classes"
		} else if strings.HasSuffix(resource, "y") {
			return strings.TrimSuffix(resource, "y") + "ies" // For kinds ending in 'y' like "Policy", making it "policies"
		} else {
			return resource + "s"
		}
	}
}

func convertToOblikPodAutoscaler(u *unstructured.Unstructured) (*OblikPodAutoscaler, error) {
	var opa OblikPodAutoscaler
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), &opa)
	if err != nil {
		return nil, err
	}
	group, version := splitAPIVersion(opa.Spec.TargetRef.APIVersion)
	if group == "" && version != "" {
		group = "core"
	}
	resource := convertKindToResource(opa.Spec.TargetRef.Kind)
	opa.Spec.TargetRef.Group = group
	opa.Spec.TargetRef.Version = version
	opa.Spec.TargetRef.Resource = resource
	return &opa, nil
}

func handleEvent(eventType watch.EventType, clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, dynamicClient dynamic.Interface, vpaGVR schema.GroupVersionResource, opa *OblikPodAutoscaler) {
	fmt.Printf("Event Type: %s, OblikPodAutoscaler Name: %s\n", eventType, opa.Name)
	switch eventType {
	case watch.Added:
		handleCreateOrUpdate(clientset, metricsClient, dynamicClient, vpaGVR, opa)
	case watch.Modified:
		handleCreateOrUpdate(clientset, metricsClient, dynamicClient, vpaGVR, opa)
	case watch.Deleted:
		handleDelete(clientset, dynamicClient, opa)
	}
}

func splitAPIVersion(apiVersion string) (string, string) {
	parts := strings.Split(apiVersion, "/")
	if len(parts) == 2 {
		return parts[0], parts[1] // Returns group and version
	}
	// Handle the case for core API group which might be specified as just "v1"
	if len(parts) == 1 {
		return "", parts[0] // Returns empty group and version
	}
	return "", "" // Return empty if the format is not as expected
}

func targetRefExists(dynamicClient dynamic.Interface, targetRef TargetRef, namespace string) bool {
	gvr := schema.GroupVersionResource{Group: targetRef.Group, Version: targetRef.Version, Resource: targetRef.Kind} // Adjust the GroupVersionResource according to your setup
	_, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
	return err == nil
}

func watchForTargetRef(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, dynamicClient dynamic.Interface, vpaGVR schema.GroupVersionResource, opa *OblikPodAutoscaler, targetRef TargetRef) {
	gvr := schema.GroupVersionResource{Group: targetRef.Group, Version: targetRef.Version, Resource: targetRef.Resource} // Adjust the GroupVersionResource according to your setup

	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", targetRef.Name),
	}

	watcher, err := dynamicClient.Resource(gvr).Namespace(opa.Namespace).Watch(context.TODO(), listOptions)
	if err != nil {
		log.Printf("Error on %v: %v", targetRef, err)
		log.Fatalf("Failed to set up watcher for %s: %s", targetRef.Resource, err)
	}

	go func() {
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				log.Printf("targetRef %s appeared. Handling its appearance.", targetRef.Name)
				handleTargetRefAppeared(clientset, metricsClient, dynamicClient, vpaGVR, opa, event.Object.(*unstructured.Unstructured))
			case watch.Deleted:
				log.Printf("targetRef %s was deleted. Handling its deletion.", targetRef.Name)
				handleTargetRefDeleted(clientset, metricsClient, dynamicClient, vpaGVR, opa, event.Object.(*unstructured.Unstructured))
			}
		}
	}()
}

func handleTargetRefAppeared(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, dynamicClient dynamic.Interface, vpaGVR schema.GroupVersionResource, opa *OblikPodAutoscaler, obj *unstructured.Unstructured) {
	// Implement logic to handle the appearance of targetRef
	log.Printf("Handling appearance of new resource: %s", obj.GetName())
	// Reinitialize or adjust OPA settings
	handleTargetRef(clientset, metricsClient, dynamicClient, vpaGVR, opa)
}

func handleTargetRefDeleted(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, dynamicClient dynamic.Interface, vpaGVR schema.GroupVersionResource, opa *OblikPodAutoscaler, obj *unstructured.Unstructured) {
	// Implement logic to handle the deletion of targetRef
	log.Printf("Handling deletion of resource: %s", obj.GetName())
	disableHPA(clientset, opa)
	disableVPA(dynamicClient, opa)
}

func handleTargetRef(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, dynamicClient dynamic.Interface, vpaGVR schema.GroupVersionResource, opa *OblikPodAutoscaler) {
	// Proceed with normal operations like ensuring HPA/VPA is correctly setup
	if shouldEnableVPA(clientset, metricsClient, opa) {
		// Enable VPA
		fmt.Println("Enabling VPA...")
		enableVPA(clientset, dynamicClient, vpaGVR, opa)
	} else {
		// Enable HPA
		fmt.Println("Enabling HPA...")
		enableHPA(clientset, dynamicClient, opa)
	}
}

func upsertVPA(dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) error {
    vpaGVR := schema.GroupVersionResource{
        Group: "autoscaling.k8s.io",
        Version: "v1",
        Resource: "verticalpodautoscalers",
    }

    vpaName := fmt.Sprintf("%s-vpa", opa.Name) // Assume VPA name is derived from OPA name
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
                    "updateMode": "Off",
                },
            },
        },
    }

    // Check if the VPA already exists
    _, err := dynamicClient.Resource(vpaGVR).Namespace(namespace).Get(context.TODO(), vpaName, metav1.GetOptions{})
    if err != nil {
        // If VPA does not exist, create it
        if dynamic.IsNotFound(err) {
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

    // If VPA exists, update it
    _, err = dynamicClient.Resource(vpaGVR).Namespace(namespace).Update(context.TODO(), vpaObject, metav1.UpdateOptions{})
    if err != nil {
        log.Printf("Failed to update VPA: %v", err)
        return err
    }
    log.Println("Updated VPA successfully")
    return nil
}


func handleCreateOrUpdate(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, dynamicClient dynamic.Interface, vpaGVR schema.GroupVersionResource, opa *OblikPodAutoscaler) {
	
	upsertVPA(dynamicClient, opa)

	if !targetRefExists(dynamicClient, opa.Spec.TargetRef, opa.Namespace) {
		log.Printf("targetRef %s/%s does not exist. Setting up watcher.", opa.Spec.TargetRef.Kind, opa.Spec.TargetRef.Name)
		watchForTargetRef(clientset, metricsClient, dynamicClient, vpaGVR, opa, opa.Spec.TargetRef)
	} else {
		log.Printf("targetRef %s/%s exists. Proceeding with normal operations.", opa.Spec.TargetRef.Kind, opa.Spec.TargetRef.Name)
		handleTargetRef(clientset, metricsClient, dynamicClient, vpaGVR, opa)
	}
}

func shouldEnableVPA(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, opa *OblikPodAutoscaler) bool {
	// Fetch current CPU and memory usage for all containers in the pod
	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(opa.Namespace).Get(context.TODO(), opa.Spec.TargetRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error fetching pod metrics: %s", err)
		return false
	}

	var totalCPU int64 = 0
	var totalMemory int64 = 0
	for _, container := range podMetrics.Containers {
		totalCPU += container.Usage.Cpu().MilliValue()  // Total CPU usage in millicores
		totalMemory += container.Usage.Memory().Value() // Total memory usage in bytes
	}

	// Convert cursors from string to int64
	cpuCursor, err := strconv.Atoi(opa.Spec.CPUCursor) // Assuming cpuCursor is percentage of CPU utilization
	if err != nil {
		log.Printf("Invalid CPU cursor value: %s", err)
		return false
	}
	memoryCursor, err := strconv.Atoi(opa.Spec.MemoryCursor) // Assuming memoryCursor is in MiB
	if err != nil {
		log.Printf("Invalid Memory cursor value: %s", err)
		return false
	}

	cpuCursor64 := int64(cpuCursor * 1000)             // Convert to millicores for comparison
	memoryCursor64 := int64(memoryCursor) * int64(MiB) // Convert MiB to bytes

	// Check if current usage is below the defined cursors
	return totalCPU < cpuCursor64 && totalMemory < memoryCursor64
}

func enableVPA(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, vpaGVR schema.GroupVersionResource, opa *OblikPodAutoscaler) {
	// First, disable HPA
	err := disableHPA(clientset, opa)
	if err != nil {
		log.Printf("Error disabling HPA: %s", err)
		return
	}

	// Serialize VPA spec from OPA spec
	vpaSpecData, err := json.Marshal(opa.Spec.VPA)
	if err != nil {
		log.Fatalf("Failed to marshal VPA spec: %s", err)
		return
	}

	// Convert serialized VPA spec to unstructured object
	var unstructuredVPA map[string]interface{}
	err = json.Unmarshal(vpaSpecData, &unstructuredVPA)
	if err != nil {
		log.Fatalf("Failed to unmarshal VPA spec: %s", err)
		return
	}

	// Set necessary fields for the unstructured VPA object
	unstructuredVPA["apiVersion"] = "autoscaling.k8s.io/v1"
	unstructuredVPA["kind"] = "VerticalPodAutoscaler"
	unstructuredVPA["metadata"] = map[string]interface{}{
		"name":      opa.Name,
		"namespace": opa.Namespace,
	}

	var isCreate bool
	// Check if the VPA exists
	_, err = dynamicClient.Resource(vpaGVR).Namespace(opa.Namespace).Get(context.TODO(), opa.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			isCreate = true
		} else {
			log.Printf("Error checking if VPA exists for %s: %v", opa.Name, err)
			return
		}
	}

	// Create or update VPA based on the unstructured object
	vpaObj := &unstructured.Unstructured{Object: unstructuredVPA}
	if isCreate {
		_, err = dynamicClient.Resource(vpaGVR).Namespace(opa.Namespace).Create(context.TODO(), vpaObj, metav1.CreateOptions{})
	} else {
		_, err = dynamicClient.Resource(vpaGVR).Namespace(opa.Namespace).Update(context.TODO(), vpaObj, metav1.UpdateOptions{})
	}
	if err != nil {
		log.Fatalf("Failed to manage VPA: %s", err)
	}
	fmt.Println("VPA managed for", opa.Name)
}

func enableHPA(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) {

	// First, disable VPA
	err := disableVPA(dynamicClient, opa)
	if err != nil {
		log.Printf("Error disabling VPA: %s", err)
		return
	}

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
	_, err = clientset.AutoscalingV1().HorizontalPodAutoscalers(opa.Namespace).Get(context.TODO(), opa.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			isCreate = true
		} else {
			log.Printf("Error checking if HPA exists for %s: %v", opa.Name, err)
			return
		}
	}

	if isCreate {
		_, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(opa.Namespace).Create(context.TODO(), hpa, metav1.CreateOptions{})
		if err != nil {
			log.Fatalf("Failed to create HPA: %s", err)
		}
	} else {
		_, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(opa.Namespace).Update(context.TODO(), hpa, metav1.UpdateOptions{})
		if err != nil {
			log.Fatalf("Failed to update HPA: %s", err)
		}
	}
	fmt.Println("HPA enabled for", opa.Name)
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

func disableVPA(dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) error {
    vpaGVR := schema.GroupVersionResource{
        Group: "autoscaling.k8s.io",
        Version: "v1",
        Resource: "verticalpodautoscalers",
    }

    // Fetch the current VPA object
    vpaObj, err := dynamicClient.Resource(vpaGVR).Namespace(opa.Namespace).Get(context.TODO(), opa.Name, metav1.GetOptions{})
    if err != nil {
        log.Printf("Failed to get VPA for %s: %v", opa.Name, err)
        return err
    }

    // Update the updateMode to "Off"
    if err := updateVPAUpdateMode(vpaObj, "Off"); err != nil {
        log.Printf("Failed to prepare VPA update for %s: %v", opa.Name, err)
        return err
    }

    // Update the VPA resource
    _, err = dynamicClient.Resource(vpaGVR).Namespace(opa.Namespace).Update(context.TODO(), vpaObj, metav1.UpdateOptions{})
    if err != nil {
        log.Printf("Failed to update VPA for %s to Audit mode: %v", opa.Name, err)
        return err
    }

    log.Printf("VPA for %s switched to Audit mode", opa.Name)
    return nil
}

func updateVPAUpdateMode(vpaObj *unstructured.Unstructured, mode string) error {
    unstructured.SetNestedField(vpaObj.Object, mode, "spec", "updatePolicy", "updateMode")
    return nil
}


func calculateCPUUtilizationTarget(cursor string) *int32 {
	utilization := int32(80) // Placeholder for calculation from cursor
	return &utilization
}

func handleDelete(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, opa *OblikPodAutoscaler) {
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
