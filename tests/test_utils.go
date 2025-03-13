package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/term"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
)

var (
	colorEnabled bool
)

func init() {
	colorEnabled = isTerminal()
	if val, ok := os.LookupEnv("FORCE_COLOR"); ok {
		switch val {
		case "1", "true", "yes":
			colorEnabled = true
		case "0", "false", "no":
			colorEnabled = false
		}
	}
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) || term.IsTerminal(int(os.Stderr.Fd()))
}

func colorize(text, color string) string {
	if colorEnabled {
		return color + text + Reset
	}
	return text
}

func int32Ptr(i int32) *int32 {
	return &i
}

func createKubeClient() (*kubernetes.Clientset, *vpaclientset.Clientset, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	config, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load Kubernetes client configuration: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	vpaClientset, err := vpaclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes VPA client: %v", err)
	}

	return clientset, vpaClientset, nil
}

func createNamespace(t *testing.T, clientset *kubernetes.Clientset, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Failed to create namespace: %v", err)
	}
	return nil
}

func deleteNamespace(t *testing.T, clientset *kubernetes.Clientset, namespace string) error {
	// Check if the namespace exists
	_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Namespace doesn't exist, no need to delete
			t.Logf("Namespace %s does not exist, skipping deletion", namespace)
			return nil
		}
		// Other error occurred while checking for namespace
		return fmt.Errorf("failed to check if namespace exists: %v", err)
	}

	// Namespace exists, proceed with deletion
	deleteNsGracePeriodSeconds := int64(1)
	err = clientset.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{
		GracePeriodSeconds: &deleteNsGracePeriodSeconds,
	})
	if err != nil {
		return fmt.Errorf("failed to delete existing namespace: %v", err)
	}
	t.Logf("Namespace %s deleted successfully", namespace)
	return nil
}

func setupTestEnvironment(t *testing.T, ns string) (*kubernetes.Clientset, *vpaclientset.Clientset, error) {
	t.Logf("Creating Kubernetes client")
	clientset, vpaClientset, err := createKubeClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	t.Logf("Deleting existing namespace if it exists")
	err = deleteNamespace(t, clientset, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete existing namespace: %v", err)
	}

	t.Logf("Creating new namespace: %s", ns)
	err = createNamespace(t, clientset, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create namespace: %v", err)
	}

	t.Cleanup(func() {
		t.Logf("Cleaning up test namespace")
		err := deleteNamespace(t, clientset, ns)
		if err != nil {
			t.Logf("Failed to delete namespace during cleanup: %v", err)
		}
	})

	return clientset, vpaClientset, nil
}

func getResource(clientset *kubernetes.Clientset, namespace, kind, name string) (metav1.Object, error) {
	switch kind {
	case "Deployment":
		return clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	case "StatefulSet":
		return clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	case "CronJob":
		return clientset.BatchV1().CronJobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("unsupported resource kind: %s", kind)
	}
}

