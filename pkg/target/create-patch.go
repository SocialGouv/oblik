package target

import (
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
)

func createPatch(obj interface{}, apiVersion, kind string) ([]byte, error) {
	var patchedObj interface{}
	switch t := obj.(type) {
	case *appsv1.Deployment:
		patchedObj = t.DeepCopy()
		patchedObj.(*appsv1.Deployment).APIVersion = apiVersion
		patchedObj.(*appsv1.Deployment).Kind = kind
		patchedObj.(*appsv1.Deployment).ObjectMeta.ManagedFields = nil
	case *appsv1.StatefulSet:
		patchedObj = t.DeepCopy()
		patchedObj.(*appsv1.StatefulSet).APIVersion = apiVersion
		patchedObj.(*appsv1.StatefulSet).Kind = kind
		patchedObj.(*appsv1.StatefulSet).ObjectMeta.ManagedFields = nil
	case *appsv1.DaemonSet:
		patchedObj = t.DeepCopy()
		patchedObj.(*appsv1.DaemonSet).APIVersion = apiVersion
		patchedObj.(*appsv1.DaemonSet).Kind = kind
		patchedObj.(*appsv1.DaemonSet).ObjectMeta.ManagedFields = nil
	case *batchv1.CronJob:
		patchedObj = t.DeepCopy()
		patchedObj.(*batchv1.CronJob).APIVersion = apiVersion
		patchedObj.(*batchv1.CronJob).Kind = kind
		patchedObj.(*batchv1.CronJob).ObjectMeta.ManagedFields = nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	jsonData, err := json.Marshal(patchedObj)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}
