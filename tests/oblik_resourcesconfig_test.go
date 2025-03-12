package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	oblikv1 "github.com/SocialGouv/oblik/pkg/apis/oblik/v1"
	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/constants"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	vpa_clientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const resourcesConfigTestNamespace = "oblik-resourcesconfig-test"

// ResourcesConfigSpecYAML represents the ResourcesConfig spec in YAML
type ResourcesConfigSpecYAML struct {
	TargetRef        map[string]string            `yaml:"targetRef"`
	AnnotationMode   string                       `yaml:"annotationMode"`
	ContainerConfigs map[string]map[string]string `yaml:"containerConfigs"`
	// Add all the fields from ResourcesConfigSpec that we want to test
	MinRequestCpu           string `yaml:"minRequestCpu"`
	MinRequestMemory        string `yaml:"minRequestMemory"`
	LimitCpuCalculatorAlgo  string `yaml:"limitCpuCalculatorAlgo"`
	LimitCpuCalculatorValue string `yaml:"limitCpuCalculatorValue"`
	LimitCpuApplyMode       string `yaml:"limitCpuApplyMode"`
}

// ResourcesConfigTestCaseYAML represents each test case in YAML
type ResourcesConfigTestCaseYAML struct {
	Name                string                   `yaml:"name"`
	ResourcesConfig     ResourcesConfigSpecYAML  `yaml:"resourcesConfig"`
	Original            ResourceRequirementsYAML `yaml:"original"`
	Expected            ResourceRequirementsYAML `yaml:"expected"`
	InitialAnnotations  map[string]string        `yaml:"initialAnnotations"`
	DeleteTest          bool                     `yaml:"deleteTest"`
	ExpectedAfterDelete ResourceRequirementsYAML `yaml:"expectedAfterDelete"`
}

// ResourcesConfigTestCasesYAML is the top-level structure for the YAML file
type ResourcesConfigTestCasesYAML struct {
	TestCases []ResourcesConfigTestCaseYAML `yaml:"test_cases"`
}

// ResourcesConfigTestCase represents a test case for ResourcesConfig
type ResourcesConfigTestCase struct {
	name                string
	resourcesConfig     ResourcesConfigSpecYAML
	original            corev1.ResourceRequirements
	expected            corev1.ResourceRequirements
	initialAnnotations  map[string]string
	deleteTest          bool
	expectedAfterDelete corev1.ResourceRequirements
}

// LoadResourcesConfigTestCasesFromYAML loads ResourcesConfig test cases from a YAML file
func LoadResourcesConfigTestCasesFromYAML(filename string) ([]ResourcesConfigTestCase, error) {
	// Read the YAML file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal the YAML into ResourcesConfigTestCasesYAML struct
	var testCasesYAML ResourcesConfigTestCasesYAML
	err = yaml.Unmarshal(data, &testCasesYAML)
	if err != nil {
		return nil, err
	}

	// Convert ResourcesConfigTestCasesYAML to []ResourcesConfigTestCase
	var testCases []ResourcesConfigTestCase
	for _, tcYAML := range testCasesYAML.TestCases {
		// Convert Original and Expected resource requirements
		originalResources, err := parseResourceRequirements(tcYAML.Original)
		if err != nil {
			return nil, fmt.Errorf("error parsing original resources in test case %s: %v", tcYAML.Name, err)
		}

		expectedResources, err := parseResourceRequirements(tcYAML.Expected)
		if err != nil {
			return nil, fmt.Errorf("error parsing expected resources in test case %s: %v", tcYAML.Name, err)
		}

		var expectedAfterDeleteResources corev1.ResourceRequirements
		if tcYAML.DeleteTest {
			expectedAfterDeleteResources, err = parseResourceRequirements(tcYAML.ExpectedAfterDelete)
			if err != nil {
				return nil, fmt.Errorf("error parsing expectedAfterDelete resources in test case %s: %v", tcYAML.Name, err)
			}
		}

		testCase := ResourcesConfigTestCase{
			name:                tcYAML.Name,
			resourcesConfig:     tcYAML.ResourcesConfig,
			original:            originalResources,
			expected:            expectedResources,
			initialAnnotations:  tcYAML.InitialAnnotations,
			deleteTest:          tcYAML.DeleteTest,
			expectedAfterDelete: expectedAfterDeleteResources,
		}

		testCases = append(testCases, testCase)
	}

	return testCases, nil
}

var resourcesConfigTests []ResourcesConfigTestCase

