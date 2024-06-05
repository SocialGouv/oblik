package controller

import (
	"fmt"
	"time"

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

	cronExpr := annotations["oblik.socialgouv.io/cron"]
	if cronExpr == "" {
		cronExpr = getEnv("OBLIK_DEFAULT_CRON", defaultCron)
	}
	cfg.CronExpr = cronExpr

	cronAddRandomMax := annotations["oblik.socialgouv.io/cron-add-random-max"]
	if cronAddRandomMax == "" {
		cronAddRandomMax = getEnv("OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX", defaultCronAddRandomMax)
	}
	cfg.CronMaxRandomDelay = parseDuration(cronAddRandomMax, 120*time.Minute)

	if annotations["oblik.socialgouv.io/request-cpu-apply-mode"] == "off" {
		cfg.RequestCPUApplyMode = ApplyModeOff
	} else {
		cfg.RequestCPUApplyMode = ApplyModeEnforce
	}

	if annotations["oblik.socialgouv.io/request-memory-apply-mode"] == "off" {
		cfg.RequestMemoryApplyMode = ApplyModeOff
	} else {
		cfg.RequestMemoryApplyMode = ApplyModeEnforce
	}

	if annotations["oblik.socialgouv.io/limit-cpu-apply-mode"] == "off" {
		cfg.LimitCPUApplyMode = ApplyModeOff
	} else {
		cfg.LimitCPUApplyMode = ApplyModeEnforce
	}

	if annotations["oblik.socialgouv.io/limit-memory-apply-mode"] == "off" {
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

	limitCPUCalculatorAlgo := annotations["oblik.socialgouv.io/limit-cpu-calculator-algo"]
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

	limitMemoryCalculatorAlgo := annotations["oblik.socialgouv.io/limit-memory-calculator-algo"]
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

	cfg.LimitMemoryCalculatorValue = annotations["oblik.socialgouv.io/limit-memory-calculator-value"]
	cfg.LimitCPUCalculatorValue = annotations["oblik.socialgouv.io/limit-cpu-calculator-value"]

	if cfg.LimitCPUCalculatorValue == "" {
		cfg.LimitCPUCalculatorValue = getEnv("OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_VALUE", "1")
	}
	if cfg.LimitMemoryCalculatorValue == "" {
		cfg.LimitMemoryCalculatorValue = getEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_VALUE", "1")
	}

	unprovidedApplyDefaultRequestCPU := annotations["oblik.socialgouv.io/unprovided-apply-default-request-cpu"]
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

	unprovidedApplyDefaultRequestMemory := annotations["oblik.socialgouv.io/unprovided-apply-default-request-memory"]
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

	increaseRequestCpuAlgo := annotations["oblik.socialgouv.io/increase-request-cpu-algo"]
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

	increaseRequestMemoryAlgo := annotations["oblik.socialgouv.io/increase-request-memory-algo"]
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

	cfg.IncreaseRequestCpuValue = annotations["oblik.socialgouv.io/increase-request-cpu-value"]
	cfg.IncreaseRequestMemoryValue = annotations["oblik.socialgouv.io/increase-request-memory-value"]

	if cfg.IncreaseRequestCpuValue == "" {
		cfg.IncreaseRequestCpuValue = getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_CPU_VALUE", "1")
	}
	if cfg.LimitMemoryCalculatorValue == "" {
		cfg.LimitMemoryCalculatorValue = getEnv("OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_VALUE", "1")
	}

	return cfg
}
