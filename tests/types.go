package main

import corev1 "k8s.io/api/core/v1"

type OblikTestCase struct {
	name           string
	original       corev1.ResourceRequirements
	expected       corev1.ResourceRequirements
	annotations    map[string]string
	shouldntUpdate bool
}
