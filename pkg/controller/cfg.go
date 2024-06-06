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

type VPAOblikConfig struct {
	Key string

	CronExpr           string
	CronMaxRandomDelay time.Duration

	RequestCPUApplyMode    ApplyMode
	RequestMemoryApplyMode ApplyMode
	LimitCPUApplyMode      ApplyMode
	LimitMemoryApplyMode   ApplyMode

	LimitCPUCalculatorAlgo     CalculatorAlgo
	LimitMemoryCalculatorAlgo  CalculatorAlgo
	LimitMemoryCalculatorValue string
	LimitCPUCalculatorValue    string

	UnprovidedApplyDefaultRequestCPUSource    UnprovidedApplyDefaultMode
	UnprovidedApplyDefaultRequestCPUValue     string
	UnprovidedApplyDefaultRequestMemorySource UnprovidedApplyDefaultMode
	UnprovidedApplyDefaultRequestMemoryValue  string

	IncreaseRequestCpuAlgo     CalculatorAlgo
	IncreaseRequestMemoryAlgo  CalculatorAlgo
	IncreaseRequestCpuValue    string
	IncreaseRequestMemoryValue string

	MinLimitCpu    *resource.Quantity
	MaxLimitCpu    *resource.Quantity
	MinLimitMemory *resource.Quantity
	MaxLimitMemory *resource.Quantity
}

func getAnnotation(name string, annotations map[string]string) string {
	return annotations["oblik.socialgouv.io/"+name]
}

