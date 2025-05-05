package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourcesConfig is the Schema for the resourcesconfigs API
type ResourcesConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourcesConfigSpec   `json:"spec,omitempty"`
	Status ResourcesConfigStatus `json:"status,omitempty"`
}

// ResourceList represents CPU and memory resource specifications
type ResourceList struct {
	// CPU resource value
	CPU string `json:"cpu,omitempty"`

	// Memory resource value
	Memory string `json:"memory,omitempty"`
}

// ResourcesConfigSpec defines the desired state of ResourcesConfig
type ResourcesConfigSpec struct {
	// TargetRef points to the controller managing the set of pods
	TargetRef TargetRef `json:"targetRef"`

	// AnnotationMode controls how annotations are managed
	// "replace" (default): Replace all oblik annotations on the target
	// "merge": Merge with existing annotations, with ResourcesConfig taking precedence
	AnnotationMode string `json:"annotationMode,omitempty"`

	// Cron expression to schedule when the recommendations are applied
	Cron string `json:"cron,omitempty"`

	// Maximum random delay added to the cron schedule
	CronAddRandomMax string `json:"cronAddRandomMax,omitempty"`

	// If true, Oblik will simulate the updates without applying them
	DryRun bool `json:"dryRun,omitempty"`

	// Enable mutating webhook resources enforcement
	WebhookEnabled bool `json:"webhookEnabled,omitempty"`

	// CPU request recommendation mode: "enforce" or "off"
	RequestCpuApplyMode string `json:"requestCpuApplyMode,omitempty"`

	// Memory request recommendation mode: "enforce" or "off"
	RequestMemoryApplyMode string `json:"requestMemoryApplyMode,omitempty"`

	// CPU limit apply mode: "enforce" or "off"
	LimitCpuApplyMode string `json:"limitCpuApplyMode,omitempty"`

	// Memory limit apply mode: "enforce" or "off"
	LimitMemoryApplyMode string `json:"limitMemoryApplyMode,omitempty"`

	// CPU limit calculator algorithm: "ratio" or "margin"
	LimitCpuCalculatorAlgo string `json:"limitCpuCalculatorAlgo,omitempty"`

	// Memory limit calculator algorithm: "ratio" or "margin"
	LimitMemoryCalculatorAlgo string `json:"limitMemoryCalculatorAlgo,omitempty"`

	// Value used by the CPU limit calculator algorithm
	LimitCpuCalculatorValue string `json:"limitCpuCalculatorValue,omitempty"`

	// Value used by the memory limit calculator algorithm
	LimitMemoryCalculatorValue string `json:"limitMemoryCalculatorValue,omitempty"`

	// Default CPU request if not provided by the VPA: "off", "minAllowed", "maxAllowed", or value
	UnprovidedApplyDefaultRequestCpu string `json:"unprovidedApplyDefaultRequestCpu,omitempty"`

	// Default memory request if not provided by the VPA: "off", "minAllowed", "maxAllowed", or value
	UnprovidedApplyDefaultRequestMemory string `json:"unprovidedApplyDefaultRequestMemory,omitempty"`

	// Algorithm to increase CPU request: "ratio" or "margin"
	IncreaseRequestCpuAlgo string `json:"increaseRequestCpuAlgo,omitempty"`

	// Value used to increase CPU request
	IncreaseRequestCpuValue string `json:"increaseRequestCpuValue,omitempty"`

	// Algorithm to increase memory request: "ratio" or "margin"
	IncreaseRequestMemoryAlgo string `json:"increaseRequestMemoryAlgo,omitempty"`

	// Value used to increase memory request
	IncreaseRequestMemoryValue string `json:"increaseRequestMemoryValue,omitempty"`

	// Minimum CPU limit value
	MinLimitCpu string `json:"minLimitCpu,omitempty"`

	// Maximum CPU limit value
	MaxLimitCpu string `json:"maxLimitCpu,omitempty"`

	// Minimum memory limit value
	MinLimitMemory string `json:"minLimitMemory,omitempty"`

	// Maximum memory limit value
	MaxLimitMemory string `json:"maxLimitMemory,omitempty"`

	// Minimum CPU request value
	MinRequestCpu string `json:"minRequestCpu,omitempty"`

	// Maximum CPU request value
	MaxRequestCpu string `json:"maxRequestCpu,omitempty"`

	// Minimum memory request value
	MinRequestMemory string `json:"minRequestMemory,omitempty"`

	// Maximum memory request value
	MaxRequestMemory string `json:"maxRequestMemory,omitempty"`

	// Minimum allowed CPU recommendation value
	MinAllowedRecommendationCpu string `json:"minAllowedRecommendationCpu,omitempty"`

	// Maximum allowed CPU recommendation value
	MaxAllowedRecommendationCpu string `json:"maxAllowedRecommendationCpu,omitempty"`

	// Minimum allowed memory recommendation value
	MinAllowedRecommendationMemory string `json:"minAllowedRecommendationMemory,omitempty"`

	// Maximum allowed memory recommendation value
	MaxAllowedRecommendationMemory string `json:"maxAllowedRecommendationMemory,omitempty"`

	// Algorithm to calculate minimum CPU request difference: "ratio" or "margin"
	MinDiffCpuRequestAlgo string `json:"minDiffCpuRequestAlgo,omitempty"`

	// Value used for minimum CPU request difference calculation
	MinDiffCpuRequestValue string `json:"minDiffCpuRequestValue,omitempty"`

	// Algorithm to calculate minimum memory request difference: "ratio" or "margin"
	MinDiffMemoryRequestAlgo string `json:"minDiffMemoryRequestAlgo,omitempty"`

	// Value used for minimum memory request difference calculation
	MinDiffMemoryRequestValue string `json:"minDiffMemoryRequestValue,omitempty"`

	// Algorithm to calculate minimum CPU limit difference: "ratio" or "margin"
	MinDiffCpuLimitAlgo string `json:"minDiffCpuLimitAlgo,omitempty"`

	// Value used for minimum CPU limit difference calculation
	MinDiffCpuLimitValue string `json:"minDiffCpuLimitValue,omitempty"`

	// Algorithm to calculate minimum memory limit difference: "ratio" or "margin"
	MinDiffMemoryLimitAlgo string `json:"minDiffMemoryLimitAlgo,omitempty"`

	// Value used for minimum memory limit difference calculation
	MinDiffMemoryLimitValue string `json:"minDiffMemoryLimitValue,omitempty"`

	// Calculate memory request from CPU request instead of recommendation
	MemoryRequestFromCpuEnabled bool `json:"memoryRequestFromCpuEnabled,omitempty"`

	// Calculate memory limit from CPU limit instead of recommendation
	MemoryLimitFromCpuEnabled bool `json:"memoryLimitFromCpuEnabled,omitempty"`

	// Algorithm to calculate memory request based on CPU request: "ratio" or "margin"
	MemoryRequestFromCpuAlgo string `json:"memoryRequestFromCpuAlgo,omitempty"`

	// Value used for calculating memory request from CPU request
	MemoryRequestFromCpuValue string `json:"memoryRequestFromCpuValue,omitempty"`

	// Algorithm to calculate memory limit based on CPU limit: "ratio" or "margin"
	MemoryLimitFromCpuAlgo string `json:"memoryLimitFromCpuAlgo,omitempty"`

	// Value used for calculating memory limit from CPU limit
	MemoryLimitFromCpuValue string `json:"memoryLimitFromCpuValue,omitempty"`

	// Select which recommendation to apply by default on request: "frugal", "balanced", "peak"
	RequestApplyTarget string `json:"requestApplyTarget,omitempty"`

	// Select which recommendation to apply for CPU request: "frugal", "balanced", "peak"
	RequestCpuApplyTarget string `json:"requestCpuApplyTarget,omitempty"`

	// Select which recommendation to apply for memory request: "frugal", "balanced", "peak"
	RequestMemoryApplyTarget string `json:"requestMemoryApplyTarget,omitempty"`

	// Select which recommendation to apply by default on limit: "auto", "frugal", "balanced", "peak"
	LimitApplyTarget string `json:"limitApplyTarget,omitempty"`

	// Select which recommendation to apply for CPU limit: "auto", "frugal", "balanced", "peak"
	LimitCpuApplyTarget string `json:"limitCpuApplyTarget,omitempty"`

	// Select which recommendation to apply for memory limit: "auto", "frugal", "balanced", "peak"
	LimitMemoryApplyTarget string `json:"limitMemoryApplyTarget,omitempty"`

	// Allowed scaling direction for CPU request: "both", "up", "down"
	RequestCpuScaleDirection string `json:"requestCpuScaleDirection,omitempty"`

	// Allowed scaling direction for memory request: "both", "up", "down"
	RequestMemoryScaleDirection string `json:"requestMemoryScaleDirection,omitempty"`

	// Allowed scaling direction for CPU limit: "both", "up", "down"
	LimitCpuScaleDirection string `json:"limitCpuScaleDirection,omitempty"`

	// Allowed scaling direction for memory limit: "both", "up", "down"
	LimitMemoryScaleDirection string `json:"limitMemoryScaleDirection,omitempty"`

	// Direct resource specifications (flat style)
	RequestCpu    string `json:"requestCpu,omitempty"`
	RequestMemory string `json:"requestMemory,omitempty"`
	LimitCpu      string `json:"limitCpu,omitempty"`
	LimitMemory   string `json:"limitMemory,omitempty"`
	
	// Kubernetes-native style resource specifications (nested)
	Request *ResourceList `json:"request,omitempty"`
	Limit   *ResourceList `json:"limit,omitempty"`

	// Container specific configurations
	ContainerConfigs map[string]ContainerConfig `json:"containerConfigs,omitempty"`
}

