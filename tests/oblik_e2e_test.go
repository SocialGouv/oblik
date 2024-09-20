package main

import (
	"context"
	"strings"

	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const oblikE2eTestNamespace = "oblik-e2e-test"

func TestOblikFeatures(t *testing.T) {
	t.Logf("Starting TestOblikFeatures")

	clientset, err := setupTestEnvironment(t, oblikE2eTestNamespace)
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	t.Logf("Test environment setup complete")

	for _, otc := range e2eOblikTests {
		otc := otc // capture range variable
		t.Run(otc.name, func(t *testing.T) {
			t.Logf("Starting test: %s", colorize(otc.name, Cyan))
			t.Parallel()
			subCtx, cancel := context.WithTimeout(context.TODO(), 20*time.Minute)
			defer cancel()
			testAnnotationsToResources(subCtx, t, clientset, otc)
			t.Logf("Finished test: %s", otc.name)
		})
	}
}

func testAnnotationsToResources(ctx context.Context, t *testing.T, clientset *kubernetes.Clientset, otc OblikTestCase) {

	appName := strings.ToLower(otc.name)
	labelSelector := map[string]string{"app": appName}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        appName,
			Namespace:   oblikE2eTestNamespace,
			Labels:      labelSelector,
			Annotations: otc.annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labelSelector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelSelector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      "busybox",
							Image:     "busybox:latest",
							Resources: otc.original,
							Command:   []string{"tail", "-f", "/dev/null"},
						},
					},
				},
			},
		},
	}

	_, err := clientset.AppsV1().Deployments(oblikE2eTestNamespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create Deployment: %v", err)
	}
	defer func() {
		err := clientset.AppsV1().Deployments(oblikE2eTestNamespace).Delete(ctx, deployment.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("Failed to delete Deployment: %v", err)
		}
	}()

	originalResource := deployment.Spec.Template.Spec.Containers[0].Resources

	if otc.shouldntUpdate {
		_, err := waitForResourceUpdate(ctx, t, clientset, oblikE2eTestNamespace, "Deployment", deployment.Name, 2*time.Minute, originalResource)
		if err == nil {
			t.Fatalf("Failed waiting for non update: %v", err)
		}
	} else {
		currentResource, err := waitForResourceUpdate(ctx, t, clientset, oblikE2eTestNamespace, "Deployment", deployment.Name, 4*time.Minute, originalResource)
		if err != nil {
			t.Fatalf("Failed waiting for update: %v", err)
		}
		if isDiff(*currentResource, otc.expected) {
			t.Log("Unexpected resources diff actual -> expected:")
			displayExpectedDiff(t, *currentResource, otc.expected)
			t.Error("Resources update does not match expectations")
		}
	}

}
