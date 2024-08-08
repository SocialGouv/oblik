package config

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func CreateConfigurable(object interface{}) *Configurable {
	return &Configurable{
		Object: object,
	}
}

type Configurable struct {
	Object interface{}
}

func (co *Configurable) GetAnnotations() map[string]string {
	switch obj := co.Object.(type) {
	case metav1.Object:
		return obj.GetAnnotations()
	default:
		return map[string]string{}
	}
}

func (co *Configurable) GetNamespace() string {
	switch obj := co.Object.(type) {
	case metav1.Object:
		return obj.GetNamespace()
	default:
		return ""
	}
}

func (co *Configurable) GetName() string {
	switch obj := co.Object.(type) {
	case metav1.Object:
		return obj.GetName()
	default:
		return ""
	}
}

func (co *Configurable) GetContainerNames() []string {
	switch obj := co.Object.(type) {
	case *appsv1.Deployment:
		return getDeploymentContainerNames(obj)
	case *appsv1.StatefulSet:
		return getStatefulSetContainerNames(obj)
	case *batchv1.CronJob:
		return getCronJobContainerNames(obj)
	case *appsv1.DaemonSet:
		return getDaemonSetContainerNames(obj)
	case *vpa.VerticalPodAutoscaler:
		return getVPAContainerNames(obj)
	// Add other cases as needed
	default:
		return []string{}
	}
}

func getDeploymentContainerNames(deployment *appsv1.Deployment) []string {
	containerNames := []string{}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		containerNames = append(containerNames, container.Name)
	}
	return containerNames
}

func getStatefulSetContainerNames(statefulSet *appsv1.StatefulSet) []string {
	containerNames := []string{}
	for _, container := range statefulSet.Spec.Template.Spec.Containers {
		containerNames = append(containerNames, container.Name)
	}
	return containerNames
}

func getCronJobContainerNames(cronJob *batchv1.CronJob) []string {
	containerNames := []string{}
	for _, container := range cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers {
		containerNames = append(containerNames, container.Name)
	}
	return containerNames
}

func getDaemonSetContainerNames(daemonSet *appsv1.DaemonSet) []string {
	containerNames := []string{}
	for _, container := range daemonSet.Spec.Template.Spec.Containers {
		containerNames = append(containerNames, container.Name)
	}
	return containerNames
}

func getVPAContainerNames(vpaResource *vpa.VerticalPodAutoscaler) []string {
	containerNames := []string{}
	if vpaResource.Status.Recommendation != nil {
		for _, containerRecommendation := range vpaResource.Status.Recommendation.ContainerRecommendations {
			containerNames = append(containerNames, containerRecommendation.ContainerName)
		}
	}
	return containerNames
}