func waitForResourceUpdate(ctx context.Context, t *testing.T, clientset *kubernetes.Clientset, namespace, kind, name string, timeout time.Duration, originalResource corev1.ResourceRequirements) (*corev1.ResourceRequirements, error) {
	t.Logf("Waiting for %s %s to update", kind, name)
	backoff := time.Second * 2
	endTime := time.Now().Add(timeout)

	for time.Now().Before(endTime) {
		// Get the current resource
		obj, err := getResource(clientset, namespace, kind, name)
		if err != nil {
			t.Logf("Error getting %s %s: %v", kind, name, err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// Extract current resources from the object
		var currentResource corev1.ResourceRequirements
		switch obj := obj.(type) {
		case *appsv1.Deployment:
			currentResource = obj.Spec.Template.Spec.Containers[0].Resources
		case *appsv1.StatefulSet:
			currentResource = obj.Spec.Template.Spec.Containers[0].Resources
		}

		// Compare only the resources (CPU and memory requests/limits)
		if isDiff(originalResource, currentResource) {
			t.Logf("%s %s updated successfully", kind, name)
			t.Logf("Update differences:")
			displayDiff(t, originalResource, currentResource)
			return &currentResource, nil
		}

		t.Logf("%s %s not yet updated, waiting...", kind, name)
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}

	return nil, fmt.Errorf("timeout waiting for %s %s to update", kind, name)
}

func displayDiff(t *testing.T, original, current corev1.ResourceRequirements) {
	if !original.Requests.Cpu().Equal(*current.Requests.Cpu()) {
		t.Logf("  CPU Request: %v -> %v", original.Requests.Cpu().String(), current.Requests.Cpu().String())
	} else {
		t.Log("  CPU Request: OK")
	}
	if !original.Requests.Memory().Equal(*current.Requests.Memory()) {
		t.Logf("  Memory Request: %v -> %v", original.Requests.Memory(), current.Requests.Memory())
	} else {
		t.Log("  Memory Request: OK")
	}
	if !original.Limits.Cpu().Equal(*current.Limits.Cpu()) {
		t.Logf("  CPU Limit: %v -> %v", original.Limits.Cpu(), current.Limits.Cpu())
	} else {
		t.Log("  CPU Limit: OK")
	}
	if !original.Limits.Memory().Equal(*current.Limits.Memory()) {
		t.Logf("  Memory Limit: %v -> %v", original.Limits.Memory(), current.Limits.Memory())
	} else {
		t.Log("  Memory Limit: OK")
	}
}

func displayExpectedDiff(t *testing.T, current, expected corev1.ResourceRequirements) {
	if !current.Requests.Cpu().Equal(*expected.Requests.Cpu()) {
		t.Logf("  CPU Request: %v -> %v", colorize(current.Requests.Cpu().String(), Red), colorize(expected.Requests.Cpu().String(), Green))
	} else {
		t.Log("  CPU Request: OK")
	}
	if !current.Requests.Memory().Equal(*expected.Requests.Memory()) {
		t.Logf("  Memory Request: %v -> %v", colorize(current.Requests.Memory().String(), Red), colorize(expected.Requests.Memory().String(), Green))
	} else {
		t.Log("  Memory Request: OK")
	}
	if !current.Limits.Cpu().Equal(*expected.Limits.Cpu()) {
		t.Logf("  CPU Limit: %v -> %v", colorize(current.Limits.Cpu().String(), Red), colorize(expected.Limits.Cpu().String(), Green))
	} else {
		t.Log("  CPU Limit: OK")
	}
	if !current.Limits.Memory().Equal(*expected.Limits.Memory()) {
		t.Logf("  Memory Limit: %v -> %v", colorize(current.Limits.Memory().String(), Red), colorize(expected.Limits.Memory().String(), Green))
	} else {
		t.Log("  Memory Limit: OK")
	}
}

func isDiff(original, current corev1.ResourceRequirements) bool {
	if !original.Requests.Cpu().Equal(*current.Requests.Cpu()) {
		return true
	}
	if !original.Limits.Cpu().Equal(*current.Limits.Cpu()) {
		return true
	}
	if !original.Requests.Memory().Equal(*current.Requests.Memory()) {
		return true
	}
	if !original.Limits.Memory().Equal(*current.Limits.Memory()) {
		return true
	}
	return false
}

// generateVPAName generates a VPA name from a workload kind and name
func generateVPAName(kind, name string) string {
	vpaName := fmt.Sprintf("oblik-%s-%s", strings.ToLower(kind), name)
	if len(vpaName) > 63 {
		hash := sha256.Sum256([]byte(vpaName))
		truncatedHash := fmt.Sprintf("%x", hash)[:8]
		vpaName = vpaName[:54] + "-" + truncatedHash
	}
	return vpaName
}

func waitForAndUpdateVPA(ctx context.Context, t *testing.T, vpaClientset *vpaclientset.Clientset, namespace, name string, cpuRecommendation, memoryRecommendation string) error {
	// Generate the VPA name
	vpaName := generateVPAName("Deployment", name)
	t.Logf("Waiting for VPA %s to be created...", vpaName)

	// List all VPAs in the namespace to see what's available
	vpaList, err := vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Logf("Error listing VPAs: %v", err)
	} else {
		t.Logf("Found %d VPAs in namespace %s", len(vpaList.Items), namespace)
		for _, v := range vpaList.Items {
			t.Logf("  VPA: %s", v.Name)
		}
	}

	// Wait for the VPA to be created by the controller
	var vpa *autoscalingv1.VerticalPodAutoscaler
	backoff := time.Second * 2
	endTime := time.Now().Add(5 * time.Minute)

	for time.Now().Before(endTime) {
		var err error
		vpa, err = vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Get(ctx, vpaName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				t.Logf("VPA %s not found yet, waiting...", vpaName)

				// List all VPAs again to see if any new ones were created
				vpaList, listErr := vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
				if listErr == nil && len(vpaList.Items) > 0 {
					t.Logf("Found %d VPAs in namespace %s while waiting", len(vpaList.Items), namespace)
					for _, v := range vpaList.Items {
						t.Logf("  VPA: %s", v.Name)
					}
				}

				time.Sleep(backoff)
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			return fmt.Errorf("error getting VPA: %v", err)
		}

		t.Logf("VPA %s found", vpaName)
		break
	}

	if vpa == nil {
		return fmt.Errorf("timeout waiting for VPA %s to be created", vpaName)
	}

	t.Logf("Found VPA: %s", vpa.Name)

	// Parse the CPU and memory recommendations
	cpuQuantity, err := resource.ParseQuantity(cpuRecommendation)
	if err != nil {
		return fmt.Errorf("failed to parse CPU recommendation: %v", err)
	}

	memoryQuantity, err := resource.ParseQuantity(memoryRecommendation)
	if err != nil {
		return fmt.Errorf("failed to parse memory recommendation: %v", err)
	}

	// Create a copy of the VPA to modify
	updatedVPA := vpa.DeepCopy()

	// Set minAllowed to our custom recommendations
	// This will influence the VPA's recommendations
	if updatedVPA.Spec.ResourcePolicy == nil {
		updatedVPA.Spec.ResourcePolicy = &autoscalingv1.PodResourcePolicy{
			ContainerPolicies: []autoscalingv1.ContainerResourcePolicy{},
		}
	}

	containerPolicy := autoscalingv1.ContainerResourcePolicy{
		ContainerName: "*", // Apply to all containers
		MinAllowed: corev1.ResourceList{
			corev1.ResourceCPU:    cpuQuantity,
			corev1.ResourceMemory: memoryQuantity,
		},
		MaxAllowed: corev1.ResourceList{
			corev1.ResourceCPU:    cpuQuantity,
			corev1.ResourceMemory: memoryQuantity,
		},
	}

	// Add or update the container policy
	found := false
	for i, policy := range updatedVPA.Spec.ResourcePolicy.ContainerPolicies {
		if policy.ContainerName == "*" {
			updatedVPA.Spec.ResourcePolicy.ContainerPolicies[i] = containerPolicy
			found = true
			break
		}
	}

	if !found {
		updatedVPA.Spec.ResourcePolicy.ContainerPolicies = append(
			updatedVPA.Spec.ResourcePolicy.ContainerPolicies,
			containerPolicy,
		)
	}

	// Update the VPA
	_, err = vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Update(ctx, updatedVPA, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update VPA: %v", err)
	}

	t.Logf("Updated VPA %s with custom recommendations: CPU=%s, Memory=%s", vpaName, cpuRecommendation, memoryRecommendation)

	// Wait for the VPA to process the update and for the recommendations to be aligned with the minAllowed values
	t.Logf("Waiting for VPA recommendations to be aligned with minAllowed values")
	backoff = time.Second * 2
	endTime = time.Now().Add(5 * time.Minute)

	for time.Now().Before(endTime) {
		// Get the latest VPA
		vpa, err = vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Get(ctx, vpaName, metav1.GetOptions{})
		if err != nil {
			t.Logf("Error getting VPA: %v", err)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}

		// Check if the recommendations are aligned with the minAllowed values
		if vpa.Status.Recommendation == nil || len(vpa.Status.Recommendation.ContainerRecommendations) == 0 {
			t.Logf("VPA %s has no recommendations yet, waiting...", vpaName)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}

		aligned := true
		for _, containerRec := range vpa.Status.Recommendation.ContainerRecommendations {
			// Check CPU recommendation
			if containerRec.Target.Cpu().Cmp(cpuQuantity) != 0 {
				t.Logf("VPA %s CPU recommendation (%s) not aligned with minAllowed (%s) yet, waiting...",
					vpaName, containerRec.Target.Cpu().String(), cpuQuantity.String())
				aligned = false
				break
			}

			// Check Memory recommendation
			if containerRec.Target.Memory().Cmp(memoryQuantity) != 0 {
				t.Logf("VPA %s Memory recommendation (%s) not aligned with minAllowed (%s) yet, waiting...",
					vpaName, containerRec.Target.Memory().String(), memoryQuantity.String())
				aligned = false
				break
			}
		}

		if aligned {
			t.Logf("VPA %s recommendations are now aligned with minAllowed values", vpaName)
			break
		}

		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}

	return nil
}
