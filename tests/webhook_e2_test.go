package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const testNamespace = "webhook-e2e-test"

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

func createNamespace(t *testing.T, clientset *kubernetes.Clientset, namespace string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}
}

func deleteNamespace(t *testing.T, clientset *kubernetes.Clientset, namespace string) {
	err := clientset.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete namespace: %v", err)
	}
}

func TestWebhook(t *testing.T) {
	clientset, err := createKubeClient()
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Clean up the namespace if it already exists
	err = clientset.CoreV1().Namespaces().Delete(context.TODO(), testNamespace, metav1.DeleteOptions{})
	if err == nil {
		time.Sleep(10 * time.Second) // Wait for the namespace to be deleted
	}

	createNamespace(t, clientset, testNamespace)
	defer deleteNamespace(t, clientset, testNamespace)

	labelSelector := map[string]string{"app": "webhook-test"}

	test := struct {
		kind     string
		resource metav1.Object
		check    func(metav1.Object) bool
	}{
		kind: "Deployment",
		resource: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-test-deployment",
				Namespace: testNamespace,
				Labels:    labelSelector,
				Annotations: map[string]string{
					"oblik.socialgouv.io/enabled": "true",
				},
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
							{Name: "nginx", Image: "nginx:latest"},
						},
					},
				},
			},
		},
		check: func(obj metav1.Object) bool {
			deployment := obj.(*appsv1.Deployment)
			resources := deployment.Spec.Template.Spec.Containers[0].Resources
			fmt.Printf("Deployment resources: %v\n", resources)
			return true
		},
	}

	t.Run(test.kind, func(t *testing.T) {
		createResource(t, clientset, testNamespace, test.kind, test.resource)
		time.Sleep(10 * time.Second)

		checkResource(t, clientset, testNamespace, test.kind, test.resource.GetName(), test.check)
		cleanupResource(t, clientset, testNamespace, test.kind, test.resource.GetName())
	})
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

func checkResource(t *testing.T, clientset *kubernetes.Clientset, namespace, kind, name string, check func(metav1.Object) bool) {
	var obj metav1.Object
	var err error
	switch kind {
	case "Deployment":
		obj, err = clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	}
	if err != nil {
		t.Fatalf("Failed to get %s: %v", kind, err)
	}

	if !check(obj) {
		t.Fatalf("%s annotations not as expected: %v", kind, obj.GetAnnotations())
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

func int32Ptr(i int32) *int32 {
	return &i
}
