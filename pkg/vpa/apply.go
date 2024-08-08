package vpa

import (
	"fmt"

	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	"github.com/SocialGouv/oblik/pkg/target"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func ApplyVPARecommendations(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, vpa *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) error {
	targetRef := vpa.Spec.TargetRef
	var update *reporting.UpdateResult
	var err error
	switch targetRef.Kind {
	case "Deployment":
		update, err = target.UpdateDeployment(clientset, vpa, scfg)
	case "StatefulSet":
		update, err = target.UpdateStatefulSet(clientset, vpa, scfg)
	case "DaemonSet":
		update, err = target.UpdateDaemonSet(clientset, vpa, scfg)
	case "CronJob":
		update, err = target.UpdateCronJob(clientset, vpa, scfg)
	case "Cluster":
		if targetRef.APIVersion == "postgresql.cnpg.io/v1" {
			update, err = target.UpdateCluster(dynamicClient, vpa, scfg)
		} else {
			err := fmt.Errorf("Unsupported Cluster kind from apiVersion: %s", targetRef.APIVersion)
			klog.Warning(err)
			return err
		}
	}
	if err != nil {
		klog.Errorf("Failed to apply updates for %s: %s", scfg.Key, err.Error())
	}
	reporting.ReportUpdated(update, scfg)
	return err
}
