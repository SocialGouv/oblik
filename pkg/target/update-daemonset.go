package target

import (
	"context"
	"fmt"

	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/kubernetes"
)

func UpdateDaemonSet(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *config.VpaWorkloadCfg) (*reporting.UpdateResult, error) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	daemonsetName := targetRef.Name
	daemonset, err := clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), daemonsetName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error fetching daemonset: %s", err.Error())
	}

	update := updateContainerResources(daemonset.Spec.Template.Spec.Containers, vpa, vcfg)

	patchData, err := createPatch(daemonset, "apps/v1", "DaemonSet")
	if err != nil {
		return nil, fmt.Errorf("Error creating patch: %s", err.Error())
	}

	if !vcfg.GetDryRun() {
		force := true
		_, err = clientset.AppsV1().DaemonSets(namespace).Patch(context.TODO(), daemonsetName, types.ApplyPatchType, patchData, metav1.PatchOptions{
			FieldManager: FieldManager,
			Force:        &force, // Force the apply to take ownership of the fields
		})
		if err != nil {
			update.Type = reporting.ResultTypeFailed
			update.Error = err
			return nil, fmt.Errorf("Error applying patch to daemonset: %s", err.Error())
		}
		update.Type = reporting.ResultTypeSuccess
	} else {
		update.Type = reporting.ResultTypeDryRun
	}
	return update, nil
}
