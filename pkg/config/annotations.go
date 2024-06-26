package config

import (
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func getAnnotationFromMap(name string, annotations map[string]string) string {
	return annotations["oblik.socialgouv.io/"+name]
}

func getVpaAnnotations(vpaResource *vpa.VerticalPodAutoscaler) map[string]string {
	annotations := vpaResource.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	return annotations
}
