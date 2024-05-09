package controller

import (
	"context"
	"fmt"
	"log"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

type TargetRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Group      string
	Version    string
	Resource   string
	Namespace  string
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

func getTargetResource(dynamicClient dynamic.Interface, targetRef TargetRef) (*unstructured.Unstructured, error) {
	gvr, err := getGVR(targetRef)
	if err != nil {
		return nil, err
	}

	// Get the current resource object
	targetObj, err := dynamicClient.Resource(gvr).Namespace(targetRef.Namespace).Get(context.TODO(), targetRef.Name, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get the target resource %s/%s: %v", targetRef.Namespace, targetRef.Name, err)
	}
	return targetObj, nil
}

func getCurrentReplicas(dynamicClient dynamic.Interface, obj *unstructured.Unstructured, targetRef TargetRef) (int32, error) {
	// Extract the 'spec.replicas' field which is common for Deployment and StatefulSet
	replicas, found, err := unstructured.NestedInt64(obj.UnstructuredContent(), "spec", "replicas")
	if err != nil || !found {
		return 0, fmt.Errorf("error reading replicas for %s %s: %v", targetRef.Kind, targetRef.Name, err)
	}

	return int32(replicas), nil
}

func enforceMinReplicas(dynamicClient dynamic.Interface, obj *unstructured.Unstructured, targetRef TargetRef, minReplicas int32) error {

	// Get current replicas from the resource
	currentReplicas, found, err := unstructured.NestedInt64(obj.UnstructuredContent(), "spec", "replicas")
	if err != nil || !found {
		return fmt.Errorf("failed to get current replicas for %s: %v", targetRef.Name, err)
	}

	if currentReplicas < int64(minReplicas) {
		// Set the replicas to minReplicas
		err := unstructured.SetNestedField(obj.Object, int64(minReplicas), "spec", "replicas")
		if err != nil {
			return fmt.Errorf("failed to set replicas in the resource spec: %v", err)
		}

		if err := applyTarget(dynamicClient, targetRef, obj); err != nil {
			return err
		}

		fmt.Printf("Updated %s %s to have minReplicas = %d\n", targetRef.Kind, targetRef.Name, minReplicas)
	} else {
		fmt.Printf("%s %s already has %d or more replicas\n", targetRef.Kind, targetRef.Name, minReplicas)
	}

	return nil
}

func enforceResourceLimits(dynamicClient dynamic.Interface, obj *unstructured.Unstructured, targetRef TargetRef, limitRatio LimitRatio, enableDefault bool, enableEnforce bool, containerRecommendations []ContainerRecommendation) error {
	if !enableDefault && !enableEnforce {
		return nil
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil || !found {
		return fmt.Errorf("error finding containers in the resource: %v", err)
	}

	modified := false
	for i, container := range containers {
		containerMap := container.(map[string]interface{})

		// Get current requests and limits
		requests, _, _ := unstructured.NestedStringMap(containerMap, "resources", "requests")
		limits, _, _ := unstructured.NestedStringMap(containerMap, "resources", "limits")

		newLimits := map[string]string{}

		var containerRecommendation ContainerRecommendation
		for _, recommendation := range containerRecommendations {
			containerName, _, _ := unstructured.NestedString(containerMap, "name")
			if recommendation.ContainerName == containerName {
				containerRecommendation = recommendation
				break
			}
		}

		if limits["cpu"] == "" || limits["cpu"] == "0" || enableEnforce {
			var reqCPUQty resource.Quantity
			if requests["cpu"] != "" && requests["cpu"] != "0" {
				reqCPUQty = resource.MustParse(requests["cpu"])
			} else {
				reqCPUQty = resource.MustParse(containerRecommendation.CPU)
			}
			newLimits["cpu"] = fmt.Sprintf("%.3f", reqCPUQty.AsApproximateFloat64()*limitRatio.CPU)
			modified = true
		} else {
			newLimits["cpu"] = limits["cpu"]
		}

		if limits["memory"] == "" || limits["memory"] == "0" || enableEnforce {
			var reqMemoryQty resource.Quantity
			if requests["memory"] != "" && requests["memory"] != "0" {
				reqMemoryQty = resource.MustParse(requests["memory"])
			} else {
				reqMemoryQty = resource.MustParse(containerRecommendation.Memory)
			}
			newLimits["memory"] = fmt.Sprintf("%.0f", reqMemoryQty.AsApproximateFloat64()*limitRatio.Memory)
			if err := unstructured.SetNestedStringMap(containerMap, newLimits, "resources", "limits"); err != nil {
				return fmt.Errorf("failed to set new limits: %v", err)
			}
			modified = true
		} else {
			newLimits["memory"] = limits["memory"]
		}
		// log.Printf("newLimits %v", newLimits)

		containers[i] = containerMap
	}

	log.Printf("Resource limits not modified")

	if !modified {
		return nil
	}

	if err := unstructured.SetNestedSlice(obj.Object, containers, "spec", "template", "spec", "containers"); err != nil {
		return fmt.Errorf("failed to update containers in the resource: %v", err)
	}
	if err := applyTarget(dynamicClient, targetRef, obj); err != nil {
		return fmt.Errorf("failed to update the resource limits: %v", err)
	}

	log.Printf("Resource limits updated")

	return nil
}

func applyTarget(dynamicClient dynamic.Interface, targetRef TargetRef, obj *unstructured.Unstructured) error {
	gvr, err := getGVR(targetRef)
	if err != nil {
		return err
	}

	// Convert unstructured object to JSON bytes
	data, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal deployment JSON: %s", err)
	}

	// Send a server-side apply request
	force := true
	_, err = dynamicClient.Resource(gvr).Namespace(targetRef.Namespace).Patch(context.Background(), targetRef.Name, types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "example-manager",
		Force:        &force,
	})
	if err != nil {
		return fmt.Errorf("failed to apply %s: %s", targetRef.Kind, err)
	}
	return nil
}

func getGVR(targetRef TargetRef) (schema.GroupVersionResource, error) {
	switch targetRef.Kind {
	case "Deployment":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil
	case "StatefulSet":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unsupported kind: %s", targetRef.Kind)
	}
}
