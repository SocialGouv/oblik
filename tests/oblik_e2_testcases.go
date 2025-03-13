package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type ResourceRequirementsYAML struct {
	Requests map[string]string `yaml:"requests"`
	Limits   map[string]string `yaml:"limits"`
}

// OblikTestCaseYAML represents each test case in YAML
type OblikTestCaseYAML struct {
	Name               string                   `yaml:"name"`
	Annotations        map[string]string        `yaml:"annotations"`
	Original           ResourceRequirementsYAML `yaml:"original"`
	Expected           ResourceRequirementsYAML `yaml:"expected"`
	ShouldntUpdate     bool                     `yaml:"shouldntUpdate"`
	UpdateVPA          bool                     `yaml:"updateVPA"`
	VPARecommendations struct {
		CPU    string `yaml:"cpu"`
		Memory string `yaml:"memory"`
	} `yaml:"vpaRecommendations"`
}

// TestCasesYAML is the top-level structure for the YAML file
type TestCasesYAML struct {
	TestCases []OblikTestCaseYAML `yaml:"test_cases"`
}

// LoadTestCasesFromYAML loads test cases from a YAML file
func LoadTestCasesFromYAML(filename string) ([]OblikTestCase, error) {
	// Read the YAML file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal the YAML into TestCasesYAML struct
	var testCasesYAML TestCasesYAML
	err = yaml.Unmarshal(data, &testCasesYAML)
	if err != nil {
		return nil, err
	}

	// Convert TestCasesYAML to []OblikTestCase
	var testCases []OblikTestCase
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

		testCase := OblikTestCase{
			name:           tcYAML.Name,
			annotations:    tcYAML.Annotations,
			original:       originalResources,
			expected:       expectedResources,
			shouldntUpdate: tcYAML.ShouldntUpdate,
			updateVPA:      tcYAML.UpdateVPA,
		}

		if tcYAML.UpdateVPA {
			testCase.vpaRecommendations = &VPARecommendations{
				CPU:    tcYAML.VPARecommendations.CPU,
				Memory: tcYAML.VPARecommendations.Memory,
			}
		}

		testCases = append(testCases, testCase)
	}

	return testCases, nil
}

// Helper function to parse ResourceRequirementsYAML into corev1.ResourceRequirements
func parseResourceRequirements(rrYAML ResourceRequirementsYAML) (corev1.ResourceRequirements, error) {
	rr := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	// Parse Requests
	for k, v := range rrYAML.Requests {
		quantity, err := resource.ParseQuantity(v)
		if err != nil {
			return rr, fmt.Errorf("invalid quantity for request %s: %v", k, err)
		}
		rr.Requests[corev1.ResourceName(k)] = quantity
	}

	// Parse Limits
	for k, v := range rrYAML.Limits {
		quantity, err := resource.ParseQuantity(v)
		if err != nil {
			return rr, fmt.Errorf("invalid quantity for limit %s: %v", k, err)
		}
		rr.Limits[corev1.ResourceName(k)] = quantity
	}

	return rr, nil
}

var e2eOblikTests []OblikTestCase

func init() {
	var err error
	e2eOblikTests, err = LoadTestCasesFromYAML("oblik_e2e_testcases.yaml")
	if err != nil {
		panic(err)
	}
}
