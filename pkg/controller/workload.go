package controller

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Workload interface {
	GetContainers() []corev1.Container
	SetContainers(containers []corev1.Container)
	GetObjectMeta() *metav1.ObjectMeta
	GetSpec() interface{}
}

type DeploymentWrapper struct {
	Deployment *appsv1.Deployment
}

func (w *DeploymentWrapper) GetContainers() []corev1.Container {
	return w.Deployment.Spec.Template.Spec.Containers
}

func (w *DeploymentWrapper) SetContainers(containers []corev1.Container) {
	w.Deployment.Spec.Template.Spec.Containers = containers
}

func (w *DeploymentWrapper) GetObjectMeta() *metav1.ObjectMeta {
	return &w.Deployment.ObjectMeta
}

func (w *DeploymentWrapper) GetSpec() interface{} {
	return w.Deployment.Spec
}

type StatefulSetWrapper struct {
	StatefulSet *appsv1.StatefulSet
}

func (w *StatefulSetWrapper) GetContainers() []corev1.Container {
	return w.StatefulSet.Spec.Template.Spec.Containers
}

func (w *StatefulSetWrapper) SetContainers(containers []corev1.Container) {
	w.StatefulSet.Spec.Template.Spec.Containers = containers
}

func (w *StatefulSetWrapper) GetObjectMeta() *metav1.ObjectMeta {
	return &w.StatefulSet.ObjectMeta
}

func (w *StatefulSetWrapper) GetSpec() interface{} {
	return w.StatefulSet.Spec
}

type CronJobWrapper struct {
	CronJob *batchv1.CronJob
}

func (w *CronJobWrapper) GetContainers() []corev1.Container {
	return w.CronJob.Spec.JobTemplate.Spec.Template.Spec.Containers
}

func (w *CronJobWrapper) SetContainers(containers []corev1.Container) {
	w.CronJob.Spec.JobTemplate.Spec.Template.Spec.Containers = containers
}

func (w *CronJobWrapper) GetObjectMeta() *metav1.ObjectMeta {
	return &w.CronJob.ObjectMeta
}

func (w *CronJobWrapper) GetSpec() interface{} {
	return w.CronJob.Spec
}
