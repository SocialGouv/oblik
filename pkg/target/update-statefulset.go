package target

import (
	"context"
	"fmt"

	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/logical"
	"github.com/SocialGouv/oblik/pkg/reporting"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/kubernetes"
)

func UpdateStatefulSet(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *config.VpaWorkloadCfg) (*reporting.UpdateResult, error) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	statefulSetName := targetRef.Name

	statefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error fetching stateful set: %s", err.Error())
	}

	update := logical.UpdateContainerResources(statefulSet.Spec.Template.Spec.Containers, vpa, vcfg)

	patchData, err := createPatch(statefulSet, "apps/v1", "StatefulSet")
	if err != nil {
		return nil, fmt.Errorf("Error creating patch: %s", err.Error())
	}

	if !vcfg.GetDryRun() {
		force := true
		_, err = clientset.AppsV1().StatefulSets(namespace).Patch(context.TODO(), statefulSetName, types.ApplyPatchType, patchData, metav1.PatchOptions{
			FieldManager: FieldManager,
			Force:        &force, // Force the apply to take ownership of the fields
		})
		if err != nil {
			update.Type = reporting.ResultTypeFailed
			update.Error = err
			return nil, fmt.Errorf("Error applying patch to statefulset: %s", err.Error())
		}
		update.Type = reporting.ResultTypeSuccess
	} else {
		update.Type = reporting.ResultTypeDryRun
	}
	return update, nil
}
