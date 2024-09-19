package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/term"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// isTerminal checks if the output is a terminal
func isTerminal() bool {
	// Check if either stdout or stderr is a terminal
	return term.IsTerminal(int(os.Stdout.Fd())) || term.IsTerminal(int(os.Stderr.Fd()))
}

func colorize(text, color string) string {
	return color + text + Reset
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func int32Ptr(i int32) *int32 {
	return &i
}

func createKubeClient() (*kubernetes.Clientset, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	config, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Kubernetes client configuration: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	return clientset, nil
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
	deleteNsGracePeriodSeconds := int64(1)
	err := clientset.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{
		GracePeriodSeconds: &deleteNsGracePeriodSeconds,
	})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete existing namespace: %v", err)
	}
	return nil
}

func createResource(t *testing.T, clientset *kubernetes.Clientset, namespace, kind string, resource metav1.Object) {
	switch kind {
	case "Deployment":
		_, err := clientset.AppsV1().Deployments(namespace).Create(context.TODO(), resource.(*appsv1.Deployment), metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create Deployment: %v", err)
		}
	}
}

func cleanupResource(t *testing.T, clientset *kubernetes.Clientset, namespace, kind, name string) {
	switch kind {
	case "Deployment":
		err := clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("Failed to delete Deployment: %v", err)
		}
	}
}

func setupTestEnvironment(t *testing.T, ns string) (*kubernetes.Clientset, error) {
	t.Logf("Creating Kubernetes client")
	clientset, err := createKubeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	t.Logf("Deleting existing namespace if it exists")
	err = deleteNamespace(t, clientset, ns)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing namespace: %v", err)
	}

	if err == nil {
		t.Logf("Waiting for namespace deletion")
		time.Sleep(5 * time.Second)
	}

	t.Logf("Creating new namespace: %s", ns)
	err = createNamespace(t, clientset, ns)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %v", err)
	}

	t.Cleanup(func() {
		t.Logf("Cleaning up test namespace")
		err := deleteNamespace(t, clientset, ns)
		if err != nil {
			t.Logf("Failed to delete namespace during cleanup: %v", err)
		}
	})

	return clientset, nil
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
