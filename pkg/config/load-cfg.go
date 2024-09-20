package config

import (
	"github.com/SocialGouv/oblik/pkg/calculator"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

type LoadCfg struct {
	Key string

	RequestCPUApplyMode    *ApplyMode
	RequestMemoryApplyMode *ApplyMode
	LimitCPUApplyMode      *ApplyMode
	LimitMemoryApplyMode   *ApplyMode

	LimitCPUCalculatorAlgo     *calculator.CalculatorAlgo
	LimitMemoryCalculatorAlgo  *calculator.CalculatorAlgo
	LimitMemoryCalculatorValue *string
	LimitCPUCalculatorValue    *string

	UnprovidedApplyDefaultRequestCPUSource    *UnprovidedApplyDefaultMode
	UnprovidedApplyDefaultRequestCPUValue     *string
	UnprovidedApplyDefaultRequestMemorySource *UnprovidedApplyDefaultMode
	UnprovidedApplyDefaultRequestMemoryValue  *string

	IncreaseRequestCpuAlgo     *calculator.CalculatorAlgo
	IncreaseRequestMemoryAlgo  *calculator.CalculatorAlgo
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

	MinAllowedRecommendationCpu    *resource.Quantity
	MaxAllowedRecommendationCpu    *resource.Quantity
	MinAllowedRecommendationMemory *resource.Quantity
	MaxAllowedRecommendationMemory *resource.Quantity

	MinDiffCpuRequestAlgo     *calculator.CalculatorAlgo
	MinDiffCpuRequestValue    *string
	MinDiffMemoryRequestAlgo  *calculator.CalculatorAlgo
	MinDiffMemoryRequestValue *string
	MinDiffCpuLimitAlgo       *calculator.CalculatorAlgo
	MinDiffCpuLimitValue      *string
	MinDiffMemoryLimitAlgo    *calculator.CalculatorAlgo
	MinDiffMemoryLimitValue   *string

	MemoryRequestFromCpuEnabled *bool
	MemoryRequestFromCpuAlgo    *calculator.CalculatorAlgo
	MemoryRequestFromCpuValue   *string
	MemoryLimitFromCpuEnabled   *bool
	MemoryLimitFromCpuAlgo      *calculator.CalculatorAlgo
	MemoryLimitFromCpuValue     *string

	RequestApplyTarget       *RequestApplyTarget
	RequestCpuApplyTarget    *RequestApplyTarget
	RequestMemoryApplyTarget *RequestApplyTarget

	LimitApplyTarget       *LimitApplyTarget
	LimitCpuApplyTarget    *LimitApplyTarget
	LimitMemoryApplyTarget *LimitApplyTarget

	RequestCpuScaleDirection    *ScaleDirection
	RequestMemoryScaleDirection *ScaleDirection
	LimitCpuScaleDirection      *ScaleDirection
	LimitMemoryScaleDirection   *ScaleDirection
}

func loadAnnotableCommonCfg(cfg *LoadCfg, annotable Annotable, annotationSuffix string) {

	annotations := getAnnotations(annotable)

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
	if limitCPUCalculatorAlgo != "" {
		switch limitCPUCalculatorAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.LimitCPUCalculatorAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.LimitCPUCalculatorAlgo = &algo
		default:
			klog.Warningf("Unknown calculator algorithm: %s", limitCPUCalculatorAlgo)
		}
	}

	limitMemoryCalculatorAlgo := getAnnotation("limit-memory-calculator-algo")
	if limitMemoryCalculatorAlgo != "" {
		switch limitMemoryCalculatorAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.LimitMemoryCalculatorAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.LimitMemoryCalculatorAlgo = &algo
		default:
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
	if increaseRequestCpuAlgo != "" {
		switch increaseRequestCpuAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.IncreaseRequestCpuAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.IncreaseRequestCpuAlgo = &algo
		default:
			klog.Warningf("Unknown calculator algorithm: %s", increaseRequestCpuAlgo)
		}
	}

	increaseRequestMemoryAlgo := getAnnotation("increase-request-memory-algo")
	if increaseRequestMemoryAlgo != "" {
		switch increaseRequestMemoryAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.IncreaseRequestMemoryAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.IncreaseRequestMemoryAlgo = &algo
		default:
			klog.Warningf("Unknown calculator algorithm: %s", increaseRequestMemoryAlgo)
		}
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

	minDiffCpuRequestAlgo := getAnnotation("min-diff-cpu-request-algo")
	if minDiffCpuRequestAlgo != "" {
		switch minDiffCpuRequestAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.MinDiffCpuRequestAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.MinDiffCpuRequestAlgo = &algo
		default:
			klog.Warningf("Unknown calculator algorithm: %s", minDiffCpuRequestAlgo)
		}
	}

	minDiffMemoryRequestAlgo := getAnnotation("min-diff-memory-request-algo")
	if minDiffMemoryRequestAlgo != "" {
		switch minDiffMemoryRequestAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.MinDiffMemoryRequestAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.MinDiffMemoryRequestAlgo = &algo
		default:
			klog.Warningf("Unknown calculator algorithm: %s", minDiffMemoryRequestAlgo)
		}
	}

	minDiffCpuRequestValue := getAnnotation("min-diff-cpu-request-value")
	minDiffMemoryRequestValue := getAnnotation("min-diff-memory-request-value")
	if minDiffCpuRequestValue != "" {
		cfg.MinDiffCpuRequestValue = &minDiffCpuRequestValue
	}
	if minDiffMemoryRequestValue != "" {
		cfg.MinDiffMemoryRequestValue = &minDiffMemoryRequestValue
	}

	minDiffCpuLimitAlgo := getAnnotation("min-diff-cpu-limit-algo")
	if minDiffCpuLimitAlgo != "" {
		switch minDiffCpuLimitAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.MinDiffCpuLimitAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.MinDiffCpuLimitAlgo = &algo
		default:
			klog.Warningf("Unknown calculator algorithm: %s", minDiffCpuLimitAlgo)
		}
	}

	minDiffMemoryLimitAlgo := getAnnotation("min-diff-memory-limit-algo")
	if minDiffMemoryLimitAlgo != "" {
		switch minDiffMemoryLimitAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.MinDiffMemoryLimitAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.MinDiffMemoryLimitAlgo = &algo
		default:
			klog.Warningf("Unknown calculator algorithm: %s", minDiffMemoryLimitAlgo)
		}
	}

	minDiffCpuLimitValue := getAnnotation("min-diff-cpu-limit-value")
	minDiffMemoryLimitValue := getAnnotation("min-diff-memory-limit-value")
	if minDiffCpuLimitValue != "" {
		cfg.MinDiffCpuLimitValue = &minDiffCpuLimitValue
	}
	if minDiffMemoryLimitValue != "" {
		cfg.MinDiffMemoryLimitValue = &minDiffMemoryLimitValue
	}

	memoryRequestFromCpuEnabled := getAnnotation("memory-request-from-cpu-enabled")
	if memoryRequestFromCpuEnabled == "true" {
		memoryRequestFromCpuEnabledBool := true
		cfg.MemoryRequestFromCpuEnabled = &memoryRequestFromCpuEnabledBool
	}
	memoryRequestFromCpuAlgo := getAnnotation("memory-request-from-cpu-algo")
	if memoryRequestFromCpuAlgo != "" {
		switch memoryRequestFromCpuAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.MemoryRequestFromCpuAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.MemoryRequestFromCpuAlgo = &algo
		default:
			klog.Warningf("Unknown calculator algorithm: %s", memoryRequestFromCpuAlgo)
		}
	}
	memoryRequestFromCpuValue := getAnnotation("memory-request-from-cpu-value")
	if memoryRequestFromCpuValue != "" {
		cfg.MemoryRequestFromCpuValue = &memoryRequestFromCpuValue
	}

	memoryLimitFromCpuEnabled := getAnnotation("memory-limit-from-cpu-enabled")
	if memoryLimitFromCpuEnabled == "true" {
		memoryLimitFromCpuEnabledBool := true
		cfg.MemoryLimitFromCpuEnabled = &memoryLimitFromCpuEnabledBool
	}
	memoryLimitFromCpuAlgo := getAnnotation("memory-limit-from-cpu-algo")
	if memoryLimitFromCpuAlgo != "" {
		switch memoryLimitFromCpuAlgo {
		case "ratio":
			algo := calculator.CalculatorAlgoRatio
			cfg.MemoryLimitFromCpuAlgo = &algo
		case "margin":
			algo := calculator.CalculatorAlgoMargin
			cfg.MemoryLimitFromCpuAlgo = &algo
		default:
			klog.Warningf("Unknown calculator algorithm: %s", memoryLimitFromCpuAlgo)
		}
	}
	memoryLimitFromCpuValue := getAnnotation("memory-limit-from-cpu-value")
	if memoryLimitFromCpuValue != "" {
		cfg.MemoryLimitFromCpuValue = &memoryLimitFromCpuValue
	}

	requestApplyTarget := getAnnotation("request-apply-target")
	if requestApplyTarget != "" {
		switch requestApplyTarget {
		case "lowerBound":
			fallthrough
		case "frugal":
			applyTarget := RequestApplyTargetFrugal
			cfg.RequestApplyTarget = &applyTarget
		case "target":
			fallthrough
		case "balanced":
			applyTarget := RequestApplyTargetBalanced
			cfg.RequestApplyTarget = &applyTarget
		case "upperBound":
			fallthrough
		case "peak":
			applyTarget := RequestApplyTargetPeak
			cfg.RequestApplyTarget = &applyTarget
		default:
			klog.Warningf("Unknown request-apply-target: %s", requestApplyTarget)
		}
	}

	requestCpuApplyTarget := getAnnotation("request-cpu-apply-target")
	if requestCpuApplyTarget != "" {
		switch requestCpuApplyTarget {
		case "lowerBound":
			fallthrough
		case "frugal":
			applyTarget := RequestApplyTargetFrugal
			cfg.RequestCpuApplyTarget = &applyTarget
		case "target":
			fallthrough
		case "balanced":
			applyTarget := RequestApplyTargetBalanced
			cfg.RequestCpuApplyTarget = &applyTarget
		case "upperBound":
			fallthrough
		case "peak":
			applyTarget := RequestApplyTargetPeak
			cfg.RequestCpuApplyTarget = &applyTarget
		default:
			klog.Warningf("Unknown request-apply-target: %s", requestCpuApplyTarget)
		}
	}

	requestMemoryApplyTarget := getAnnotation("request-memory-apply-target")
	if requestMemoryApplyTarget != "" {
		switch requestMemoryApplyTarget {
		case "lowerBound":
			fallthrough
		case "frugal":
			applyTarget := RequestApplyTargetFrugal
			cfg.RequestMemoryApplyTarget = &applyTarget
		case "target":
			fallthrough
		case "balanced":
			applyTarget := RequestApplyTargetBalanced
			cfg.RequestMemoryApplyTarget = &applyTarget
		case "upperBound":
			fallthrough
		case "peak":
			applyTarget := RequestApplyTargetPeak
			cfg.RequestMemoryApplyTarget = &applyTarget
		default:
			klog.Warningf("Unknown request-apply-target: %s", requestMemoryApplyTarget)
		}
	}

	limitApplyTarget := getAnnotation("limit-apply-target")
	if limitApplyTarget != "" {
		switch limitApplyTarget {
		case "lowerBound":
			fallthrough
		case "frugal":
			applyTarget := LimitApplyTargetFrugal
			cfg.LimitApplyTarget = &applyTarget
		case "target":
			fallthrough
		case "balanced":
			applyTarget := LimitApplyTargetBalanced
			cfg.LimitApplyTarget = &applyTarget
		case "upperBound":
			fallthrough
		case "peak":
			applyTarget := LimitApplyTargetPeak
			cfg.LimitApplyTarget = &applyTarget
		default:
			klog.Warningf("Unknown apply-target: %s", limitApplyTarget)
		}
	}

	limitCpuApplyTarget := getAnnotation("limit-cpu-apply-target")
	if limitCpuApplyTarget != "" {
		switch limitCpuApplyTarget {
		case "lowerBound":
			fallthrough
		case "frugal":
			applyTarget := LimitApplyTargetFrugal
			cfg.LimitCpuApplyTarget = &applyTarget
		case "target":
			fallthrough
		case "balanced":
			applyTarget := LimitApplyTargetBalanced
			cfg.LimitCpuApplyTarget = &applyTarget
		case "upperBound":
			fallthrough
		case "peak":
			applyTarget := LimitApplyTargetPeak
			cfg.LimitCpuApplyTarget = &applyTarget
		default:
			klog.Warningf("Unknown apply-target: %s", limitCpuApplyTarget)
		}
	}

	limitMemoryApplyTarget := getAnnotation("limit-memory-apply-target")
	if limitMemoryApplyTarget != "" {
		switch limitMemoryApplyTarget {
		case "lowerBound":
			fallthrough
		case "frugal":
			applyTarget := LimitApplyTargetFrugal
			cfg.LimitMemoryApplyTarget = &applyTarget
		case "target":
			fallthrough
		case "balanced":
			applyTarget := LimitApplyTargetBalanced
			cfg.LimitMemoryApplyTarget = &applyTarget
		case "upperBound":
			fallthrough
		case "peak":
			applyTarget := LimitApplyTargetPeak
			cfg.LimitMemoryApplyTarget = &applyTarget
		default:
			klog.Warningf("Unknown apply-target: %s", limitMemoryApplyTarget)
		}
	}

	requestCpuScaleDirection := getAnnotation("request-cpu-scale-direction")
	if requestCpuScaleDirection != "" {
		switch requestCpuScaleDirection {
		case "both":
			scaleDirection := ScaleDirectionBoth
			cfg.RequestCpuScaleDirection = &scaleDirection
		case "up":
			scaleDirection := ScaleDirectionUp
			cfg.RequestCpuScaleDirection = &scaleDirection
		case "down":
			scaleDirection := ScaleDirectionDown
			cfg.RequestCpuScaleDirection = &scaleDirection
		default:
			klog.Warningf("Unknown scale-direction: %s", requestCpuScaleDirection)
		}
	}

	requestMemoryScaleDirection := getAnnotation("request-memory-scale-direction")
	if requestMemoryScaleDirection != "" {
		switch requestMemoryScaleDirection {
		case "both":
			scaleDirection := ScaleDirectionBoth
			cfg.RequestMemoryScaleDirection = &scaleDirection
		case "up":
			scaleDirection := ScaleDirectionUp
			cfg.RequestMemoryScaleDirection = &scaleDirection
		case "down":
			scaleDirection := ScaleDirectionDown
			cfg.RequestMemoryScaleDirection = &scaleDirection
		default:
			klog.Warningf("Unknown scale-direction: %s", requestMemoryScaleDirection)
		}
	}

	limitCpuScaleDirection := getAnnotation("limit-cpu-scale-direction")
	if limitCpuScaleDirection != "" {
		switch limitCpuScaleDirection {
		case "both":
			scaleDirection := ScaleDirectionBoth
			cfg.LimitCpuScaleDirection = &scaleDirection
		case "up":
			scaleDirection := ScaleDirectionUp
			cfg.LimitCpuScaleDirection = &scaleDirection
		case "down":
			scaleDirection := ScaleDirectionDown
			cfg.LimitCpuScaleDirection = &scaleDirection
		default:
			klog.Warningf("Unknown scale-direction: %s", limitCpuScaleDirection)
		}
	}

	limitMemoryScaleDirection := getAnnotation("limit-memory-scale-direction")
	if limitMemoryScaleDirection != "" {
		switch limitMemoryScaleDirection {
		case "both":
			scaleDirection := ScaleDirectionBoth
			cfg.LimitMemoryScaleDirection = &scaleDirection
		case "up":
			scaleDirection := ScaleDirectionUp
			cfg.LimitMemoryScaleDirection = &scaleDirection
		case "down":
			scaleDirection := ScaleDirectionDown
			cfg.LimitMemoryScaleDirection = &scaleDirection
		default:
			klog.Warningf("Unknown scale-direction: %s", limitMemoryScaleDirection)
		}
	}
}
