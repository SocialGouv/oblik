package main

import (
	"context"
	"flag"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpa_clientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

const oblikE2eTestNamespace = "oblik-e2e-test"

var testCaseName = flag.String("test-case", "", "Specific test case to run")
var noParallel = flag.Bool("no-parallel", false, "Disable parallel execution")

func TestOblikFeatures(t *testing.T) {
	flag.Parse()
	t.Logf("Starting TestOblikFeatures")

	testClientset, vpaClientset, err := setupTestEnvironment(t, oblikE2eTestNamespace)
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	t.Logf("Test environment setup complete")

	var found bool
	for _, otc := range e2eOblikTests {
		otc := otc // capture range variable
		if *testCaseName != "" && otc.name != *testCaseName {
			continue // Skip tests that don't match the specified test case
		}

		found = true
		t.Run(otc.name, func(t *testing.T) {
			t.Logf("Starting test: %s", colorize(otc.name, Cyan))
			if !*noParallel {
				t.Parallel()
			}
			subCtx, cancel := context.WithTimeout(context.TODO(), 20*time.Minute)
			defer cancel()
			testAnnotationsToResources(subCtx, t, testClientset, vpaClientset, otc)
			t.Logf("Finished test: %s", otc.name)
		})
	}

	if !found {
		t.Logf("No test case found for name: %s", *testCaseName)
	}
}

func testAnnotationsToResources(ctx context.Context, t *testing.T, clientset *kubernetes.Clientset, vpaClientset *vpa_clientset.Clientset, otc OblikTestCase) {
	appName := strings.ToLower(otc.name)
	labelSelector := map[string]string{"app": appName}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        appName,
			Namespace:   oblikE2eTestNamespace,
			Labels:      map[string]string{"app": appName, "oblik.socialgouv.io/enabled": "true"},
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
		currentResource, err := waitForResourceUpdate(ctx, t, clientset, oblikE2eTestNamespace, "Deployment", deployment.Name, 10*time.Minute, originalResource)
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
