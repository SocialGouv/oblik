package resourcesconfig

import (
	"context"
	"fmt"
	"strings"

	oblikv1 "github.com/SocialGouv/oblik/pkg/apis/oblik/v1"
	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceNotFoundError is a custom error type for when a resource is not found
type ResourceNotFoundError struct {
	Kind string
	Name string
}

func (e *ResourceNotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Kind, e.Name)
}

// IsResourceNotFoundError checks if an error is a ResourceNotFoundError
func IsResourceNotFoundError(err error) bool {
	_, ok := err.(*ResourceNotFoundError)
	return ok
}

// SyncAnnotations syncs annotations from ResourcesConfig to the target workload
func SyncAnnotations(ctx context.Context, kubeClients *client.KubeClients, rc *oblikv1.ResourcesConfig) error {
	// Find the target workload
	target, err := findTarget(ctx, kubeClients, rc)
	if err != nil {
		return err
	}

	// Get current annotations
	currentAnnotations := target.GetAnnotations()
	if currentAnnotations == nil {
		currentAnnotations = make(map[string]string)
	}

	// Create new annotations map
	newAnnotations := make(map[string]string)

	// If annotation mode is "merge", copy existing annotations
	if rc.Spec.AnnotationMode == "merge" {
		for k, v := range currentAnnotations {
			newAnnotations[k] = v
		}
	} else {
		// If annotation mode is "replace", only copy non-oblik annotations
		for k, v := range currentAnnotations {
			if !strings.HasPrefix(k, constants.PREFIX) {
				newAnnotations[k] = v
			}
		}
	}

	// Add annotations from ResourcesConfig
	addAnnotationsFromResourcesConfig(newAnnotations, rc)

	// Update the target workload
	return updateTargetAnnotations(ctx, kubeClients, target, newAnnotations)
}

// RemoveAnnotations removes all oblik annotations and labels from the target workload
func RemoveAnnotations(ctx context.Context, kubeClients *client.KubeClients, rc *oblikv1.ResourcesConfig) error {
	// Find the target workload
	target, err := findTarget(ctx, kubeClients, rc)
	if err != nil {
		return err
	}

	// Get current annotations
	currentAnnotations := target.GetAnnotations()
	if currentAnnotations == nil {
		currentAnnotations = make(map[string]string)
	}

	// Create new annotations map without oblik annotations
	newAnnotations := make(map[string]string)
	for k, v := range currentAnnotations {
		if !strings.HasPrefix(k, constants.PREFIX) {
			newAnnotations[k] = v
		}
	}

	// Get current labels
	currentLabels := target.GetLabels()
	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}

	// Create new labels map without oblik labels
	newLabels := make(map[string]string)
	for k, v := range currentLabels {
		if !strings.HasPrefix(k, constants.PREFIX) {
			newLabels[k] = v
		}
	}

	// Update the target workload
	return updateTargetWithAnnotationsAndLabels(ctx, kubeClients, target, newAnnotations, newLabels)
}