func init() {
	var err error
	resourcesConfigTests, err = LoadResourcesConfigTestCasesFromYAML("oblik_resourcesconfig_testcases.yaml")
	if err != nil {
		panic(err)
	}
}

func TestResourcesConfigFeatures(t *testing.T) {
	flag.Parse()
	t.Logf("Starting TestResourcesConfigFeatures")

	// Setup test environment
	testClientset, vpaClientset, err := setupTestEnvironment(t, resourcesConfigTestNamespace)
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	t.Logf("Test environment setup complete")

	// Create ResourcesConfig clientset
	rcClientset, err := createResourcesConfigClientset()
	if err != nil {
		t.Fatalf("Failed to create ResourcesConfig clientset: %v", err)
	}

	// Run each test case
	for _, rtc := range resourcesConfigTests {
		rtc := rtc // capture range variable
		t.Run(rtc.name, func(t *testing.T) {
			t.Logf("Starting test: %s", colorize(rtc.name, Cyan))
			if !*noParallel {
				t.Parallel()
			}
			subCtx, cancel := context.WithTimeout(context.TODO(), 20*time.Minute)
			defer cancel()
			testResourcesConfig(subCtx, t, testClientset, vpaClientset, rcClientset, rtc)
			t.Logf("Finished test: %s", rtc.name)
		})
	}
}

func testResourcesConfig(ctx context.Context, t *testing.T, clientset *kubernetes.Clientset, vpaClientset *vpa_clientset.Clientset, rcClientset *client.ResourcesConfigClientset, rtc ResourcesConfigTestCase) {
	appName := strings.ToLower(rtc.name)
	labelSelector := map[string]string{"app": appName}

	// Create annotations map
	annotations := make(map[string]string)
	if rtc.initialAnnotations != nil {
		for k, v := range rtc.initialAnnotations {
			annotations[k] = v
		}
	}

	// Create deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        appName,
			Namespace:   resourcesConfigTestNamespace,
			Labels:      map[string]string{"app": appName},
			Annotations: annotations,
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
							Resources: rtc.original,
							Command:   []string{"tail", "-f", "/dev/null"},
						},
					},
				},
			},
		},
	}

	// Create deployment
	_, err := clientset.AppsV1().Deployments(resourcesConfigTestNamespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create Deployment: %v", err)
	}
	defer func() {
		err := clientset.AppsV1().Deployments(resourcesConfigTestNamespace).Delete(ctx, deployment.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("Failed to delete Deployment: %v", err)
		}
	}()

	// Create ResourcesConfig
	rc := createResourcesConfigObject(rtc, appName, resourcesConfigTestNamespace)
	createdRC, err := rcClientset.OblikV1().Create(ctx, resourcesConfigTestNamespace, rc, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create ResourcesConfig: %v", err)
	}

	// Wait for ResourcesConfig to be processed and annotations to be synced
	err = waitForAnnotationSync(ctx, t, clientset, appName, resourcesConfigTestNamespace)
	if err != nil {
		t.Fatalf("Failed waiting for annotation sync: %v", err)
	}

	// Wait for resources to be updated
	currentResource, err := waitForResourceUpdate(ctx, t, clientset, resourcesConfigTestNamespace, "Deployment", appName, 10*time.Minute, rtc.original)
	if err != nil {
		t.Fatalf("Failed waiting for resource update: %v", err)
	}

	// Verify resources match expected
	if isDiff(*currentResource, rtc.expected) {
		t.Log("Unexpected resources diff actual -> expected:")
		displayExpectedDiff(t, *currentResource, rtc.expected)
		t.Error("Resources update does not match expectations")
	}

	// If this is a delete test, delete the ResourcesConfig and verify resources are reset
	if rtc.deleteTest {
		t.Logf("Testing ResourcesConfig deletion for %s", rtc.name)

		// Delete ResourcesConfig
		err = rcClientset.OblikV1().Delete(ctx, resourcesConfigTestNamespace, createdRC.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("Failed to delete ResourcesConfig: %v", err)
		}

		// Wait for annotations to be removed
		err = waitForAnnotationsRemoval(ctx, t, clientset, appName, resourcesConfigTestNamespace)
		if err != nil {
			t.Fatalf("Failed waiting for annotations removal: %v", err)
		}
	}
}

