package controller

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

const defaultCron = "0 2 * * *"
const defaultCronAddRandomMax = "120m"

type ApplyMode int

const (
	ApplyModeEnforce ApplyMode = iota
	ApplyModeOff
)

type CalculatorAlgo int

const (
	CalculatorAlgoRatio CalculatorAlgo = iota
	CalculatorAlgoMargin
)

type UnprovidedApplyDefaultMode int

const (
	UnprovidedApplyDefaultModeOff UnprovidedApplyDefaultMode = iota
	UnprovidedApplyDefaultModeMinAllowed
	UnprovidedApplyDefaultModeMaxAllowed
	UnprovidedApplyDefaultModeValue
)

type VpaContainerCfg struct {
	Key           string
	ContainerName string
	*LoadCfg
}
type VpaWorkloadCfg struct {
	Key string
	*LoadCfg
	Containers map[string]*VpaContainerCfg
}

func (v *VpaWorkloadCfg) GetRequestCPUApplyMode(containerName string) ApplyMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].RequestCPUApplyMode != nil {
		return *v.Containers[containerName].RequestCPUApplyMode
	}
	if v.RequestCPUApplyMode != nil {
		return *v.RequestCPUApplyMode
	}
	return ApplyModeEnforce
}

func (v *VpaWorkloadCfg) GetRequestMemoryApplyMode(containerName string) ApplyMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].RequestMemoryApplyMode != nil {
		return *v.Containers[containerName].RequestMemoryApplyMode
	}
	if v.RequestMemoryApplyMode != nil {
		return *v.RequestMemoryApplyMode
	}
	return ApplyModeEnforce
}

func (v *VpaWorkloadCfg) GetLimitCPUApplyMode(containerName string) ApplyMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitCPUApplyMode != nil {
		return *v.Containers[containerName].LimitCPUApplyMode
	}
	if v.LimitCPUApplyMode != nil {
		return *v.LimitCPUApplyMode
	}
	return ApplyModeEnforce
}

func (v *VpaWorkloadCfg) GetLimitMemoryApplyMode(containerName string) ApplyMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitMemoryApplyMode != nil {
		return *v.Containers[containerName].LimitMemoryApplyMode
	}
	if v.LimitMemoryApplyMode != nil {
		return *v.LimitMemoryApplyMode
	}
	return ApplyModeEnforce
}

func (v *VpaWorkloadCfg) GetLimitCPUCalculatorAlgo(containerName string) CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitCPUCalculatorAlgo != nil {
		return *v.Containers[containerName].LimitCPUCalculatorAlgo
	}
	if v.LimitCPUCalculatorAlgo != nil {
		return *v.LimitCPUCalculatorAlgo
	}
	var defaultLimitCPUCalculatorAlgo CalculatorAlgo
	defaultLimitCPUCalculatorAlgoParam := getEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_ALGO", "ratio")
	switch defaultLimitCPUCalculatorAlgoParam {
	case "ratio":
		defaultLimitCPUCalculatorAlgo = CalculatorAlgoRatio
	case "margin":
		defaultLimitCPUCalculatorAlgo = CalculatorAlgoMargin
	default:
		klog.Warningf("Unknown calculator algorithm: %s", defaultLimitCPUCalculatorAlgoParam)
		defaultLimitCPUCalculatorAlgo = CalculatorAlgoRatio
	}
	return defaultLimitCPUCalculatorAlgo
}

func (v *VpaWorkloadCfg) GetLimitMemoryCalculatorAlgo(containerName string) CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitMemoryCalculatorAlgo != nil {
		return *v.Containers[containerName].LimitMemoryCalculatorAlgo
	}
	if v.LimitMemoryCalculatorAlgo != nil {
		return *v.LimitMemoryCalculatorAlgo
	}
	var defaultLimitMemoryCalculatorAlgo CalculatorAlgo
	defaultLimitMemoryCalculatorAlgoParam := getEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_ALGO", "ratio")
	switch defaultLimitMemoryCalculatorAlgoParam {
	case "ratio":
		defaultLimitMemoryCalculatorAlgo = CalculatorAlgoRatio
	case "margin":
		defaultLimitMemoryCalculatorAlgo = CalculatorAlgoMargin
	default:
		klog.Warningf("Unknown calculator algorithm: %s", defaultLimitMemoryCalculatorAlgoParam)
		defaultLimitMemoryCalculatorAlgo = CalculatorAlgoRatio
	}
	return defaultLimitMemoryCalculatorAlgo
}

