package utils

import (
	"strings"

	cnpg "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
)

func GetObjectMetadata(obj interface{}) (metav1.Object, string, string) {
	var metadata metav1.Object
	var namespace, name string

	switch v := obj.(type) {
	case *appsv1.Deployment:
		metadata = &v.ObjectMeta
	case *appsv1.StatefulSet:
		metadata = &v.ObjectMeta
	case *batchv1.CronJob:
		metadata = &v.ObjectMeta
	case *appsv1.DaemonSet:
		metadata = &v.ObjectMeta
	case *cnpg.Cluster:
		metadata = &v.ObjectMeta
	case *unstructured.Unstructured:
		metadata = v
	default:
		klog.Errorf("Unsupported object type: %T", obj)
		return nil, "", ""
	}

	if metadata != nil {
		namespace = metadata.GetNamespace()
		name = metadata.GetName()
	}

	return metadata, namespace, name
}

func GetOblikAnnotations(annotations map[string]string) map[string]string {
	oblikAnnotations := make(map[string]string)
	for k, v := range annotations {
		if strings.HasPrefix(k, "oblik.socialgouv.io/") {
			oblikAnnotations[k] = v
		}
	}
	return oblikAnnotations
}

func GetAPIVersion(obj interface{}) string {
	switch v := obj.(type) {
	case *appsv1.Deployment, *appsv1.StatefulSet, *appsv1.DaemonSet:
		return "apps/v1"
	case *batchv1.CronJob:
		return "batch/v1"
	case *cnpg.Cluster:
		return "postgresql.cnpg.io/v1"
	case *unstructured.Unstructured:
		return v.GetAPIVersion()
	default:
		return ""
	}
}

func GetKind(obj interface{}) string {
	switch v := obj.(type) {
	case *appsv1.Deployment:
		return "Deployment"
	case *appsv1.StatefulSet:
		return "StatefulSet"
	case *batchv1.CronJob:
		return "CronJob"
	case *appsv1.DaemonSet:
		return "DaemonSet"
	case *cnpg.Cluster:
		return "Cluster"
	case *unstructured.Unstructured:
		return v.GetKind()
	default:
		return ""
	}
}

func GetJSONPath(prefix, key string) string {
	if prefix == "" {
		return "/" + strings.Replace(key, "/", "~1", -1)
	}
	return prefix + "/" + strings.Replace(key, "/", "~1", -1)
}
