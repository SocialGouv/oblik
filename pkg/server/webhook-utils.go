package server

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

func createJSONPatch(originalJSON []byte, modified *unstructured.Unstructured) ([]byte, error) {
	var original map[string]interface{}
	if err := json.Unmarshal(originalJSON, &original); err != nil {
		return nil, fmt.Errorf("could not unmarshal original object: %v", err)
	}

	patch := []map[string]interface{}{}

	// Compare and create patch operations
	compareMaps("", original, modified.Object, &patch)

	// If no changes are needed, return an empty patch
	if len(patch) == 0 {
		return []byte("[]"), nil
	}

	return json.Marshal(patch)
}

func compareMaps(prefix string, original, modified map[string]interface{}, patch *[]map[string]interface{}) {
	for key, modifiedValue := range modified {
		path := utils.GetJSONPath(prefix, key)
		originalValue, exists := original[key]

		if !exists {
			// Add operation for new fields
			*patch = append(*patch, map[string]interface{}{
				"op":    "add",
				"path":  path,
				"value": modifiedValue,
			})
		} else if !reflect.DeepEqual(originalValue, modifiedValue) {
			// Replace operation for changed fields
			switch modifiedValue.(type) {
			case map[string]interface{}:
				// Recursively compare nested maps
				if originalMap, ok := originalValue.(map[string]interface{}); ok {
					compareMaps(path, originalMap, modifiedValue.(map[string]interface{}), patch)
				} else {
					// If types don't match, replace the entire value
					*patch = append(*patch, map[string]interface{}{
						"op":    "replace",
						"path":  path,
						"value": modifiedValue,
					})
				}
			default:
				// For non-map types, use replace operation
				*patch = append(*patch, map[string]interface{}{
					"op":    "replace",
					"path":  path,
					"value": modifiedValue,
				})
			}
		}
	}

	// Check for removed fields
	for key := range original {
		if _, exists := modified[key]; !exists {
			*patch = append(*patch, map[string]interface{}{
				"op":   "remove",
				"path": utils.GetJSONPath(prefix, key),
			})
		}
	}
}

func getVPAResource(obj *unstructured.Unstructured, kubeClients *client.KubeClients) *autoscalingv1.VerticalPodAutoscaler {
	namespace := obj.GetNamespace()
	vpaName := obj.GetName()
	vpa, err := kubeClients.VpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Get(context.TODO(), vpaName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("No VPA resource found for %s/%s: %v", namespace, vpaName, err)
		} else {
			klog.Errorf("Error retrieving VPA for %s/%s: %v", namespace, vpaName, err)
		}
		return nil
	}
	return vpa
}