func (v *VpaWorkloadCfg) GetLimitMemoryCalculatorValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitMemoryCalculatorValue != nil {
		return *v.Containers[containerName].LimitMemoryCalculatorValue
	}
	if v.LimitMemoryCalculatorValue != nil {
		return *v.LimitMemoryCalculatorValue
	}
	return getEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_VALUE", "1")
}

func (v *VpaWorkloadCfg) GetLimitCPUCalculatorValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitCPUCalculatorValue != nil {
		return *v.Containers[containerName].LimitCPUCalculatorValue
	}
	if v.LimitCPUCalculatorValue != nil {
		return *v.LimitCPUCalculatorValue
	}
	return getEnv("OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_VALUE", "1")
}

func (v *VpaWorkloadCfg) GetUnprovidedApplyDefaultRequestCPUSource(containerName string) UnprovidedApplyDefaultMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].UnprovidedApplyDefaultRequestCPUSource != nil {
		return *v.Containers[containerName].UnprovidedApplyDefaultRequestCPUSource
	}
	if v.UnprovidedApplyDefaultRequestCPUSource != nil {
		return *v.UnprovidedApplyDefaultRequestCPUSource
	}
	switch getEnv("OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_CPU", "off") {
	case "off":
		return UnprovidedApplyDefaultModeOff
	case "max":
		fallthrough
	case "maxAllowed":
		return UnprovidedApplyDefaultModeMaxAllowed
	case "min":
		fallthrough
	case "minAllowed":
		return UnprovidedApplyDefaultModeMinAllowed
	default:
		return UnprovidedApplyDefaultModeValue
	}
}

func (v *VpaWorkloadCfg) GetUnprovidedApplyDefaultRequestCPUValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].UnprovidedApplyDefaultRequestCPUValue != nil {
		return *v.Containers[containerName].UnprovidedApplyDefaultRequestCPUValue
	}
	if v.UnprovidedApplyDefaultRequestCPUValue != nil {
		return *v.UnprovidedApplyDefaultRequestCPUValue
	}
	return ""
}

func (v *VpaWorkloadCfg) GetUnprovidedApplyDefaultRequestMemorySource(containerName string) UnprovidedApplyDefaultMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].UnprovidedApplyDefaultRequestMemorySource != nil {
		return *v.Containers[containerName].UnprovidedApplyDefaultRequestMemorySource
	}
	if v.UnprovidedApplyDefaultRequestMemorySource != nil {
		return *v.UnprovidedApplyDefaultRequestMemorySource
	}
	switch getEnv("OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_MEMORY", "off") {
	case "off":
		return UnprovidedApplyDefaultModeOff
	case "max":
		fallthrough
	case "maxAllowed":
		return UnprovidedApplyDefaultModeMaxAllowed
	case "min":
		fallthrough
	case "minAllowed":
		return UnprovidedApplyDefaultModeMinAllowed
	default:
		return UnprovidedApplyDefaultModeValue
	}
}

func (v *VpaWorkloadCfg) GetUnprovidedApplyDefaultRequestMemoryValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].UnprovidedApplyDefaultRequestMemoryValue != nil {
		return *v.Containers[containerName].UnprovidedApplyDefaultRequestMemoryValue
	}
	if v.UnprovidedApplyDefaultRequestMemoryValue != nil {
		return *v.UnprovidedApplyDefaultRequestMemoryValue
	}
	return ""
}