// findTarget finds the target workload based on the targetRef
func findTarget(ctx context.Context, kubeClients *client.KubeClients, rc *oblikv1.ResourcesConfig) (metav1.Object, error) {
	targetRef := rc.Spec.TargetRef
	namespace := rc.Namespace

	switch targetRef.Kind {
	case "Deployment":
		deployment, err := kubeClients.Clientset.AppsV1().Deployments(namespace).Get(ctx, targetRef.Name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return nil, &ResourceNotFoundError{Kind: "deployments.apps", Name: targetRef.Name}
			}
			return nil, err
		}
		return deployment, nil
	case "StatefulSet":
		statefulSet, err := kubeClients.Clientset.AppsV1().StatefulSets(namespace).Get(ctx, targetRef.Name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return nil, &ResourceNotFoundError{Kind: "statefulsets.apps", Name: targetRef.Name}
			}
			return nil, err
		}
		return statefulSet, nil
	case "DaemonSet":
		daemonSet, err := kubeClients.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, targetRef.Name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return nil, &ResourceNotFoundError{Kind: "daemonsets.apps", Name: targetRef.Name}
			}
			return nil, err
		}
		return daemonSet, nil
	case "CronJob":
		cronJob, err := kubeClients.Clientset.BatchV1().CronJobs(namespace).Get(ctx, targetRef.Name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return nil, &ResourceNotFoundError{Kind: "cronjobs.batch", Name: targetRef.Name}
			}
			return nil, err
		}
		return cronJob, nil
	case "Cluster":
		// Handle CNPG Cluster
		gvr := schema.GroupVersionResource{
			Group:    "postgresql.cnpg.io",
			Version:  "v1",
			Resource: "clusters",
		}
		cluster, err := kubeClients.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, targetRef.Name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return nil, &ResourceNotFoundError{Kind: "clusters.postgresql.cnpg.io", Name: targetRef.Name}
			}
			return nil, err
		}
		return cluster, nil
	default:
		return nil, fmt.Errorf("unsupported target kind: %s", targetRef.Kind)
	}
}