// createResourcesConfigObject creates a ResourcesConfig object from the test case
func createResourcesConfigObject(rtc ResourcesConfigTestCase, name, namespace string) *oblikv1.ResourcesConfig {
	// Create a ResourcesConfig object
	rc := &oblikv1.ResourcesConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: constants.PREFIX + "v1",
			Kind:       "ResourcesConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: oblikv1.ResourcesConfigSpec{
			TargetRef: oblikv1.TargetRef{
				Kind: rtc.resourcesConfig.TargetRef["kind"],
				Name: name, // Use the test name as the target name
			},
			AnnotationMode: rtc.resourcesConfig.AnnotationMode,
		},
	}

	// Set fields from the test case
	if rtc.resourcesConfig.MinRequestCpu != "" {
		rc.Spec.MinRequestCpu = rtc.resourcesConfig.MinRequestCpu
	}
	if rtc.resourcesConfig.MinRequestMemory != "" {
		rc.Spec.MinRequestMemory = rtc.resourcesConfig.MinRequestMemory
	}
	if rtc.resourcesConfig.LimitCpuCalculatorAlgo != "" {
		rc.Spec.LimitCpuCalculatorAlgo = rtc.resourcesConfig.LimitCpuCalculatorAlgo
	}
	if rtc.resourcesConfig.LimitCpuCalculatorValue != "" {
		rc.Spec.LimitCpuCalculatorValue = rtc.resourcesConfig.LimitCpuCalculatorValue
	}
	if rtc.resourcesConfig.LimitCpuApplyMode != "" {
		rc.Spec.LimitCpuApplyMode = rtc.resourcesConfig.LimitCpuApplyMode
	}

	// Set container configs if present
	if len(rtc.resourcesConfig.ContainerConfigs) > 0 {
		rc.Spec.ContainerConfigs = make(map[string]oblikv1.ContainerConfig)
		for containerName, config := range rtc.resourcesConfig.ContainerConfigs {
			containerConfig := oblikv1.ContainerConfig{}
			if minCpu, ok := config["minRequestCpu"]; ok {
				containerConfig.MinRequestCpu = minCpu
			}
			if minMem, ok := config["minRequestMemory"]; ok {
				containerConfig.MinRequestMemory = minMem
			}
			rc.Spec.ContainerConfigs[containerName] = containerConfig
		}
	}

	return rc
}

// createResourcesConfigClientset creates a clientset for ResourcesConfig
func createResourcesConfigClientset() (*client.ResourcesConfigClientset, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	config, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Kubernetes client configuration: %v", err)
	}

	// Add ResourcesConfig types to the scheme
	client.AddToScheme()

	// Create ResourcesConfig clientset
	rcClientset, err := client.NewResourcesConfigClientset(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ResourcesConfig clientset: %v", err)
	}

	return rcClientset, nil
}

// waitForAnnotationSync waits for annotations to be synced from ResourcesConfig to the target
func waitForAnnotationSync(ctx context.Context, t *testing.T, clientset *kubernetes.Clientset, name, namespace string) error {
	t.Logf("Waiting for annotations to be synced to Deployment %s", name)
	backoff := time.Second * 2

	var lastErr error
	err := wait.PollUntil(backoff, func() (bool, error) {
		// Get the deployment
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			lastErr = fmt.Errorf("error getting deployment: %v", err)
			return false, nil
		}

		// Check if the deployment has the oblik.socialgouv.io/enabled label
		if _, ok := deployment.Labels[constants.PREFIX+"enabled"]; ok {
			t.Logf("Annotations synced to Deployment %s", name)
			return true, nil
		}

		t.Logf("Waiting for annotations to be synced to Deployment %s", name)
		return false, nil
	}, ctx.Done())

	if err != nil {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("timeout waiting for annotations to be synced: %v", err)
	}

	return nil
}

// waitForAnnotationsRemoval waits for oblik annotations to be removed from the target
func waitForAnnotationsRemoval(ctx context.Context, t *testing.T, clientset *kubernetes.Clientset, name, namespace string) error {
	t.Logf("Waiting for annotations to be removed from Deployment %s", name)
	backoff := time.Second * 2

	var lastErr error
	err := wait.PollUntil(backoff, func() (bool, error) {
		// Get the deployment
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			lastErr = fmt.Errorf("error getting deployment: %v", err)
			return false, nil
		}

		// Check if the deployment has any oblik annotations
		hasOblikAnnotations := false
		for key := range deployment.Annotations {
			if strings.HasPrefix(key, constants.PREFIX) {
				hasOblikAnnotations = true
				break
			}
		}

		if !hasOblikAnnotations {
			t.Logf("Annotations removed from Deployment %s", name)
			return true, nil
		}

		t.Logf("Waiting for annotations to be removed from Deployment %s", name)
		return false, nil
	}, ctx.Done())

	if err != nil {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("timeout waiting for annotations to be removed: %v", err)
	}

	return nil
}