func (v *VpaWorkloadCfg) GetIncreaseRequestCpuAlgo(containerName string) CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestCpuAlgo != nil {
		return *v.Containers[containerName].IncreaseRequestCpuAlgo
	}
	if v.IncreaseRequestCpuAlgo != nil {
		return *v.IncreaseRequestCpuAlgo
	}
	return CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetIncreaseRequestMemoryAlgo(containerName string) CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestMemoryAlgo != nil {
		return *v.Containers[containerName].IncreaseRequestMemoryAlgo
	}
	if v.IncreaseRequestMemoryAlgo != nil {
		return *v.IncreaseRequestMemoryAlgo
	}
	return CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetIncreaseRequestCpuValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestCpuValue != nil {
		return *v.Containers[containerName].IncreaseRequestCpuValue
	}
	if v.IncreaseRequestCpuValue != nil {
		return *v.IncreaseRequestCpuValue
	}
	return getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_CPU_VALUE", "1")
}

func (v *VpaWorkloadCfg) GetIncreaseRequestMemoryValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestMemoryValue != nil {
		return *v.Containers[containerName].IncreaseRequestMemoryValue
	}
	if v.IncreaseRequestMemoryValue != nil {
		return *v.IncreaseRequestMemoryValue
	}
	return getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_VALUE", "1")
}

func (v *VpaWorkloadCfg) GetMinLimitCpu(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinLimitCpu != nil {
		return v.Containers[containerName].MinLimitCpu
	}
	if v.MinLimitCpu != nil {
		return v.MinLimitCpu
	}
	minLimitCpuStr := getEnv("OBLIK_DEFAULT_MIN_LIMIT_CPU", "")
	if minLimitCpuStr != "" {
		minLimitCpu, err := resource.ParseQuantity(minLimitCpuStr)
		if err != nil {
			klog.Warningf("Error parsing min-limit-cpu: %s, error: %s", minLimitCpuStr, err.Error())
		} else {
			return &minLimitCpu
		}
	}
	return nil
}

func (v *VpaWorkloadCfg) GetMaxLimitCpu(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MaxLimitCpu != nil {
		return v.Containers[containerName].MaxLimitCpu
	}
	if v.MaxLimitCpu != nil {
		return v.MaxLimitCpu
	}
	maxLimitCpuStr := getEnv("OBLIK_DEFAULT_MAX_LIMIT_CPU", "")
	if maxLimitCpuStr != "" {
		maxLimitCpu, err := resource.ParseQuantity(maxLimitCpuStr)
		if err != nil {
			klog.Warningf("Error parsing max-limit-cpu: %s, error: %s", maxLimitCpuStr, err.Error())
		} else {
			return &maxLimitCpu
		}
	}
	return nil
}

func (v *VpaWorkloadCfg) GetMinLimitMemory(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinLimitMemory != nil {
		return v.Containers[containerName].MinLimitMemory
	}
	if v.MinLimitMemory != nil {
		return v.MinLimitMemory
	}
	minLimitMemoryStr := getEnv("OBLIK_DEFAULT_MIN_LIMIT_MEMORY", "")
	if minLimitMemoryStr != "" {
		minLimitMemory, err := resource.ParseQuantity(minLimitMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing min-limit-memory: %s, error: %s", minLimitMemoryStr, err.Error())
		} else {
			return &minLimitMemory
		}
	}
	return nil
}

func (v *VpaWorkloadCfg) GetMaxLimitMemory(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MaxLimitMemory != nil {
		return v.Containers[containerName].MaxLimitMemory
	}
	if v.MaxLimitMemory != nil {
		return v.MaxLimitMemory
	}
	maxLimitMemoryStr := getEnv("OBLIK_DEFAULT_MAX_LIMIT_MEMORY", "")
	if maxLimitMemoryStr != "" {
		maxLimitMemory, err := resource.ParseQuantity(maxLimitMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing max-limit-memory: %s, error: %s", maxLimitMemoryStr, err.Error())
		} else {
			return &maxLimitMemory
		}
	}
	return nil
}

