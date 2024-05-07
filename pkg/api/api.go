package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OblikPodAutoscaler is the Schema for the oblikpodautoscalers API
type OblikPodAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OblikPodAutoscalerSpec   `json:"spec,omitempty"`
	Status OblikPodAutoscalerStatus `json:"status,omitempty"`
}

// OblikPodAutoscalerSpec defines the desired state of OblikPodAutoscaler
type OblikPodAutoscalerSpec struct {
	CpuCursor    string    `json:"cpuCursor,omitempty"`
	MemoryCursor string    `json:"memoryCursor,omitempty"`
	TargetRef    TargetRef `json:"targetRef,omitempty"`
}

type TargetRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}

// OblikPodAutoscalerStatus defines the observed state of OblikPodAutoscaler
type OblikPodAutoscalerStatus struct {
	// Fields here reflect the status of the operator's actions
}

// +kubebuilder:object:root=true

// OblikPodAutoscalerList contains a list of OblikPodAutoscaler
type OblikPodAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OblikPodAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OblikPodAutoscaler{}, &OblikPodAutoscalerList{})
}
