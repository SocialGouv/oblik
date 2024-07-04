package target

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/logical"
	"github.com/SocialGouv/oblik/pkg/reporting"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	jsonpatch "github.com/evanphx/json-patch/v5"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/dynamic"
)

func UpdateCluster(dynamicClient *dynamic.DynamicClient, vpa *vpa.VerticalPodAutoscaler, vcfg *config.VpaWorkloadCfg) (*reporting.UpdateResult, error) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	clusterName := targetRef.Name

	gvr := schema.GroupVersionResource{
		Group:    "postgresql.cnpg.io",
		Version:  "v1",
		Resource: "clusters",
	}

	clusterResource, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error fetching cluster: %s", err.Error())
	}

	originalClusterJSON, err := json.Marshal(clusterResource)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling original cluster: %s", err.Error())
	}

	var cluster cnpgv1.Cluster
	err = json.Unmarshal(originalClusterJSON, &cluster)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling cluster: %s", err.Error())
	}

	containers := []corev1.Container{
		corev1.Container{
			Name:      "postgres",
			Resources: cluster.Spec.Resources,
		},
	}
	update := logical.UpdateContainerResources(containers, vpa, vcfg)
	cluster.Spec.Resources = containers[0].Resources

	updatedClusterJSON, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling updated cluster: %s", err.Error())
	}

	patchBytes, err := jsonpatch.CreateMergePatch(originalClusterJSON, updatedClusterJSON)
	if err != nil {
		return nil, fmt.Errorf("Error creating patch: %s", err.Error())
	}

	if !vcfg.GetDryRun() {
		_, err = dynamicClient.Resource(gvr).Namespace(namespace).Patch(context.TODO(), clusterName, types.MergePatchType, patchBytes, metav1.PatchOptions{
			FieldManager: FieldManager,
		})
		if err != nil {
			update.Type = reporting.ResultTypeFailed
			update.Error = err
			return nil, fmt.Errorf("Error applying patch to cluster: %s", err.Error())
		}
		update.Type = reporting.ResultTypeSuccess
	} else {
		update.Type = reporting.ResultTypeDryRun
	}
	return update, nil
}
