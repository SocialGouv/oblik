package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
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
	Kind string `json:"kind"`
	Name string `json:"name"`
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
		case watch.Added:
			pod := event.Object.(*OblikPodAutoscaler)
			handleCreateOrUpdate(clientset, metricsClient, dynamicClient, vpaGVR, pod, true)
		case watch.Modified:
			pod := event.Object.(*OblikPodAutoscaler)
			handleCreateOrUpdate(clientset, metricsClient, dynamicClient, vpaGVR, pod, false)
		case watch.Deleted:
			pod := event.Object.(*OblikPodAutoscaler)
			handleDelete(clientset, dynamicClient, pod)
		}
	}
}

func handleCreateOrUpdate(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, dynamicClient dynamic.Interface, vpaGVR schema.GroupVersionResource, opa *OblikPodAutoscaler, isCreate bool) {
	if shouldEnableVPA(clientset, metricsClient, opa) {
		// Enable VPA
		fmt.Println("Enabling VPA...")
		enableVPA(dynamicClient, vpaGVR, opa, isCreate)
	} else {
		// Enable HPA
		fmt.Println("Enabling HPA...")
		enableHPA(clientset, opa, isCreate)
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

func enableVPA(dynamicClient dynamic.Interface, vpaGVR schema.GroupVersionResource, opa *OblikPodAutoscaler, isCreate bool) {
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

func enableHPA(clientset *kubernetes.Clientset, opa *OblikPodAutoscaler, isCreate bool) {
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
