package config

import (
	"fmt"
	"time"

	"github.com/SocialGouv/oblik/pkg/calculator"
	"github.com/SocialGouv/oblik/pkg/utils"
	"k8s.io/apimachinery/pkg/api/resource"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

type VpaWorkloadCfg struct {
	Key string
	*LoadCfg
	Containers map[string]*VpaContainerCfg
	DryRun     bool
}

func (v *VpaWorkloadCfg) GetDryRun() bool {
	return v.DryRun
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

func (v *VpaWorkloadCfg) GetLimitCPUCalculatorAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitCPUCalculatorAlgo != nil {
		return *v.Containers[containerName].LimitCPUCalculatorAlgo
	}
	if v.LimitCPUCalculatorAlgo != nil {
		return *v.LimitCPUCalculatorAlgo
	}
	defaultLimitCPUCalculatorAlgoParam := utils.GetEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_ALGO", "")
	if defaultLimitCPUCalculatorAlgoParam != "" {
		switch defaultLimitCPUCalculatorAlgoParam {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", defaultLimitCPUCalculatorAlgoParam)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetLimitMemoryCalculatorAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitMemoryCalculatorAlgo != nil {
		return *v.Containers[containerName].LimitMemoryCalculatorAlgo
	}
	if v.LimitMemoryCalculatorAlgo != nil {
		return *v.LimitMemoryCalculatorAlgo
	}
	defaultLimitMemoryCalculatorAlgoParam := utils.GetEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_ALGO", "")
	if defaultLimitMemoryCalculatorAlgoParam != "" {
		switch defaultLimitMemoryCalculatorAlgoParam {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", defaultLimitMemoryCalculatorAlgoParam)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetLimitMemoryCalculatorValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitMemoryCalculatorValue != nil {
		return *v.Containers[containerName].LimitMemoryCalculatorValue
	}
	if v.LimitMemoryCalculatorValue != nil {
		return *v.LimitMemoryCalculatorValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_VALUE", "1")
}

func (v *VpaWorkloadCfg) GetLimitCPUCalculatorValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitCPUCalculatorValue != nil {
		return *v.Containers[containerName].LimitCPUCalculatorValue
	}
	if v.LimitCPUCalculatorValue != nil {
		return *v.LimitCPUCalculatorValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_VALUE", "1")
}

func (v *VpaWorkloadCfg) GetUnprovidedApplyDefaultRequestCPUSource(containerName string) UnprovidedApplyDefaultMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].UnprovidedApplyDefaultRequestCPUSource != nil {
		return *v.Containers[containerName].UnprovidedApplyDefaultRequestCPUSource
	}
	if v.UnprovidedApplyDefaultRequestCPUSource != nil {
		return *v.UnprovidedApplyDefaultRequestCPUSource
	}
	switch utils.GetEnv("OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_CPU", "off") {
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
	switch utils.GetEnv("OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_MEMORY", "off") {
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

func (v *VpaWorkloadCfg) GetIncreaseRequestCpuAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestCpuAlgo != nil {
		return *v.Containers[containerName].IncreaseRequestCpuAlgo
	}
	if v.IncreaseRequestCpuAlgo != nil {
		return *v.IncreaseRequestCpuAlgo
	}
	increaseRequestCpuAlgo := utils.GetEnv("OBLIK_DEFAULT_INCREASE_REQUEST_CPU_ALGO", "")
	if increaseRequestCpuAlgo != "" {
		switch increaseRequestCpuAlgo {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", increaseRequestCpuAlgo)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetIncreaseRequestMemoryAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestMemoryAlgo != nil {
		return *v.Containers[containerName].IncreaseRequestMemoryAlgo
	}
	if v.IncreaseRequestMemoryAlgo != nil {
		return *v.IncreaseRequestMemoryAlgo
	}
	increaseRequestMemoryAlgo := utils.GetEnv("OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_ALGO", "")
	if increaseRequestMemoryAlgo != "" {
		switch increaseRequestMemoryAlgo {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", increaseRequestMemoryAlgo)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetIncreaseRequestCpuValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestCpuValue != nil {
		return *v.Containers[containerName].IncreaseRequestCpuValue
	}
	if v.IncreaseRequestCpuValue != nil {
		return *v.IncreaseRequestCpuValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_INCREASE_REQUEST_CPU_VALUE", "1")
}

func (v *VpaWorkloadCfg) GetIncreaseRequestMemoryValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestMemoryValue != nil {
		return *v.Containers[containerName].IncreaseRequestMemoryValue
	}
	if v.IncreaseRequestMemoryValue != nil {
		return *v.IncreaseRequestMemoryValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_VALUE", "1")
}

func (v *VpaWorkloadCfg) GetMinLimitCpu(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinLimitCpu != nil {
		return v.Containers[containerName].MinLimitCpu
	}
	if v.MinLimitCpu != nil {
		return v.MinLimitCpu
	}
	minLimitCpuStr := utils.GetEnv("OBLIK_DEFAULT_MIN_LIMIT_CPU", "")
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
	maxLimitCpuStr := utils.GetEnv("OBLIK_DEFAULT_MAX_LIMIT_CPU", "")
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
	minLimitMemoryStr := utils.GetEnv("OBLIK_DEFAULT_MIN_LIMIT_MEMORY", "")
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
	maxLimitMemoryStr := utils.GetEnv("OBLIK_DEFAULT_MAX_LIMIT_MEMORY", "")
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
	minRequestCpuStr := utils.GetEnv("OBLIK_DEFAULT_MIN_REQUEST_CPU", "")
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
	maxRequestCpuStr := utils.GetEnv("OBLIK_DEFAULT_MAX_REQUEST_CPU", "")
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
	minRequestMemoryStr := utils.GetEnv("OBLIK_DEFAULT_MIN_REQUEST_MEMORY", "")
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
	maxRequestMemoryStr := utils.GetEnv("OBLIK_DEFAULT_MAX_REQUEST_MEMORY", "")
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

func (v *VpaWorkloadCfg) GetMinDiffCpuRequestAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffCpuRequestAlgo != nil {
		return *v.Containers[containerName].MinDiffCpuRequestAlgo
	}
	if v.MinDiffCpuRequestAlgo != nil {
		return *v.MinDiffCpuRequestAlgo
	}
	minDiffCpuRequestAlgo := utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_CPU_REQUEST_ALGO", "")
	if minDiffCpuRequestAlgo != "" {
		switch minDiffCpuRequestAlgo {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", minDiffCpuRequestAlgo)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetMinDiffCpuRequestValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffCpuRequestValue != nil {
		return *v.Containers[containerName].MinDiffCpuRequestValue
	}
	if v.MinDiffCpuRequestValue != nil {
		return *v.MinDiffCpuRequestValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_CPU_REQUEST_VALUE", "0")
}

func (v *VpaWorkloadCfg) GetMinDiffMemoryRequestAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffMemoryRequestAlgo != nil {
		return *v.Containers[containerName].MinDiffMemoryRequestAlgo
	}
	if v.MinDiffMemoryRequestAlgo != nil {
		return *v.MinDiffMemoryRequestAlgo
	}
	minDiffMemoryRequestAlgo := utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_MEMORY_REQUEST_ALGO", "")
	if minDiffMemoryRequestAlgo != "" {
		switch minDiffMemoryRequestAlgo {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", minDiffMemoryRequestAlgo)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetMinDiffMemoryRequestValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffMemoryRequestValue != nil {
		return *v.Containers[containerName].MinDiffMemoryRequestValue
	}
	if v.MinDiffMemoryRequestValue != nil {
		return *v.MinDiffMemoryRequestValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_MEMORY_REQUEST_VALUE", "0")
}

func (v *VpaWorkloadCfg) GetMinDiffCpuLimitAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffCpuLimitAlgo != nil {
		return *v.Containers[containerName].MinDiffCpuLimitAlgo
	}
	if v.MinDiffCpuLimitAlgo != nil {
		return *v.MinDiffCpuLimitAlgo
	}
	minDiffCpuLimitAlgo := utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_CPU_LIMIT_ALGO", "")
	if minDiffCpuLimitAlgo != "" {
		switch minDiffCpuLimitAlgo {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", minDiffCpuLimitAlgo)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetMinDiffCpuLimitValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffCpuLimitValue != nil {
		return *v.Containers[containerName].MinDiffCpuLimitValue
	}
	if v.MinDiffCpuLimitValue != nil {
		return *v.MinDiffCpuLimitValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_CPU_LIMIT_VALUE", "0")
}

func (v *VpaWorkloadCfg) GetMinDiffMemoryLimitAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffMemoryLimitAlgo != nil {
		return *v.Containers[containerName].MinDiffMemoryLimitAlgo
	}
	if v.MinDiffMemoryLimitAlgo != nil {
		return *v.MinDiffMemoryLimitAlgo
	}
	minDiffMemoryLimitAlgo := utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_MEMORY_LIMIT_ALGO", "")
	if minDiffMemoryLimitAlgo != "" {
		switch minDiffMemoryLimitAlgo {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", minDiffMemoryLimitAlgo)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetMinDiffMemoryLimitValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffMemoryLimitValue != nil {
		return *v.Containers[containerName].MinDiffMemoryLimitValue
	}
	if v.MinDiffMemoryLimitValue != nil {
		return *v.MinDiffMemoryLimitValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_MEMORY_LIMIT_VALUE", "0")
}

func (v *VpaWorkloadCfg) GetMemoryRequestFromCpuEnabled(containerName string) bool {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryRequestFromCpuEnabled != nil {
		return *v.Containers[containerName].MemoryRequestFromCpuEnabled
	}
	if v.MemoryRequestFromCpuEnabled != nil {
		return *v.MemoryRequestFromCpuEnabled
	}
	memoryRequestFromCpuEnabled := utils.GetEnv("OBLIK_DEFAULT_MEMORY_REQUEST_FROM_CPU_ENABLED", "")
	return memoryRequestFromCpuEnabled == "true"
}

func (v *VpaWorkloadCfg) GetMemoryRequestFromCpuAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryRequestFromCpuAlgo != nil {
		return *v.Containers[containerName].MemoryRequestFromCpuAlgo
	}
	if v.MemoryRequestFromCpuAlgo != nil {
		return *v.MemoryRequestFromCpuAlgo
	}
	memoryRequestFromCpuAlgo := utils.GetEnv("OBLIK_DEFAULT_MEMORY_REQUEST_FROM_CPU_ALGO", "")
	if memoryRequestFromCpuAlgo != "" {
		switch memoryRequestFromCpuAlgo {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", memoryRequestFromCpuAlgo)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetMemoryRequestFromCpuValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryRequestFromCpuValue != nil {
		return *v.Containers[containerName].MemoryRequestFromCpuValue
	}
	if v.MemoryRequestFromCpuValue != nil {
		return *v.MemoryRequestFromCpuValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MEMORY_REQUEST_FROM_CPU_VALUE", "2")
}

func (v *VpaWorkloadCfg) GetMemoryLimitFromCpuEnabled(containerName string) bool {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryLimitFromCpuEnabled != nil {
		return *v.Containers[containerName].MemoryLimitFromCpuEnabled
	}
	if v.MemoryLimitFromCpuEnabled != nil {
		return *v.MemoryLimitFromCpuEnabled
	}
	memoryLimitFromCpuEnabled := utils.GetEnv("OBLIK_DEFAULT_MEMORY_LIMIT_FROM_CPU_ENABLED", "")
	return memoryLimitFromCpuEnabled == "true"
}

func (v *VpaWorkloadCfg) GetMemoryLimitFromCpuAlgo(containerName string) calculator.CalculatorAlgo {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryLimitFromCpuAlgo != nil {
		return *v.Containers[containerName].MemoryLimitFromCpuAlgo
	}
	if v.MemoryLimitFromCpuAlgo != nil {
		return *v.MemoryLimitFromCpuAlgo
	}
	memoryLimitFromCpuAlgo := utils.GetEnv("OBLIK_DEFAULT_MEMORY_LIMIT_FROM_CPU_ALGO", "")
	if memoryLimitFromCpuAlgo != "" {
		switch memoryLimitFromCpuAlgo {
		case "ratio":
			return calculator.CalculatorAlgoRatio
		case "margin":
			return calculator.CalculatorAlgoMargin
		default:
			klog.Warningf("Unknown calculator algorithm: %s", memoryLimitFromCpuAlgo)
		}
	}
	return calculator.CalculatorAlgoRatio
}

func (v *VpaWorkloadCfg) GetMemoryLimitFromCpuValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryLimitFromCpuValue != nil {
		return *v.Containers[containerName].MemoryLimitFromCpuValue
	}
	if v.MemoryLimitFromCpuValue != nil {
		return *v.MemoryLimitFromCpuValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MEMORY_LIMIT_FROM_CPU_VALUE", "2")
}

func (v *VpaWorkloadCfg) GetRequestApplyTarget(containerName string) ApplyTarget {
	if v.Containers[containerName] != nil && v.Containers[containerName].RequestApplyTarget != nil {
		return *v.Containers[containerName].RequestApplyTarget
	}
	if v.RequestApplyTarget != nil {
		return *v.RequestApplyTarget
	}
	requestApplyTarget := utils.GetEnv("OBLIK_DEFAULT_REQUEST_APPLY_TARGET", "")
	if requestApplyTarget != "" {
		switch requestApplyTarget {
		case "lowerBound":
			fallthrough
		case "frugal":
			return ApplyTargetFrugal
		case "target":
			fallthrough
		case "balanced":
			return ApplyTargetBalanced
		case "upperBound":
			fallthrough
		case "peak":
			return ApplyTargetPeak
		default:
			klog.Warningf("Unknown apply-target: %s", requestApplyTarget)
		}
	}
	return ApplyTargetBalanced
}

func (v *VpaWorkloadCfg) GetRequestCpuApplyTarget(containerName string) ApplyTarget {
	if v.Containers[containerName] != nil && v.Containers[containerName].RequestCpuApplyTarget != nil {
		return *v.Containers[containerName].RequestCpuApplyTarget
	}
	if v.RequestCpuApplyTarget != nil {
		return *v.RequestCpuApplyTarget
	}
	requestCpuApplyTarget := utils.GetEnv("OBLIK_DEFAULT_REQUEST_CPU_APPLY_TARGET", "")
	if requestCpuApplyTarget != "" {
		switch requestCpuApplyTarget {
		case "lowerBound":
			fallthrough
		case "frugal":
			return ApplyTargetFrugal
		case "target":
			fallthrough
		case "balanced":
			return ApplyTargetBalanced
		case "upperBound":
			fallthrough
		case "peak":
			return ApplyTargetPeak
		default:
			klog.Warningf("Unknown apply-target: %s", requestCpuApplyTarget)
		}
	}
	return v.GetRequestApplyTarget(containerName)
}

func (v *VpaWorkloadCfg) GetRequestMemoryApplyTarget(containerName string) ApplyTarget {
	if v.Containers[containerName] != nil && v.Containers[containerName].RequestMemoryApplyTarget != nil {
		return *v.Containers[containerName].RequestMemoryApplyTarget
	}
	if v.RequestMemoryApplyTarget != nil {
		return *v.RequestMemoryApplyTarget
	}
	requestMemoryApplyTarget := utils.GetEnv("OBLIK_DEFAULT_REQUEST_MEMORY_APPLY_TARGET", "")
	if requestMemoryApplyTarget != "" {
		switch requestMemoryApplyTarget {
		case "lowerBound":
			fallthrough
		case "frugal":
			return ApplyTargetFrugal
		case "target":
			fallthrough
		case "balanced":
			return ApplyTargetBalanced
		case "upperBound":
			fallthrough
		case "peak":
			return ApplyTargetPeak
		default:
			klog.Warningf("Unknown apply-target: %s", requestMemoryApplyTarget)
		}
	}
	return v.GetRequestApplyTarget(containerName)
}

func CreateVpaWorkloadCfg(vpaResource *vpa.VerticalPodAutoscaler) *VpaWorkloadCfg {
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
		cronExpr = utils.GetEnv("OBLIK_DEFAULT_CRON", defaultCron)
	}
	cfg.CronExpr = cronExpr

	cronAddRandomMax := getAnnotation("cron-add-random-max")
	if cronAddRandomMax == "" {
		cronAddRandomMax = utils.GetEnv("OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX", defaultCronAddRandomMax)
	}
	cfg.CronMaxRandomDelay = utils.ParseDuration(cronAddRandomMax, 120*time.Minute)

	dryRunStr := getAnnotation("dry-run")
	if dryRunStr == "" {
		dryRunStr = utils.GetEnv("OBLIK_DEFAULT_DRY_RUN", "false")
	}
	if dryRunStr == "true" {
		cfg.DryRun = true
	}

	loadVpaCommonCfg(cfg.LoadCfg, vpaResource, "")

	if vpaResource.Status.Recommendation != nil {
		for _, containerRecommendation := range vpaResource.Status.Recommendation.ContainerRecommendations {
			vpaContainerCfg := createVpaContainerCfg(vpaResource, containerRecommendation.ContainerName)
			cfg.Containers[containerRecommendation.ContainerName] = vpaContainerCfg
		}
	}

	return cfg
}