func (v *VpaWorkloadCfg) GetMinRequestCpu(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinRequestCpu != nil {
		return v.Containers[containerName].MinRequestCpu
	}
	if v.MinRequestCpu != nil {
		return v.MinRequestCpu
	}
	minRequestCpuStr := getEnv("OBLIK_DEFAULT_MIN_REQUEST_CPU", "")
	if minRequestCpuStr != "" {
		minRequestCpu, err := resource.ParseQuantity(minRequestCpuStr)
		if err != nil {
			klog.Warningf("Error parsing min-request-cpu: %s, error: %s", minRequestCpuStr, err.Error())
		} else {
			return &minRequestCpu
		}
	}
	return nil
}

func (v *VpaWorkloadCfg) GetMaxRequestCpu(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MaxRequestCpu != nil {
		return v.Containers[containerName].MaxRequestCpu
	}
	if v.MaxRequestCpu != nil {
		return v.MaxRequestCpu
	}
	maxRequestCpuStr := getEnv("OBLIK_DEFAULT_MAX_REQUEST_CPU", "")
	if maxRequestCpuStr != "" {
		maxRequestCpu, err := resource.ParseQuantity(maxRequestCpuStr)
		if err != nil {
			klog.Warningf("Error parsing max-request-cpu: %s, error: %s", maxRequestCpuStr, err.Error())
		} else {
			return &maxRequestCpu
		}
	}
	return nil
}

func (v *VpaWorkloadCfg) GetMinRequestMemory(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinRequestMemory != nil {
		return v.Containers[containerName].MinRequestMemory
	}
	if v.MinRequestMemory != nil {
		return v.MinRequestMemory
	}
	minRequestMemoryStr := getEnv("OBLIK_DEFAULT_MIN_REQUEST_MEMORY", "")
	if minRequestMemoryStr != "" {
		minRequestMemory, err := resource.ParseQuantity(minRequestMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing min-request-memory: %s, error: %s", minRequestMemoryStr, err.Error())
		} else {
			return &minRequestMemory
		}
	}
	return nil
}

func (v *VpaWorkloadCfg) GetMaxRequestMemory(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MaxRequestMemory != nil {
		return v.Containers[containerName].MaxRequestMemory
	}
	if v.MaxRequestMemory != nil {
		return v.MaxRequestMemory
	}
	maxRequestMemoryStr := getEnv("OBLIK_DEFAULT_MAX_REQUEST_MEMORY", "")
	if maxRequestMemoryStr != "" {
		maxRequestMemory, err := resource.ParseQuantity(maxRequestMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing max-request-memory: %s, error: %s", maxRequestMemoryStr, err.Error())
		} else {
			return &maxRequestMemory
		}
	}
	return nil
}

type LoadCfg struct {
	Key                string
	CronExpr           string
	CronMaxRandomDelay time.Duration

	RequestCPUApplyMode    *ApplyMode
	RequestMemoryApplyMode *ApplyMode
	LimitCPUApplyMode      *ApplyMode
	LimitMemoryApplyMode   *ApplyMode

	LimitCPUCalculatorAlgo     *CalculatorAlgo
	LimitMemoryCalculatorAlgo  *CalculatorAlgo
	LimitMemoryCalculatorValue *string
	LimitCPUCalculatorValue    *string

	UnprovidedApplyDefaultRequestCPUSource    *UnprovidedApplyDefaultMode
	UnprovidedApplyDefaultRequestCPUValue     *string
	UnprovidedApplyDefaultRequestMemorySource *UnprovidedApplyDefaultMode
	UnprovidedApplyDefaultRequestMemoryValue  *string

	IncreaseRequestCpuAlgo     *CalculatorAlgo
	IncreaseRequestMemoryAlgo  *CalculatorAlgo
	IncreaseRequestCpuValue    *string
	IncreaseRequestMemoryValue *string

	MinLimitCpu    *resource.Quantity
	MaxLimitCpu    *resource.Quantity
	MinLimitMemory *resource.Quantity
	MaxLimitMemory *resource.Quantity

	MinRequestCpu    *resource.Quantity
	MaxRequestCpu    *resource.Quantity
	MinRequestMemory *resource.Quantity
	MaxRequestMemory *resource.Quantity
}

func getAnnotationFromMap(name string, annotations map[string]string) string {
	return annotations["oblik.socialgouv.io/"+name]
}

