package resourcesconfig

import (
	"context"
	"time"

	oblikv1 "github.com/SocialGouv/oblik/pkg/apis/oblik/v1"
	"github.com/SocialGouv/oblik/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// UpdateStatus updates the status of the ResourcesConfig
func UpdateStatus(ctx context.Context, kubeClients *client.KubeClients, rc *oblikv1.ResourcesConfig, success bool, message string) {
	// Create a copy of the ResourcesConfig
	rcCopy := rc.DeepCopy()

	// Update status fields
	now := metav1.NewTime(time.Now())
	rcCopy.Status.LastUpdateTime = now
	rcCopy.Status.ObservedGeneration = rc.Generation

	if success {
		rcCopy.Status.LastSyncTime = now

		// Update conditions
		setCondition(rcCopy, "Synced", metav1.ConditionTrue, "SyncSucceeded", "Successfully synced annotations to target")
	} else {
		// Update conditions
		setCondition(rcCopy, "Synced", metav1.ConditionFalse, "SyncFailed", message)
	}

	// Update the ResourcesConfig status
	_, err := kubeClients.ResourcesConfigClientset.OblikV1().UpdateStatus(ctx, rcCopy.Namespace, rcCopy, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Error updating ResourcesConfig status: %s", err.Error())
	}
}

// setCondition sets a condition on the ResourcesConfig
func setCondition(rc *oblikv1.ResourcesConfig, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.NewTime(time.Now())

	// Find existing condition
	for i, condition := range rc.Status.Conditions {
		if condition.Type == conditionType {
			// Update existing condition
			if condition.Status != status {
				condition.LastTransitionTime = now
			}
			condition.Status = status
			condition.Reason = reason
			condition.Message = message
			condition.ObservedGeneration = rc.Generation

			rc.Status.Conditions[i] = condition
			return
		}
	}

	// Create new condition
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: rc.Generation,
	}

	rc.Status.Conditions = append(rc.Status.Conditions, condition)
}