// addAnnotationsFromResourcesConfig adds annotations from ResourcesConfig to the annotations map
func addAnnotationsFromResourcesConfig(annotations map[string]string, rc *oblikv1.ResourcesConfig) {
	// Note: oblik.socialgouv.io/enabled is added as a label in updateTargetAnnotations, not here

	// Add annotations based on ResourcesConfig fields
	if rc.Spec.Cron != "" {
		annotations[constants.PREFIX+"cron"] = rc.Spec.Cron
	}
	if rc.Spec.CronAddRandomMax != "" {
		annotations[constants.PREFIX+"cron-add-random-max"] = rc.Spec.CronAddRandomMax
	}
	if rc.Spec.DryRun {
		annotations[constants.PREFIX+"dry-run"] = "true"
	}
	if rc.Spec.WebhookEnabled {
		annotations[constants.PREFIX+"webhook-enabled"] = "true"
	}
	if rc.Spec.RequestCpuApplyMode != "" {
		annotations[constants.PREFIX+"request-cpu-apply-mode"] = rc.Spec.RequestCpuApplyMode
	}
	if rc.Spec.RequestMemoryApplyMode != "" {
		annotations[constants.PREFIX+"request-memory-apply-mode"] = rc.Spec.RequestMemoryApplyMode
	}
	if rc.Spec.LimitCpuApplyMode != "" {
		annotations[constants.PREFIX+"limit-cpu-apply-mode"] = rc.Spec.LimitCpuApplyMode
	}
	if rc.Spec.LimitMemoryApplyMode != "" {
		annotations[constants.PREFIX+"limit-memory-apply-mode"] = rc.Spec.LimitMemoryApplyMode
	}
	if rc.Spec.LimitCpuCalculatorAlgo != "" {
		annotations[constants.PREFIX+"limit-cpu-calculator-algo"] = rc.Spec.LimitCpuCalculatorAlgo
	}
	if rc.Spec.LimitMemoryCalculatorAlgo != "" {
		annotations[constants.PREFIX+"limit-memory-calculator-algo"] = rc.Spec.LimitMemoryCalculatorAlgo
	}
	if rc.Spec.LimitCpuCalculatorValue != "" {
		annotations[constants.PREFIX+"limit-cpu-calculator-value"] = rc.Spec.LimitCpuCalculatorValue
	}
	if rc.Spec.LimitMemoryCalculatorValue != "" {
		annotations[constants.PREFIX+"limit-memory-calculator-value"] = rc.Spec.LimitMemoryCalculatorValue
	}
	if rc.Spec.UnprovidedApplyDefaultRequestCpu != "" {
		annotations[constants.PREFIX+"unprovided-apply-default-request-cpu"] = rc.Spec.UnprovidedApplyDefaultRequestCpu
	}
	if rc.Spec.UnprovidedApplyDefaultRequestMemory != "" {
		annotations[constants.PREFIX+"unprovided-apply-default-request-memory"] = rc.Spec.UnprovidedApplyDefaultRequestMemory
	}
	if rc.Spec.IncreaseRequestCpuAlgo != "" {
		annotations[constants.PREFIX+"increase-request-cpu-algo"] = rc.Spec.IncreaseRequestCpuAlgo
	}
	if rc.Spec.IncreaseRequestCpuValue != "" {
		annotations[constants.PREFIX+"increase-request-cpu-value"] = rc.Spec.IncreaseRequestCpuValue
	}
	if rc.Spec.IncreaseRequestMemoryAlgo != "" {
		annotations[constants.PREFIX+"increase-request-memory-algo"] = rc.Spec.IncreaseRequestMemoryAlgo
	}
	if rc.Spec.IncreaseRequestMemoryValue != "" {
		annotations[constants.PREFIX+"increase-request-memory-value"] = rc.Spec.IncreaseRequestMemoryValue
	}
	if rc.Spec.MinLimitCpu != "" {
		annotations[constants.PREFIX+"min-limit-cpu"] = rc.Spec.MinLimitCpu
	}
	if rc.Spec.MaxLimitCpu != "" {
		annotations[constants.PREFIX+"max-limit-cpu"] = rc.Spec.MaxLimitCpu
	}
	if rc.Spec.MinLimitMemory != "" {
		annotations[constants.PREFIX+"min-limit-memory"] = rc.Spec.MinLimitMemory
	}
	if rc.Spec.MaxLimitMemory != "" {
		annotations[constants.PREFIX+"max-limit-memory"] = rc.Spec.MaxLimitMemory
	}
	if rc.Spec.MinRequestCpu != "" {
		annotations[constants.PREFIX+"min-request-cpu"] = rc.Spec.MinRequestCpu
	}
	if rc.Spec.MaxRequestCpu != "" {
		annotations[constants.PREFIX+"max-request-cpu"] = rc.Spec.MaxRequestCpu
	}
	if rc.Spec.MinRequestMemory != "" {
		annotations[constants.PREFIX+"min-request-memory"] = rc.Spec.MinRequestMemory
	}
	if rc.Spec.MaxRequestMemory != "" {
		annotations[constants.PREFIX+"max-request-memory"] = rc.Spec.MaxRequestMemory
	}
	if rc.Spec.MinAllowedRecommendationCpu != "" {
		annotations[constants.PREFIX+"min-allowed-recommendation-cpu"] = rc.Spec.MinAllowedRecommendationCpu
	}
	if rc.Spec.MaxAllowedRecommendationCpu != "" {
		annotations[constants.PREFIX+"max-allowed-recommendation-cpu"] = rc.Spec.MaxAllowedRecommendationCpu
	}
	if rc.Spec.MinAllowedRecommendationMemory != "" {
		annotations[constants.PREFIX+"min-allowed-recommendation-memory"] = rc.Spec.MinAllowedRecommendationMemory
	}
	if rc.Spec.MaxAllowedRecommendationMemory != "" {
		annotations[constants.PREFIX+"max-allowed-recommendation-memory"] = rc.Spec.MaxAllowedRecommendationMemory
	}
	if rc.Spec.MinDiffCpuRequestAlgo != "" {
		annotations[constants.PREFIX+"min-diff-cpu-request-algo"] = rc.Spec.MinDiffCpuRequestAlgo
	}
	if rc.Spec.MinDiffCpuRequestValue != "" {
		annotations[constants.PREFIX+"min-diff-cpu-request-value"] = rc.Spec.MinDiffCpuRequestValue
	}
	if rc.Spec.MinDiffMemoryRequestAlgo != "" {
		annotations[constants.PREFIX+"min-diff-memory-request-algo"] = rc.Spec.MinDiffMemoryRequestAlgo
	}
	if rc.Spec.MinDiffMemoryRequestValue != "" {
		annotations[constants.PREFIX+"min-diff-memory-request-value"] = rc.Spec.MinDiffMemoryRequestValue
	}
	if rc.Spec.MinDiffCpuLimitAlgo != "" {
		annotations[constants.PREFIX+"min-diff-cpu-limit-algo"] = rc.Spec.MinDiffCpuLimitAlgo
	}
	if rc.Spec.MinDiffCpuLimitValue != "" {
		annotations[constants.PREFIX+"min-diff-cpu-limit-value"] = rc.Spec.MinDiffCpuLimitValue
	}
	if rc.Spec.MinDiffMemoryLimitAlgo != "" {
		annotations[constants.PREFIX+"min-diff-memory-limit-algo"] = rc.Spec.MinDiffMemoryLimitAlgo
	}
	if rc.Spec.MinDiffMemoryLimitValue != "" {
		annotations[constants.PREFIX+"min-diff-memory-limit-value"] = rc.Spec.MinDiffMemoryLimitValue
	}
	if rc.Spec.MemoryRequestFromCpuEnabled {
		annotations[constants.PREFIX+"memory-request-from-cpu-enabled"] = "true"
	}
	if rc.Spec.MemoryLimitFromCpuEnabled {
		annotations[constants.PREFIX+"memory-limit-from-cpu-enabled"] = "true"
	}
	if rc.Spec.MemoryRequestFromCpuAlgo != "" {
		annotations[constants.PREFIX+"memory-request-from-cpu-algo"] = rc.Spec.MemoryRequestFromCpuAlgo
	}
	if rc.Spec.MemoryRequestFromCpuValue != "" {
		annotations[constants.PREFIX+"memory-request-from-cpu-value"] = rc.Spec.MemoryRequestFromCpuValue
	}
	if rc.Spec.MemoryLimitFromCpuAlgo != "" {
		annotations[constants.PREFIX+"memory-limit-from-cpu-algo"] = rc.Spec.MemoryLimitFromCpuAlgo
	}
	if rc.Spec.MemoryLimitFromCpuValue != "" {
		annotations[constants.PREFIX+"memory-limit-from-cpu-value"] = rc.Spec.MemoryLimitFromCpuValue
	}
	if rc.Spec.RequestApplyTarget != "" {
		annotations[constants.PREFIX+"request-apply-target"] = rc.Spec.RequestApplyTarget
	}
	if rc.Spec.RequestCpuApplyTarget != "" {
		annotations[constants.PREFIX+"request-cpu-apply-target"] = rc.Spec.RequestCpuApplyTarget
	}
	if rc.Spec.RequestMemoryApplyTarget != "" {
		annotations[constants.PREFIX+"request-memory-apply-target"] = rc.Spec.RequestMemoryApplyTarget
	}
	if rc.Spec.LimitApplyTarget != "" {
		annotations[constants.PREFIX+"limit-apply-target"] = rc.Spec.LimitApplyTarget
	}
	if rc.Spec.LimitCpuApplyTarget != "" {
		annotations[constants.PREFIX+"limit-cpu-apply-target"] = rc.Spec.LimitCpuApplyTarget
	}
	if rc.Spec.LimitMemoryApplyTarget != "" {
		annotations[constants.PREFIX+"limit-memory-apply-target"] = rc.Spec.LimitMemoryApplyTarget
	}
	if rc.Spec.RequestCpuScaleDirection != "" {
		annotations[constants.PREFIX+"request-cpu-scale-direction"] = rc.Spec.RequestCpuScaleDirection
	}
	if rc.Spec.RequestMemoryScaleDirection != "" {
		annotations[constants.PREFIX+"request-memory-scale-direction"] = rc.Spec.RequestMemoryScaleDirection
	}
	if rc.Spec.LimitCpuScaleDirection != "" {
		annotations[constants.PREFIX+"limit-cpu-scale-direction"] = rc.Spec.LimitCpuScaleDirection
	}
	if rc.Spec.LimitMemoryScaleDirection != "" {
		annotations[constants.PREFIX+"limit-memory-scale-direction"] = rc.Spec.LimitMemoryScaleDirection
	}

	// Add direct resource specifications (flat style)
	if rc.Spec.RequestCpu != "" {
		annotations[constants.PREFIX+"request-cpu"] = rc.Spec.RequestCpu
	}
	if rc.Spec.RequestMemory != "" {
		annotations[constants.PREFIX+"request-memory"] = rc.Spec.RequestMemory
	}
	if rc.Spec.LimitCpu != "" {
		annotations[constants.PREFIX+"limit-cpu"] = rc.Spec.LimitCpu
	}
	if rc.Spec.LimitMemory != "" {
		annotations[constants.PREFIX+"limit-memory"] = rc.Spec.LimitMemory
	}

	// Add Kubernetes-native style resource specifications (nested)
	if rc.Spec.Request != nil {
		if rc.Spec.Request.CPU != "" {
			annotations[constants.PREFIX+"request-cpu"] = rc.Spec.Request.CPU
		}
		if rc.Spec.Request.Memory != "" {
			annotations[constants.PREFIX+"request-memory"] = rc.Spec.Request.Memory
		}
	}
	if rc.Spec.Limit != nil {
		if rc.Spec.Limit.CPU != "" {
			annotations[constants.PREFIX+"limit-cpu"] = rc.Spec.Limit.CPU
		}
		if rc.Spec.Limit.Memory != "" {
			annotations[constants.PREFIX+"limit-memory"] = rc.Spec.Limit.Memory
		}
	}

	// Handle container-specific configurations
	if rc.Spec.ContainerConfigs != nil {
		for containerName, containerConfig := range rc.Spec.ContainerConfigs {
			// Direct resource specifications (flat style)
			if containerConfig.RequestCpu != "" {
				annotations[constants.PREFIX+"request-cpu."+containerName] = containerConfig.RequestCpu
			}
			if containerConfig.RequestMemory != "" {
				annotations[constants.PREFIX+"request-memory."+containerName] = containerConfig.RequestMemory
			}
			if containerConfig.LimitCpu != "" {
				annotations[constants.PREFIX+"limit-cpu."+containerName] = containerConfig.LimitCpu
			}
			if containerConfig.LimitMemory != "" {
				annotations[constants.PREFIX+"limit-memory."+containerName] = containerConfig.LimitMemory
			}
			
			// Kubernetes-native style resource specifications (nested)
			if containerConfig.Request != nil {
				if containerConfig.Request.CPU != "" {
					annotations[constants.PREFIX+"request-cpu."+containerName] = containerConfig.Request.CPU
				}
				if containerConfig.Request.Memory != "" {
					annotations[constants.PREFIX+"request-memory."+containerName] = containerConfig.Request.Memory
				}
			}
			if containerConfig.Limit != nil {
				if containerConfig.Limit.CPU != "" {
					annotations[constants.PREFIX+"limit-cpu."+containerName] = containerConfig.Limit.CPU
				}
				if containerConfig.Limit.Memory != "" {
					annotations[constants.PREFIX+"limit-memory."+containerName] = containerConfig.Limit.Memory
				}
			}
			
			// Original container-specific configurations
			if containerConfig.MinLimitCpu != "" {
				annotations[constants.PREFIX+"min-limit-cpu."+containerName] = containerConfig.MinLimitCpu
			}
			if containerConfig.MaxLimitCpu != "" {
				annotations[constants.PREFIX+"max-limit-cpu."+containerName] = containerConfig.MaxLimitCpu
			}
			if containerConfig.MinLimitMemory != "" {
				annotations[constants.PREFIX+"min-limit-memory."+containerName] = containerConfig.MinLimitMemory
			}
			if containerConfig.MaxLimitMemory != "" {
				annotations[constants.PREFIX+"max-limit-memory."+containerName] = containerConfig.MaxLimitMemory
			}
			if containerConfig.MinRequestCpu != "" {
				annotations[constants.PREFIX+"min-request-cpu."+containerName] = containerConfig.MinRequestCpu
			}
			if containerConfig.MaxRequestCpu != "" {
				annotations[constants.PREFIX+"max-request-cpu."+containerName] = containerConfig.MaxRequestCpu
			}
			if containerConfig.MinRequestMemory != "" {
				annotations[constants.PREFIX+"min-request-memory."+containerName] = containerConfig.MinRequestMemory
			}
			if containerConfig.MaxRequestMemory != "" {
				annotations[constants.PREFIX+"max-request-memory."+containerName] = containerConfig.MaxRequestMemory
			}
			if containerConfig.MinAllowedRecommendationCpu != "" {
				annotations[constants.PREFIX+"min-allowed-recommendation-cpu."+containerName] = containerConfig.MinAllowedRecommendationCpu
			}
			if containerConfig.MaxAllowedRecommendationCpu != "" {
				annotations[constants.PREFIX+"max-allowed-recommendation-cpu."+containerName] = containerConfig.MaxAllowedRecommendationCpu
			}
			if containerConfig.MinAllowedRecommendationMemory != "" {
				annotations[constants.PREFIX+"min-allowed-recommendation-memory."+containerName] = containerConfig.MinAllowedRecommendationMemory
			}
			if containerConfig.MaxAllowedRecommendationMemory != "" {
				annotations[constants.PREFIX+"max-allowed-recommendation-memory."+containerName] = containerConfig.MaxAllowedRecommendationMemory
			}
		}
	}
}