func getVpaAnnotations(vpaResource *vpa.VerticalPodAutoscaler) map[string]string {
	annotations := vpaResource.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	return annotations
}

func createVpaWorkloadCfg(vpaResource *vpa.VerticalPodAutoscaler) *VpaWorkloadCfg {
	key := fmt.Sprintf("%s/%s", vpaResource.Namespace, vpaResource.Name)
	cfg := &VpaWorkloadCfg{
		Key: key,
		LoadCfg: &LoadCfg{
			Key: key,
		},
		Containers: map[string]*VpaContainerCfg{},
	}

	annotations := getVpaAnnotations(vpaResource)
	getAnnotation := func(key string) string {
		return getAnnotationFromMap(key, annotations)
	}

	cronExpr := getAnnotation("cron")
	if cronExpr == "" {
		cronExpr = getEnv("OBLIK_DEFAULT_CRON", defaultCron)
	}
	cfg.CronExpr = cronExpr

	cronAddRandomMax := getAnnotation("cron-add-random-max")
	if cronAddRandomMax == "" {
		cronAddRandomMax = getEnv("OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX", defaultCronAddRandomMax)
	}
	cfg.CronMaxRandomDelay = parseDuration(cronAddRandomMax, 120*time.Minute)

	loadVpaCommonCfg(cfg.LoadCfg, vpaResource, "")

	if vpaResource.Status.Recommendation != nil {
		for _, containerRecommendation := range vpaResource.Status.Recommendation.ContainerRecommendations {
			vpaContainerCfg := createVpaContainerCfg(vpaResource, containerRecommendation.ContainerName)
			cfg.Containers[containerRecommendation.ContainerName] = vpaContainerCfg
		}
	}

	return cfg
}

func createVpaContainerCfg(vpaResource *vpa.VerticalPodAutoscaler, containerName string) *VpaContainerCfg {
	key := fmt.Sprintf("%s/%s", vpaResource.Namespace, vpaResource.Name)
	cfg := &VpaContainerCfg{
		Key:           key,
		ContainerName: containerName,
		LoadCfg: &LoadCfg{
			Key: key,
		},
	}
	loadVpaCommonCfg(cfg.LoadCfg, vpaResource, containerName)
	return cfg
}

