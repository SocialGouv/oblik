package vpa

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/constants"
	"github.com/SocialGouv/oblik/pkg/utils"
	autoscaling "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func GenerateVPAName(kind, name string) string {
	vpaName := fmt.Sprintf("%s%s-%s", config.VpaPrefix, strings.ToLower(kind), name)
	if len(vpaName) > 63 {
		hash := sha256.Sum256([]byte(vpaName))
		truncatedHash := fmt.Sprintf("%x", hash)[:8]
		vpaName = vpaName[:54] + "-" + truncatedHash
	}
	return vpaName
}

func AddVPA(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, vpaClientset *vpaclientset.Clientset, obj interface{}) {
	metadata, namespace, name := utils.GetObjectMetadata(obj)
	if metadata == nil {
		klog.Errorf("Error getting metadata for object")
		return
	}

	kind := utils.GetKind(obj)
	vpaName := GenerateVPAName(kind, name)

	// Check if VPA already exists
	_, err := vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Get(context.TODO(), vpaName, metav1.GetOptions{})
	if err == nil {
		// VPA exists, update it
		UpdateVPA(clientset, dynamicClient, vpaClientset, obj)
		return
	} else if !strings.Contains(err.Error(), "not found") {
		klog.Errorf("Error checking VPA existence for %s/%s: %v", namespace, name, err)
		return
	}

	// VPA doesn't exist, create it
	annotations := utils.GetOblikAnnotations(metadata.GetAnnotations())
	updateMode := vpa.UpdateModeOff
	vpa := &vpa.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:        vpaName,
			Namespace:   namespace,
			Annotations: annotations,
			Labels: map[string]string{
				constants.PREFIX + "enabled": "true",
			},
		},
		Spec: vpa.VerticalPodAutoscalerSpec{
			TargetRef: &autoscaling.CrossVersionObjectReference{
				APIVersion: utils.GetAPIVersion(obj),
				Kind:       kind,
				Name:       name,
			},
			UpdatePolicy: &vpa.PodUpdatePolicy{
				UpdateMode: &updateMode,
			},
		},
	}

	_, err = vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Create(context.TODO(), vpa, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "unable to create new content in namespace") && strings.Contains(err.Error(), "because it is being terminated") {
			klog.Infof("Skipping VPA creation for %s/%s in namespace %s: namespace is being terminated", kind, name, namespace)
			return
		}
		klog.Errorf("Error creating VPA for %s/%s: %v", namespace, name, err)
	} else {
		klog.Infof("Created VPA %s for %s/%s", vpaName, namespace, name)
	}
}

func UpdateVPA(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, vpaClientset *vpaclientset.Clientset, obj interface{}) {
	metadata, namespace, name := utils.GetObjectMetadata(obj)
	if metadata == nil {
		klog.Errorf("Error getting metadata for object")
		return
	}

	annotations := utils.GetOblikAnnotations(metadata.GetAnnotations())
	kind := utils.GetKind(obj)
	vpaName := GenerateVPAName(kind, name)

	vpa, err := vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Get(context.TODO(), vpaName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return
		}
		klog.Errorf("Error getting VPA for %s/%s: %v", namespace, name, err)
		return
	}

	vpa.ObjectMeta.Annotations = annotations
	vpa.ObjectMeta.Labels[constants.PREFIX+"enabled"] = "true"

	_, err = vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Update(context.TODO(), vpa, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Error updating VPA for %s/%s: %v", namespace, name, err)
	} else {
		klog.Infof("Updated VPA %s for %s/%s", vpaName, namespace, name)
	}
}

func DeleteVPA(vpaClientset *vpaclientset.Clientset, obj interface{}) {
	metadata, namespace, name := utils.GetObjectMetadata(obj)
	if metadata == nil {
		klog.Errorf("Error getting metadata for object")
		return
	}

	kind := utils.GetKind(obj)
	vpaName := GenerateVPAName(kind, name)

	err := vpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Delete(context.TODO(), vpaName, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return
		}
		klog.Errorf("Error deleting VPA for %s/%s: %v", namespace, name, err)
	} else {
		klog.Infof("Deleted VPA %s for %s/%s", vpaName, namespace, name)
	}
}
