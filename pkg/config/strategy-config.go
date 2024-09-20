package config

import (
	"fmt"
	"time"

	"github.com/SocialGouv/oblik/pkg/calculator"
	"github.com/SocialGouv/oblik/pkg/utils"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

func CreateStrategyConfig(configurable *Configurable) *StrategyConfig {
	key := fmt.Sprintf("%s/%s", configurable.GetNamespace(), configurable.GetName())
	cfg := &StrategyConfig{
		Key: key,
		LoadCfg: &LoadCfg{
			Key: key,
		},
		Containers: map[string]*ContainerConfig{},
	}

	annotations := getAnnotations(configurable)
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

	webhookEnabled := getAnnotation("webhook-enabled")
	if webhookEnabled == "" {
		webhookEnabled = utils.GetEnv("OBLIK_DEFAULT_WEBHOOK_ENABLED", "true")
	}
	if webhookEnabled == "true" {
		cfg.WebhookEnabled = true
	}

	loadAnnotableCommonCfg(cfg.LoadCfg, configurable, "")

	containerNames := configurable.GetContainerNames()
	for _, containerName := range containerNames {
		containerConfig := createContainerConfig(configurable, containerName)
		cfg.Containers[containerName] = containerConfig
	}

	return cfg
}

type StrategyConfig struct {
	Key                string
	CronExpr           string
	CronMaxRandomDelay time.Duration
	DryRun             bool
	WebhookEnabled     bool
	Containers         map[string]*ContainerConfig
	*LoadCfg
}

func (v *StrategyConfig) GetDryRun() bool {
	return v.DryRun
}

func (v *StrategyConfig) GetRequestCPUApplyMode(containerName string) ApplyMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].RequestCPUApplyMode != nil {
		return *v.Containers[containerName].RequestCPUApplyMode
	}
	if v.RequestCPUApplyMode != nil {
		return *v.RequestCPUApplyMode
	}
	return ApplyModeEnforce
}

func (v *StrategyConfig) GetRequestMemoryApplyMode(containerName string) ApplyMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].RequestMemoryApplyMode != nil {
		return *v.Containers[containerName].RequestMemoryApplyMode
	}
	if v.RequestMemoryApplyMode != nil {
		return *v.RequestMemoryApplyMode
	}
	return ApplyModeEnforce
}

func (v *StrategyConfig) GetLimitCPUApplyMode(containerName string) ApplyMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitCPUApplyMode != nil {
		return *v.Containers[containerName].LimitCPUApplyMode
	}
	if v.LimitCPUApplyMode != nil {
		return *v.LimitCPUApplyMode
	}
	return ApplyModeEnforce
}

func (v *StrategyConfig) GetLimitMemoryApplyMode(containerName string) ApplyMode {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitMemoryApplyMode != nil {
		return *v.Containers[containerName].LimitMemoryApplyMode
	}
	if v.LimitMemoryApplyMode != nil {
		return *v.LimitMemoryApplyMode
	}
	return ApplyModeEnforce
}

func (v *StrategyConfig) GetLimitCPUCalculatorAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetLimitMemoryCalculatorAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetLimitMemoryCalculatorValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitMemoryCalculatorValue != nil {
		return *v.Containers[containerName].LimitMemoryCalculatorValue
	}
	if v.LimitMemoryCalculatorValue != nil {
		return *v.LimitMemoryCalculatorValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_VALUE", "1")
}

func (v *StrategyConfig) GetLimitCPUCalculatorValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitCPUCalculatorValue != nil {
		return *v.Containers[containerName].LimitCPUCalculatorValue
	}
	if v.LimitCPUCalculatorValue != nil {
		return *v.LimitCPUCalculatorValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_VALUE", "1")
}

func (v *StrategyConfig) GetUnprovidedApplyDefaultRequestCPUSource(containerName string) UnprovidedApplyDefaultMode {
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

func (v *StrategyConfig) GetUnprovidedApplyDefaultRequestCPUValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].UnprovidedApplyDefaultRequestCPUValue != nil {
		return *v.Containers[containerName].UnprovidedApplyDefaultRequestCPUValue
	}
	if v.UnprovidedApplyDefaultRequestCPUValue != nil {
		return *v.UnprovidedApplyDefaultRequestCPUValue
	}
	value := utils.GetEnv("OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_CPU", "")
	switch value {
	case "off":
		return ""
	case "max":
		fallthrough
	case "maxAllowed":
		return ""
	case "min":
		fallthrough
	case "minAllowed":
		return ""
	default:
		return value
	}
}