// updateTargetAnnotations updates the annotations and labels on the target workload
func updateTargetAnnotations(ctx context.Context, kubeClients *client.KubeClients, target metav1.Object, annotations map[string]string) error {
	// Add the enabled label
	labels := target.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[constants.PREFIX+"enabled"] = "true"

	// Update the target workload with annotations and labels
	return updateTargetWithAnnotationsAndLabels(ctx, kubeClients, target, annotations, labels)
}

// updateTargetWithAnnotationsAndLabels updates the annotations and labels on the target workload
func updateTargetWithAnnotationsAndLabels(ctx context.Context, kubeClients *client.KubeClients, target metav1.Object, annotations, labels map[string]string) error {
	switch t := target.(type) {
	case *appsv1.Deployment:
		t.SetAnnotations(annotations)
		t.SetLabels(labels)
		_, err := kubeClients.Clientset.AppsV1().Deployments(t.GetNamespace()).Update(ctx, t, metav1.UpdateOptions{})
		return err
	case *appsv1.StatefulSet:
		t.SetAnnotations(annotations)
		t.SetLabels(labels)
		_, err := kubeClients.Clientset.AppsV1().StatefulSets(t.GetNamespace()).Update(ctx, t, metav1.UpdateOptions{})
		return err
	case *appsv1.DaemonSet:
		t.SetAnnotations(annotations)
		t.SetLabels(labels)
		_, err := kubeClients.Clientset.AppsV1().DaemonSets(t.GetNamespace()).Update(ctx, t, metav1.UpdateOptions{})
		return err
	case *batchv1.CronJob:
		t.SetAnnotations(annotations)
		t.SetLabels(labels)
		_, err := kubeClients.Clientset.BatchV1().CronJobs(t.GetNamespace()).Update(ctx, t, metav1.UpdateOptions{})
		return err
	case *unstructured.Unstructured:
		t.SetAnnotations(annotations)
		t.SetLabels(labels)
		gvr := schema.GroupVersionResource{
			Group:    "postgresql.cnpg.io",
			Version:  "v1",
			Resource: "clusters",
		}
		_, err := kubeClients.DynamicClient.Resource(gvr).Namespace(t.GetNamespace()).Update(ctx, t, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("unsupported target type: %T", target)
	}
}
