package target

import (
	"fmt"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	ovpa "github.com/SocialGouv/oblik/pkg/vpa"
	"k8s.io/apimachinery/pkg/api/errors"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

func ApplyVPARecommendations(kubeClients *client.KubeClients, vpa *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) error {
	clientset := kubeClients.Clientset
	dynamicClient := kubeClients.DynamicClient
	vpaClientset := kubeClients.VpaClientset

	targetRef := vpa.Spec.TargetRef
	var update *reporting.UpdateResult
	var err error
	switch targetRef.Kind {
	case "Deployment":
		update, err = UpdateDeployment(clientset, vpa, scfg)
	case "StatefulSet":
		update, err = UpdateStatefulSet(clientset, vpa, scfg)
	case "DaemonSet":
		update, err = UpdateDaemonSet(clientset, vpa, scfg)
	case "CronJob":
		update, err = UpdateCronJob(clientset, vpa, scfg)
	case "Cluster":
		if targetRef.APIVersion == "postgresql.cnpg.io/v1" {
			update, err = UpdateCluster(dynamicClient, vpa, scfg)
		} else {
			err := fmt.Errorf("Unsupported Cluster kind from apiVersion: %s", targetRef.APIVersion)
			klog.Warning(err)
			return err
		}
	default:
		err := fmt.Errorf("Unsupported apiVersion/kind: %s/%s", targetRef.APIVersion, targetRef.Kind)
		klog.Warning(err)
		return err
	}
	if err != nil {
		if errors.IsNotFound(err) {
			ovpa.DeleteVPA(vpaClientset, vpa)
			return nil
		}
		klog.Errorf("Failed to apply updates for %s: %s", scfg.Key, err.Error())
	}
	reporting.ReportUpdated(update, scfg)
	return err
}