func (v *StrategyConfig) GetUnprovidedApplyDefaultRequestMemorySource(containerName string) UnprovidedApplyDefaultMode {
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

func (v *StrategyConfig) GetUnprovidedApplyDefaultRequestMemoryValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].UnprovidedApplyDefaultRequestMemoryValue != nil {
		return *v.Containers[containerName].UnprovidedApplyDefaultRequestMemoryValue
	}
	if v.UnprovidedApplyDefaultRequestMemoryValue != nil {
		return *v.UnprovidedApplyDefaultRequestMemoryValue
	}
	value := utils.GetEnv("OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_MEMORY", "")
	switch value {
	case "off":
		return ""
	case "max":
		fallthrough
	case "maxAllowed":
		return ""
	case "min":
		fallthrough
	case "minAllowed":
		return ""
	default:
		return value
	}
}

func (v *StrategyConfig) GetIncreaseRequestCpuAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetIncreaseRequestMemoryAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetIncreaseRequestCpuValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestCpuValue != nil {
		return *v.Containers[containerName].IncreaseRequestCpuValue
	}
	if v.IncreaseRequestCpuValue != nil {
		return *v.IncreaseRequestCpuValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_INCREASE_REQUEST_CPU_VALUE", "1")
}

func (v *StrategyConfig) GetIncreaseRequestMemoryValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].IncreaseRequestMemoryValue != nil {
		return *v.Containers[containerName].IncreaseRequestMemoryValue
	}
	if v.IncreaseRequestMemoryValue != nil {
		return *v.IncreaseRequestMemoryValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_VALUE", "1")
}