// TargetRef points to the controller managing the set of pods
type TargetRef struct {
	// API version of the referent
	APIVersion string `json:"apiVersion,omitempty"`

	// Kind of the referent
	Kind string `json:"kind"`

	// Name of the referent
	Name string `json:"name"`
}

// ContainerConfig defines container-specific configurations
type ContainerConfig struct {
	// Direct resource specifications (flat style)
	RequestCpu    string `json:"requestCpu,omitempty"`
	RequestMemory string `json:"requestMemory,omitempty"`
	LimitCpu      string `json:"limitCpu,omitempty"`
	LimitMemory   string `json:"limitMemory,omitempty"`
	
	// Kubernetes-native style resource specifications (nested)
	Request *ResourceList `json:"request,omitempty"`
	Limit   *ResourceList `json:"limit,omitempty"`
	// Minimum CPU limit value
	MinLimitCpu string `json:"minLimitCpu,omitempty"`

	// Maximum CPU limit value
	MaxLimitCpu string `json:"maxLimitCpu,omitempty"`

	// Minimum memory limit value
	MinLimitMemory string `json:"minLimitMemory,omitempty"`

	// Maximum memory limit value
	MaxLimitMemory string `json:"maxLimitMemory,omitempty"`

	// Minimum CPU request value
	MinRequestCpu string `json:"minRequestCpu,omitempty"`

	// Maximum CPU request value
	MaxRequestCpu string `json:"maxRequestCpu,omitempty"`

	// Minimum memory request value
	MinRequestMemory string `json:"minRequestMemory,omitempty"`

	// Maximum memory request value
	MaxRequestMemory string `json:"maxRequestMemory,omitempty"`

	// Minimum allowed CPU recommendation value
	MinAllowedRecommendationCpu string `json:"minAllowedRecommendationCpu,omitempty"`

	// Maximum allowed CPU recommendation value
	MaxAllowedRecommendationCpu string `json:"maxAllowedRecommendationCpu,omitempty"`

	// Minimum allowed memory recommendation value
	MinAllowedRecommendationMemory string `json:"minAllowedRecommendationMemory,omitempty"`

	// Maximum allowed memory recommendation value
	MaxAllowedRecommendationMemory string `json:"maxAllowedRecommendationMemory,omitempty"`
}

// ResourcesConfigStatus defines the observed state of ResourcesConfig
type ResourcesConfigStatus struct {
	// ObservedGeneration is the most recent generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastUpdateTime is the last time the object was updated
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastSyncTime is the last time the object was successfully synced with the target resource
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourcesConfigList contains a list of ResourcesConfig
type ResourcesConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourcesConfig `json:"items"`
}
