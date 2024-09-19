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

func UpdateDeployment(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) (*reporting.UpdateResult, error) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	deploymentName := targetRef.Name
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error fetching deployment: %s", err.Error())
	}

	update := logical.UpdateContainerResources(deployment.Spec.Template.Spec.Containers, vpa, scfg)

	patchData, err := createPatch(deployment, "apps/v1", "Deployment")
	if err != nil {
		return nil, fmt.Errorf("Error creating patch: %s", err.Error())
	}

	if !scfg.GetDryRun() {
		force := true
		_, err = clientset.AppsV1().Deployments(namespace).Patch(context.TODO(), deploymentName, types.ApplyPatchType, patchData, metav1.PatchOptions{
			FieldManager: FieldManager,
			Force:        &force, // Force the apply to take ownership of the fields
		})
		if err != nil {
			update.Type = reporting.ResultTypeFailed
			update.Error = err
			return nil, fmt.Errorf("Error applying patch to deployment: %s", err.Error())
		}
		update.Type = reporting.ResultTypeSuccess
	} else {
		update.Type = reporting.ResultTypeDryRun
	}
	return update, nil
}