func (v *StrategyConfig) GetMinLimitCpu(containerName string) *resource.Quantity {
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

func (v *StrategyConfig) GetMaxLimitCpu(containerName string) *resource.Quantity {
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

func (v *StrategyConfig) GetMinLimitMemory(containerName string) *resource.Quantity {
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

func (v *StrategyConfig) GetMaxLimitMemory(containerName string) *resource.Quantity {
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

func (v *StrategyConfig) GetMinRequestCpu(containerName string) *resource.Quantity {
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

func (v *StrategyConfig) GetMaxRequestCpu(containerName string) *resource.Quantity {
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

func (v *StrategyConfig) GetMinRequestMemory(containerName string) *resource.Quantity {
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

func (v *StrategyConfig) GetMaxRequestMemory(containerName string) *resource.Quantity {
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

func (v *StrategyConfig) GetMinAllowedRecommendationCpu(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinAllowedRecommendationCpu != nil {
		return v.Containers[containerName].MinAllowedRecommendationCpu
	}
	if v.MinAllowedRecommendationCpu != nil {
		return v.MinAllowedRecommendationCpu
	}
	MinAllowedRecommendationCpuStr := utils.GetEnv("OBLIK_DEFAULT_MIN_ALLOWED_RECOMMENDATION_CPU", "25m")
	if MinAllowedRecommendationCpuStr != "" {
		MinAllowedRecommendationCpu, err := resource.ParseQuantity(MinAllowedRecommendationCpuStr)
		if err != nil {
			klog.Warningf("Error parsing min-allowed-recommendation-cpu: %s, error: %s", MinAllowedRecommendationCpuStr, err.Error())
		} else {
			return &MinAllowedRecommendationCpu
		}
	}
	return nil
}

func (v *StrategyConfig) GetMaxAllowedRecommendationCpu(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MaxAllowedRecommendationCpu != nil {
		return v.Containers[containerName].MaxAllowedRecommendationCpu
	}
	if v.MaxAllowedRecommendationCpu != nil {
		return v.MaxAllowedRecommendationCpu
	}
	MaxAllowedRecommendationCpuStr := utils.GetEnv("OBLIK_DEFAULT_MAX_ALLOWED_RECOMMENDATION_CPU", "")
	if MaxAllowedRecommendationCpuStr != "" {
		MaxAllowedRecommendationCpu, err := resource.ParseQuantity(MaxAllowedRecommendationCpuStr)
		if err != nil {
			klog.Warningf("Error parsing max-allowed-recommendation-cpu: %s, error: %s", MaxAllowedRecommendationCpuStr, err.Error())
		} else {
			return &MaxAllowedRecommendationCpu
		}
	}
	return nil
}

func (v *StrategyConfig) GetMinAllowedRecommendationMemory(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinAllowedRecommendationMemory != nil {
		return v.Containers[containerName].MinAllowedRecommendationMemory
	}
	if v.MinAllowedRecommendationMemory != nil {
		return v.MinAllowedRecommendationMemory
	}
	MinAllowedRecommendationMemoryStr := utils.GetEnv("OBLIK_DEFAULT_MIN_ALLOWED_RECOMMENDATION_MEMORY", "250Mi")
	if MinAllowedRecommendationMemoryStr != "" {
		MinAllowedRecommendationMemory, err := resource.ParseQuantity(MinAllowedRecommendationMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing min-allowed-recommendation-memory: %s, error: %s", MinAllowedRecommendationMemoryStr, err.Error())
		} else {
			return &MinAllowedRecommendationMemory
		}
	}
	return nil
}

func (v *StrategyConfig) GetMaxAllowedRecommendationMemory(containerName string) *resource.Quantity {
	if v.Containers[containerName] != nil && v.Containers[containerName].MaxAllowedRecommendationMemory != nil {
		return v.Containers[containerName].MaxAllowedRecommendationMemory
	}
	if v.MaxAllowedRecommendationMemory != nil {
		return v.MaxAllowedRecommendationMemory
	}
	MaxAllowedRecommendationMemoryStr := utils.GetEnv("OBLIK_DEFAULT_MAX_ALLOWED_RECOMMENDATION_MEMORY", "25m")
	if MaxAllowedRecommendationMemoryStr != "" {
		MaxAllowedRecommendationMemory, err := resource.ParseQuantity(MaxAllowedRecommendationMemoryStr)
		if err != nil {
			klog.Warningf("Error parsing max-allowed-recommendation-memory: %s, error: %s", MaxAllowedRecommendationMemoryStr, err.Error())
		} else {
			return &MaxAllowedRecommendationMemory
		}
	}
	return nil
}

func (v *StrategyConfig) GetMinDiffCpuRequestAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetMinDiffCpuRequestValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffCpuRequestValue != nil {
		return *v.Containers[containerName].MinDiffCpuRequestValue
	}
	if v.MinDiffCpuRequestValue != nil {
		return *v.MinDiffCpuRequestValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_CPU_REQUEST_VALUE", "0")
}

func (v *StrategyConfig) GetMinDiffMemoryRequestAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetMinDiffMemoryRequestValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffMemoryRequestValue != nil {
		return *v.Containers[containerName].MinDiffMemoryRequestValue
	}
	if v.MinDiffMemoryRequestValue != nil {
		return *v.MinDiffMemoryRequestValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_MEMORY_REQUEST_VALUE", "0")
}

func (v *StrategyConfig) GetMinDiffCpuLimitAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetMinDiffCpuLimitValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffCpuLimitValue != nil {
		return *v.Containers[containerName].MinDiffCpuLimitValue
	}
	if v.MinDiffCpuLimitValue != nil {
		return *v.MinDiffCpuLimitValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_CPU_LIMIT_VALUE", "0")
}

func (v *StrategyConfig) GetMinDiffMemoryLimitAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetMinDiffMemoryLimitValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MinDiffMemoryLimitValue != nil {
		return *v.Containers[containerName].MinDiffMemoryLimitValue
	}
	if v.MinDiffMemoryLimitValue != nil {
		return *v.MinDiffMemoryLimitValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MIN_DIFF_MEMORY_LIMIT_VALUE", "0")
}

func (v *StrategyConfig) GetMemoryRequestFromCpuEnabled(containerName string) bool {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryRequestFromCpuEnabled != nil {
		return *v.Containers[containerName].MemoryRequestFromCpuEnabled
	}
	if v.MemoryRequestFromCpuEnabled != nil {
		return *v.MemoryRequestFromCpuEnabled
	}
	memoryRequestFromCpuEnabled := utils.GetEnv("OBLIK_DEFAULT_MEMORY_REQUEST_FROM_CPU_ENABLED", "")
	return memoryRequestFromCpuEnabled == "true"
}

func (v *StrategyConfig) GetMemoryRequestFromCpuAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetMemoryRequestFromCpuValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryRequestFromCpuValue != nil {
		return *v.Containers[containerName].MemoryRequestFromCpuValue
	}
	if v.MemoryRequestFromCpuValue != nil {
		return *v.MemoryRequestFromCpuValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MEMORY_REQUEST_FROM_CPU_VALUE", "2")
}

func (v *StrategyConfig) GetMemoryLimitFromCpuEnabled(containerName string) bool {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryLimitFromCpuEnabled != nil {
		return *v.Containers[containerName].MemoryLimitFromCpuEnabled
	}
	if v.MemoryLimitFromCpuEnabled != nil {
		return *v.MemoryLimitFromCpuEnabled
	}
	memoryLimitFromCpuEnabled := utils.GetEnv("OBLIK_DEFAULT_MEMORY_LIMIT_FROM_CPU_ENABLED", "")
	return memoryLimitFromCpuEnabled == "true"
}

func (v *StrategyConfig) GetMemoryLimitFromCpuAlgo(containerName string) calculator.CalculatorAlgo {
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

func (v *StrategyConfig) GetMemoryLimitFromCpuValue(containerName string) string {
	if v.Containers[containerName] != nil && v.Containers[containerName].MemoryLimitFromCpuValue != nil {
		return *v.Containers[containerName].MemoryLimitFromCpuValue
	}
	if v.MemoryLimitFromCpuValue != nil {
		return *v.MemoryLimitFromCpuValue
	}
	return utils.GetEnv("OBLIK_DEFAULT_MEMORY_LIMIT_FROM_CPU_VALUE", "2")
}

func (v *StrategyConfig) GetRequestApplyTarget(containerName string) RequestApplyTarget {
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
			return RequestApplyTargetFrugal
		case "target":
			fallthrough
		case "balanced":
			return RequestApplyTargetBalanced
		case "upperBound":
			fallthrough
		case "peak":
			return RequestApplyTargetPeak
		default:
			klog.Warningf("Unknown request-apply-target: %s", requestApplyTarget)
		}
	}
	return RequestApplyTargetBalanced
}

func (v *StrategyConfig) GetRequestCpuApplyTarget(containerName string) RequestApplyTarget {
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
			return RequestApplyTargetFrugal
		case "target":
			fallthrough
		case "balanced":
			return RequestApplyTargetBalanced
		case "upperBound":
			fallthrough
		case "peak":
			return RequestApplyTargetPeak
		default:
			klog.Warningf("Unknown request-apply-target: %s", requestCpuApplyTarget)
		}
	}
	return v.GetRequestApplyTarget(containerName)
}

func (v *StrategyConfig) GetRequestMemoryApplyTarget(containerName string) RequestApplyTarget {
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
			return RequestApplyTargetFrugal
		case "target":
			fallthrough
		case "balanced":
			return RequestApplyTargetBalanced
		case "upperBound":
			fallthrough
		case "peak":
			return RequestApplyTargetPeak
		default:
			klog.Warningf("Unknown request-apply-target: %s", requestMemoryApplyTarget)
		}
	}
	return v.GetRequestApplyTarget(containerName)
}

func (v *StrategyConfig) GetLimitApplyTarget(containerName string) LimitApplyTarget {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitApplyTarget != nil {
		return *v.Containers[containerName].LimitApplyTarget
	}
	if v.LimitApplyTarget != nil {
		return *v.LimitApplyTarget
	}
	limitApplyTarget := utils.GetEnv("OBLIK_DEFAULT_LIMIT_APPLY_TARGET", "")
	if limitApplyTarget != "" {
		switch limitApplyTarget {
		case "auto":
			return LimitApplyTargetAuto
		case "lowerBound":
			fallthrough
		case "frugal":
			return LimitApplyTargetFrugal
		case "target":
			fallthrough
		case "balanced":
			return LimitApplyTargetBalanced
		case "upperBound":
			fallthrough
		case "peak":
			return LimitApplyTargetPeak
		default:
			klog.Warningf("Unknown limit-apply-target: %s", limitApplyTarget)
		}
	}
	return LimitApplyTargetAuto
}

func (v *StrategyConfig) GetLimitCpuApplyTarget(containerName string) LimitApplyTarget {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitCpuApplyTarget != nil {
		return *v.Containers[containerName].LimitCpuApplyTarget
	}
	if v.LimitCpuApplyTarget != nil {
		return *v.LimitCpuApplyTarget
	}
	limitCpuApplyTarget := utils.GetEnv("OBLIK_DEFAULT_LIMIT_CPU_APPLY_TARGET", "")
	if limitCpuApplyTarget != "" {
		switch limitCpuApplyTarget {
		case "auto":
			return LimitApplyTargetAuto
		case "lowerBound":
			fallthrough
		case "frugal":
			return LimitApplyTargetFrugal
		case "target":
			fallthrough
		case "balanced":
			return LimitApplyTargetBalanced
		case "upperBound":
			fallthrough
		case "peak":
			return LimitApplyTargetPeak
		default:
			klog.Warningf("Unknown limit-apply-target: %s", limitCpuApplyTarget)
		}
	}
	return LimitApplyTargetAuto
}

func (v *StrategyConfig) GetLimitMemoryApplyTarget(containerName string) LimitApplyTarget {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitMemoryApplyTarget != nil {
		return *v.Containers[containerName].LimitMemoryApplyTarget
	}
	if v.LimitMemoryApplyTarget != nil {
		return *v.LimitMemoryApplyTarget
	}
	limitMemoryApplyTarget := utils.GetEnv("OBLIK_DEFAULT_LIMIT_MEMORY_APPLY_TARGET", "")
	if limitMemoryApplyTarget != "" {
		switch limitMemoryApplyTarget {
		case "auto":
			return LimitApplyTargetAuto
		case "lowerBound":
			fallthrough
		case "frugal":
			return LimitApplyTargetFrugal
		case "target":
			fallthrough
		case "balanced":
			return LimitApplyTargetBalanced
		case "upperBound":
			fallthrough
		case "peak":
			return LimitApplyTargetPeak
		default:
			klog.Warningf("Unknown limit-apply-target: %s", limitMemoryApplyTarget)
		}
	}
	return LimitApplyTargetAuto
}

func (v *StrategyConfig) GetRequestCpuScaleDirection(containerName string) ScaleDirection {
	if v.Containers[containerName] != nil && v.Containers[containerName].RequestCpuScaleDirection != nil {
		return *v.Containers[containerName].RequestCpuScaleDirection
	}
	if v.RequestCpuScaleDirection != nil {
		return *v.RequestCpuScaleDirection
	}
	requestCpuScaleDirection := utils.GetEnv("OBLIK_DEFAULT_REQUEST_CPU_SCALE_DIRECTION", "")
	if requestCpuScaleDirection != "" {
		switch requestCpuScaleDirection {
		case "both":
			return ScaleDirectionBoth
		case "up":
			return ScaleDirectionUp
		case "down":
			return ScaleDirectionDown
		default:
			klog.Warningf("Unknown scale-direction: %s", requestCpuScaleDirection)
		}
	}
	return ScaleDirectionBoth
}

func (v *StrategyConfig) GetRequestMemoryScaleDirection(containerName string) ScaleDirection {
	if v.Containers[containerName] != nil && v.Containers[containerName].RequestMemoryScaleDirection != nil {
		return *v.Containers[containerName].RequestMemoryScaleDirection
	}
	if v.RequestMemoryScaleDirection != nil {
		return *v.RequestMemoryScaleDirection
	}
	requestMemoryScaleDirection := utils.GetEnv("OBLIK_DEFAULT_REQUEST_MEMORY_SCALE_DIRECTION", "")
	if requestMemoryScaleDirection != "" {
		switch requestMemoryScaleDirection {
		case "both":
			return ScaleDirectionBoth
		case "up":
			return ScaleDirectionUp
		case "down":
			return ScaleDirectionDown
		default:
			klog.Warningf("Unknown scale-direction: %s", requestMemoryScaleDirection)
		}
	}
	return ScaleDirectionBoth
}

func (v *StrategyConfig) GetLimitCpuScaleDirection(containerName string) ScaleDirection {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitCpuScaleDirection != nil {
		return *v.Containers[containerName].LimitCpuScaleDirection
	}
	if v.LimitCpuScaleDirection != nil {
		return *v.LimitCpuScaleDirection
	}
	limitCpuScaleDirection := utils.GetEnv("OBLIK_DEFAULT_LIMIT_CPU_SCALE_DIRECTION", "")
	if limitCpuScaleDirection != "" {
		switch limitCpuScaleDirection {
		case "both":
			return ScaleDirectionBoth
		case "up":
			return ScaleDirectionUp
		case "down":
			return ScaleDirectionDown
		default:
			klog.Warningf("Unknown scale-direction: %s", limitCpuScaleDirection)
		}
	}
	return ScaleDirectionBoth
}

func (v *StrategyConfig) GetLimitMemoryScaleDirection(containerName string) ScaleDirection {
	if v.Containers[containerName] != nil && v.Containers[containerName].LimitMemoryScaleDirection != nil {
		return *v.Containers[containerName].LimitMemoryScaleDirection
	}
	if v.LimitMemoryScaleDirection != nil {
		return *v.LimitMemoryScaleDirection
	}
	limitMemoryScaleDirection := utils.GetEnv("OBLIK_DEFAULT_LIMIT_MEMORY_SCALE_DIRECTION", "")
	if limitMemoryScaleDirection != "" {
		switch limitMemoryScaleDirection {
		case "both":
			return ScaleDirectionBoth
		case "up":
			return ScaleDirectionUp
		case "down":
			return ScaleDirectionDown
		default:
			klog.Warningf("Unknown scale-direction: %s", limitMemoryScaleDirection)
		}
	}
	return ScaleDirectionBoth
}
