package main

import (
	"context"
	"flag"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpa_types "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	vpa_clientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

const oblikE2eTestNamespace = "oblik-e2e-test"

var testCaseName = flag.String("test-case", "", "Specific test case to run")

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
			t.Parallel()
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
		err = vpaClientset.AutoscalingV1().VerticalPodAutoscalers(oblikE2eTestNamespace).Delete(ctx, deployment.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Failed to delete VPA: %v", err)
		}
	}()

	// Create a channel to signal when VPA creation is complete
	vpaCreated := make(chan struct{})

	// Create VPA asynchronously with random delay
	var wg sync.WaitGroup
	wg.Add(1)
	t.Run("CreateVPA", func(t *testing.T) {
		defer wg.Done()
		delay := time.Duration(rand.Intn(21)) * time.Second
		t.Logf("Delaying VPA creation for %v", delay)
		time.Sleep(delay)

		vpa := generateVPA(appName, otc.annotations)
		_, err := vpaClientset.AutoscalingV1().VerticalPodAutoscalers(oblikE2eTestNamespace).Create(ctx, vpa, metav1.CreateOptions{})
		if err != nil {
			t.Errorf("Failed to create VPA: %v", err)
			return
		}
		t.Logf("VPA created after delay")
		close(vpaCreated)
	})

	// Wait for VPA creation to complete
	select {
	case <-vpaCreated:
		t.Log("VPA creation completed")
	case <-ctx.Done():
		t.Fatal("Context cancelled before VPA creation completed")
	}

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

	// Wait for the VPA creation goroutine to finish
	wg.Wait()
}

func generateVPA(name string, annotations map[string]string) *vpa_types.VerticalPodAutoscaler {
	updateMode := vpa_types.UpdateModeOff
	vpa := &vpa_types.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"oblik.socialgouv.io/enabled": "true",
			},
			Annotations: annotations,
		},
		Spec: vpa_types.VerticalPodAutoscalerSpec{
			TargetRef: &autoscaling.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       name,
			},
			UpdatePolicy: &vpa_types.PodUpdatePolicy{
				UpdateMode: &updateMode,
			},
			ResourcePolicy: &vpa_types.PodResourcePolicy{
				ContainerPolicies: []vpa_types.ContainerResourcePolicy{
					{
						ContainerName: "*",
					},
				},
			},
		},
	}
	return vpa
}