func createVPAOblikConfig(vpa *vpa.VerticalPodAutoscaler) *VPAOblikConfig {
	key := fmt.Sprintf("%s/%s", vpa.Namespace, vpa.Name)

	cfg := &VPAOblikConfig{
		Key: key,
	}

	annotations := vpa.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	cronExpr := getAnnotation("cron", annotations)
	if cronExpr == "" {
		cronExpr = getEnv("OBLIK_DEFAULT_CRON", defaultCron)
	}
	cfg.CronExpr = cronExpr

	cronAddRandomMax := getAnnotation("cron-add-random-max", annotations)
	if cronAddRandomMax == "" {
		cronAddRandomMax = getEnv("OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX", defaultCronAddRandomMax)
	}
	cfg.CronMaxRandomDelay = parseDuration(cronAddRandomMax, 120*time.Minute)

	if getAnnotation("request-cpu-apply-mode", annotations) == "off" {
		cfg.RequestCPUApplyMode = ApplyModeOff
	} else {
		cfg.RequestCPUApplyMode = ApplyModeEnforce
	}

	if getAnnotation("request-memory-apply-mode", annotations) == "off" {
		cfg.RequestMemoryApplyMode = ApplyModeOff
	} else {
		cfg.RequestMemoryApplyMode = ApplyModeEnforce
	}

	if getAnnotation("limit-cpu-apply-mode", annotations) == "off" {
		cfg.LimitCPUApplyMode = ApplyModeOff
	} else {
		cfg.LimitCPUApplyMode = ApplyModeEnforce
	}

	if getAnnotation("limit-memory-apply-mode", annotations) == "off" {
		cfg.LimitMemoryApplyMode = ApplyModeOff
	} else {
		cfg.LimitMemoryApplyMode = ApplyModeEnforce
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

	limitCPUCalculatorAlgo := getAnnotation("limit-cpu-calculator-algo", annotations)
	switch limitCPUCalculatorAlgo {
	case "ratio":
		cfg.LimitCPUCalculatorAlgo = CalculatorAlgoRatio
	case "margin":
		cfg.LimitCPUCalculatorAlgo = CalculatorAlgoMargin
	default:
		if limitCPUCalculatorAlgo != "" {
			klog.Warningf("Unknown calculator algorithm: %s", limitCPUCalculatorAlgo)
		}
		cfg.LimitCPUCalculatorAlgo = defaultLimitCPUCalculatorAlgo
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

	limitMemoryCalculatorAlgo := getAnnotation("limit-memory-calculator-algo", annotations)
	switch limitMemoryCalculatorAlgo {
	case "ratio":
		cfg.LimitMemoryCalculatorAlgo = CalculatorAlgoRatio
	case "margin":
		cfg.LimitMemoryCalculatorAlgo = CalculatorAlgoMargin
	default:
		if limitMemoryCalculatorAlgo != "" {
			klog.Warningf("Unknown calculator algorithm: %s", limitMemoryCalculatorAlgo)
		}
		cfg.LimitMemoryCalculatorAlgo = defaultLimitMemoryCalculatorAlgo
	}

	cfg.LimitMemoryCalculatorValue = getAnnotation("limit-memory-calculator-value", annotations)
	cfg.LimitCPUCalculatorValue = getAnnotation("limit-cpu-calculator-value", annotations)

	if cfg.LimitCPUCalculatorValue == "" {
		cfg.LimitCPUCalculatorValue = getEnv("OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_VALUE", "1")
	}
	if cfg.LimitMemoryCalculatorValue == "" {
		cfg.LimitMemoryCalculatorValue = getEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_VALUE", "1")
	}

	unprovidedApplyDefaultRequestCPU := getAnnotation("unprovided-apply-default-request-cpu", annotations)
	if unprovidedApplyDefaultRequestCPU == "" {
		unprovidedApplyDefaultRequestCPU = getEnv("OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_CPU", "off")
	}

	switch unprovidedApplyDefaultRequestCPU {
	case "off":
		cfg.UnprovidedApplyDefaultRequestCPUSource = UnprovidedApplyDefaultModeOff
	case "maxAllowed":
		cfg.UnprovidedApplyDefaultRequestCPUSource = UnprovidedApplyDefaultModeMaxAllowed
	case "minAllowed":
		cfg.UnprovidedApplyDefaultRequestCPUSource = UnprovidedApplyDefaultModeMinAllowed
	default:
		cfg.UnprovidedApplyDefaultRequestCPUSource = UnprovidedApplyDefaultModeValue
		cfg.UnprovidedApplyDefaultRequestCPUValue = unprovidedApplyDefaultRequestCPU
	}

	unprovidedApplyDefaultRequestMemory := getAnnotation("unprovided-apply-default-request-memory", annotations)
	if unprovidedApplyDefaultRequestMemory == "" {
		unprovidedApplyDefaultRequestMemory = getEnv("OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_MEMORY", "off")
	}

	switch unprovidedApplyDefaultRequestMemory {
	case "off":
		cfg.UnprovidedApplyDefaultRequestMemorySource = UnprovidedApplyDefaultModeOff
	case "maxAllowed":
		cfg.UnprovidedApplyDefaultRequestMemorySource = UnprovidedApplyDefaultModeMaxAllowed
	case "minAllowed":
		cfg.UnprovidedApplyDefaultRequestMemorySource = UnprovidedApplyDefaultModeMinAllowed
	default:
		cfg.UnprovidedApplyDefaultRequestMemorySource = UnprovidedApplyDefaultModeValue
		cfg.UnprovidedApplyDefaultRequestMemoryValue = unprovidedApplyDefaultRequestMemory
	}

	increaseRequestCpuAlgo := getAnnotation("increase-request-cpu-algo", annotations)
	if increaseRequestCpuAlgo == "" {
		increaseRequestCpuAlgo = getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_CPU_ALGO", "ratio")
	}
	switch increaseRequestCpuAlgo {
	case "ratio":
		cfg.IncreaseRequestCpuAlgo = CalculatorAlgoRatio
	case "margin":
		cfg.IncreaseRequestCpuAlgo = CalculatorAlgoMargin
	default:
		klog.Warningf("Unknown calculator algorithm: %s", increaseRequestCpuAlgo)
		cfg.IncreaseRequestCpuAlgo = CalculatorAlgoRatio
	}

	increaseRequestMemoryAlgo := getAnnotation("increase-request-memory-algo", annotations)
	if increaseRequestMemoryAlgo == "" {
		increaseRequestMemoryAlgo = getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_ALGO", "ratio")
	}
	switch increaseRequestMemoryAlgo {
	case "ratio":
		cfg.IncreaseRequestMemoryAlgo = CalculatorAlgoRatio
	case "margin":
		cfg.IncreaseRequestMemoryAlgo = CalculatorAlgoMargin
	default:
		klog.Warningf("Unknown calculator algorithm: %s", increaseRequestMemoryAlgo)
		cfg.IncreaseRequestMemoryAlgo = CalculatorAlgoRatio
	}

	cfg.IncreaseRequestCpuValue = getAnnotation("increase-request-cpu-value", annotations)
	cfg.IncreaseRequestMemoryValue = getAnnotation("increase-request-memory-value", annotations)

	if cfg.IncreaseRequestCpuValue == "" {
		cfg.IncreaseRequestCpuValue = getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_CPU_VALUE", "1")
	}
	if cfg.IncreaseRequestMemoryValue == "" {
		cfg.IncreaseRequestMemoryValue = getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_VALUE", "1")
	}

	minLimitCpuStr := getAnnotation("min-limit-cpu", annotations)
	if minLimitCpuStr == "" {
		minLimitCpuStr = getEnv("OBLIK_DEFAULT_MIN_LIMIT_CPU", "")
	}
	if minLimitCpuStr != "" {
		minLimitCpu, err := resource.ParseQuantity(minLimitCpuStr)
		if err != nil {
			klog.Warningf("Error parsing min-limit-cpu: %s, error: %s", minLimitCpuStr, err.Error())
		} else {
			cfg.MinLimitCpu = &minLimitCpu
		}
	}

	maxLimitCpuStr := getAnnotation("max-limit-cpu", annotations)
	if maxLimitCpuStr == "" {
		maxLimitCpuStr = getEnv("OBLIK_DEFAULT_MAX_LIMIT_CPU", "")
	}
	if maxLimitCpuStr != "" {
		maxLimitCpu, err := resource.ParseQuantity(maxLimitCpuStr)
		if err != nil {
			klog.Warningf("Error parsing max-limit-cpu: %s, error: %s", maxLimitCpuStr, err.Error())
		} else {
			cfg.MinLimitCpu = &maxLimitCpu
		}
	}

	minLimitMemoryStr := getAnnotation("min-limit-memory", annotations)
	if minLimitMemoryStr == "" {
		minLimitMemoryStr = getEnv("OBLIK_DEFAULT_MIN_LIMIT_MEMORY", "")
	}
	if minLimitMemoryStr != "" {
		minLimitMemory, err := resource.ParseQuantity(minLimitMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing min-limit-memory: %s, error: %s", minLimitMemoryStr, err.Error())
		} else {
			cfg.MinLimitMemory = &minLimitMemory
		}
	}

	maxLimitMemoryStr := getAnnotation("max-limit-memory", annotations)
	if maxLimitMemoryStr == "" {
		maxLimitMemoryStr = getEnv("OBLIK_DEFAULT_MAX_LIMIT_MEMORY", "")
	}
	if maxLimitMemoryStr != "" {
		maxLimitMemory, err := resource.ParseQuantity(maxLimitMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing max-limit-memory: %s, error: %s", maxLimitMemoryStr, err.Error())
		} else {
			cfg.MinLimitMemory = &maxLimitMemory
		}
	}

	return cfg
}