func loadVpaCommonCfg(cfg *LoadCfg, vpaResource *vpa.VerticalPodAutoscaler, annotationSuffix string) {

	annotations := getVpaAnnotations(vpaResource)

	getAnnotation := func(key string) string {
		if annotationSuffix != "" {
			key = key + "." + annotationSuffix
		}
		return getAnnotationFromMap(key, annotations)
	}

	if getAnnotation("request-cpu-apply-mode") == "off" {
		applyMode := ApplyModeOff
		cfg.RequestCPUApplyMode = &applyMode
	}

	if getAnnotation("request-memory-apply-mode") == "off" {
		applyMode := ApplyModeOff
		cfg.RequestMemoryApplyMode = &applyMode
	}

	if getAnnotation("limit-cpu-apply-mode") == "off" {
		applyMode := ApplyModeOff
		cfg.LimitCPUApplyMode = &applyMode
	}

	if getAnnotation("limit-memory-apply-mode") == "off" {
		applyMode := ApplyModeOff
		cfg.LimitMemoryApplyMode = &applyMode
	}

	limitCPUCalculatorAlgo := getAnnotation("limit-cpu-calculator-algo")
	switch limitCPUCalculatorAlgo {
	case "ratio":
		algo := CalculatorAlgoRatio
		cfg.LimitCPUCalculatorAlgo = &algo
	case "margin":
		algo := CalculatorAlgoMargin
		cfg.LimitCPUCalculatorAlgo = &algo
	default:
		if limitCPUCalculatorAlgo != "" {
			klog.Warningf("Unknown calculator algorithm: %s", limitCPUCalculatorAlgo)
		}
	}

	limitMemoryCalculatorAlgo := getAnnotation("limit-memory-calculator-algo")
	switch limitMemoryCalculatorAlgo {
	case "ratio":
		algo := CalculatorAlgoRatio
		cfg.LimitMemoryCalculatorAlgo = &algo
	case "margin":
		algo := CalculatorAlgoMargin
		cfg.LimitMemoryCalculatorAlgo = &algo
	default:
		if limitMemoryCalculatorAlgo != "" {
			klog.Warningf("Unknown calculator algorithm: %s", limitMemoryCalculatorAlgo)
		}
	}

	limitCPUCalculatorValue := getAnnotation("limit-cpu-calculator-value")
	limitMemoryCalculatorValue := getAnnotation("limit-memory-calculator-value")
	if limitCPUCalculatorValue != "" {
		cfg.LimitCPUCalculatorValue = &limitCPUCalculatorValue
	}
	if limitMemoryCalculatorValue != "" {
		cfg.LimitMemoryCalculatorValue = &limitMemoryCalculatorValue
	}

	unprovidedApplyDefaultRequestCPU := getAnnotation("unprovided-apply-default-request-cpu")
	if unprovidedApplyDefaultRequestCPU != "" {
		var unprovidedApplyDefaultRequestCPUSource UnprovidedApplyDefaultMode
		switch unprovidedApplyDefaultRequestCPU {
		case "off":
			unprovidedApplyDefaultRequestCPUSource = UnprovidedApplyDefaultModeOff
		case "max":
			fallthrough
		case "maxAllowed":
			unprovidedApplyDefaultRequestCPUSource = UnprovidedApplyDefaultModeMaxAllowed
		case "min":
			fallthrough
		case "minAllowed":
			unprovidedApplyDefaultRequestCPUSource = UnprovidedApplyDefaultModeMinAllowed
		default:
			unprovidedApplyDefaultRequestCPUSource = UnprovidedApplyDefaultModeValue
			cfg.UnprovidedApplyDefaultRequestCPUValue = &unprovidedApplyDefaultRequestCPU
		}
		cfg.UnprovidedApplyDefaultRequestCPUSource = &unprovidedApplyDefaultRequestCPUSource
	}

	unprovidedApplyDefaultRequestMemory := getAnnotation("unprovided-apply-default-request-memory")
	if unprovidedApplyDefaultRequestMemory != "" {
		var unprovidedApplyDefaultRequestMemorySource UnprovidedApplyDefaultMode
		switch unprovidedApplyDefaultRequestMemory {
		case "off":
			unprovidedApplyDefaultRequestMemorySource = UnprovidedApplyDefaultModeOff
		case "maxAllowed":
			unprovidedApplyDefaultRequestMemorySource = UnprovidedApplyDefaultModeMaxAllowed
		case "minAllowed":
			unprovidedApplyDefaultRequestMemorySource = UnprovidedApplyDefaultModeMinAllowed
		default:
			unprovidedApplyDefaultRequestMemorySource = UnprovidedApplyDefaultModeValue
			cfg.UnprovidedApplyDefaultRequestMemoryValue = &unprovidedApplyDefaultRequestMemory
		}
		cfg.UnprovidedApplyDefaultRequestMemorySource = &unprovidedApplyDefaultRequestMemorySource
	}

	increaseRequestCpuAlgo := getAnnotation("increase-request-cpu-algo")
	if increaseRequestCpuAlgo == "" {
		increaseRequestCpuAlgo = getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_CPU_ALGO", "ratio")
	}
	switch increaseRequestCpuAlgo {
	case "ratio":
		algo := CalculatorAlgoRatio
		cfg.IncreaseRequestCpuAlgo = &algo
	case "margin":
		algo := CalculatorAlgoMargin
		cfg.IncreaseRequestCpuAlgo = &algo
	default:
		klog.Warningf("Unknown calculator algorithm: %s", increaseRequestCpuAlgo)
		algo := CalculatorAlgoRatio
		cfg.IncreaseRequestCpuAlgo = &algo
	}

	increaseRequestMemoryAlgo := getAnnotation("increase-request-memory-algo")
	if increaseRequestMemoryAlgo == "" {
		increaseRequestMemoryAlgo = getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_ALGO", "ratio")
	}
	switch increaseRequestMemoryAlgo {
	case "ratio":
		algo := CalculatorAlgoRatio
		cfg.IncreaseRequestMemoryAlgo = &algo
	case "margin":
		algo := CalculatorAlgoMargin
		cfg.IncreaseRequestMemoryAlgo = &algo
	default:
		klog.Warningf("Unknown calculator algorithm: %s", increaseRequestMemoryAlgo)
		algo := CalculatorAlgoRatio
		cfg.IncreaseRequestMemoryAlgo = &algo
	}

	increaseRequestCpuValue := getAnnotation("increase-request-cpu-value")
	increaseRequestMemoryValue := getAnnotation("increase-request-memory-value")
	if increaseRequestCpuValue != "" {
		cfg.IncreaseRequestCpuValue = &increaseRequestCpuValue
	}
	if increaseRequestMemoryValue != "" {
		cfg.IncreaseRequestMemoryValue = &increaseRequestMemoryValue
	}

	minLimitCpuStr := getAnnotation("min-limit-cpu")
	if minLimitCpuStr != "" {
		minLimitCpu, err := resource.ParseQuantity(minLimitCpuStr)
		if err != nil {
			klog.Warningf("Error parsing min-limit-cpu: %s, error: %s", minLimitCpuStr, err.Error())
		} else {
			cfg.MinLimitCpu = &minLimitCpu
		}
	}

	maxLimitCpuStr := getAnnotation("max-limit-cpu")
	if maxLimitCpuStr != "" {
		maxLimitCpu, err := resource.ParseQuantity(maxLimitCpuStr)
		if err != nil {
			klog.Warningf("Error parsing max-limit-cpu: %s, error: %s", maxLimitCpuStr, err.Error())
		} else {
			cfg.MaxLimitCpu = &maxLimitCpu
		}
	}

	minLimitMemoryStr := getAnnotation("min-limit-memory")
	if minLimitMemoryStr != "" {
		minLimitMemory, err := resource.ParseQuantity(minLimitMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing min-limit-memory: %s, error: %s", minLimitMemoryStr, err.Error())
		} else {
			cfg.MinLimitMemory = &minLimitMemory
		}
	}

	maxLimitMemoryStr := getAnnotation("max-limit-memory")
	if maxLimitMemoryStr != "" {
		maxLimitMemory, err := resource.ParseQuantity(maxLimitMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing max-limit-memory: %s, error: %s", maxLimitMemoryStr, err.Error())
		} else {
			cfg.MaxLimitMemory = &maxLimitMemory
		}
	}

	minRequestCpuStr := getAnnotation("min-request-cpu")
	if minRequestCpuStr != "" {
		minRequestCpu, err := resource.ParseQuantity(minRequestCpuStr)
		if err != nil {
			klog.Warningf("Error parsing min-request-cpu: %s, error: %s", minRequestCpuStr, err.Error())
		} else {
			cfg.MinRequestCpu = &minRequestCpu
		}
	}

	maxRequestCpuStr := getAnnotation("max-request-cpu")
	if maxRequestCpuStr != "" {
		maxRequestCpu, err := resource.ParseQuantity(maxRequestCpuStr)
		if err != nil {
			klog.Warningf("Error parsing max-request-cpu: %s, error: %s", maxRequestCpuStr, err.Error())
		} else {
			cfg.MaxRequestCpu = &maxRequestCpu
		}
	}

	minRequestMemoryStr := getAnnotation("min-request-memory")
	if minRequestMemoryStr != "" {
		minRequestMemory, err := resource.ParseQuantity(minRequestMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing min-request-memory: %s, error: %s", minRequestMemoryStr, err.Error())
		} else {
			cfg.MinRequestMemory = &minRequestMemory
		}
	}

	maxRequestMemoryStr := getAnnotation("max-request-memory")
	if maxRequestMemoryStr != "" {
		maxRequestMemory, err := resource.ParseQuantity(maxRequestMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing max-request-memory: %s, error: %s", maxRequestMemoryStr, err.Error())
		} else {
			cfg.MaxRequestMemory = &maxRequestMemory
		}
	}
}
