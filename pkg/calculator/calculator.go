package calculator

import (
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

type CalculatorAlgo int

const (
	CalculatorAlgoRatio CalculatorAlgo = iota
	CalculatorAlgoMargin
)

type ResourceType int

const (
	ResourceTypeCPU ResourceType = iota
	ResourceTypeMemory
)

func CalculateResourceValue(currentValue resource.Quantity, algo CalculatorAlgo, valueStr string, resourceType ResourceType) resource.Quantity {
	// klog.Infof("CalculateResourceValue: Start - currentValue: %v, algo: %v, valueStr: %s, resourceType: %v", currentValue, algo, valueStr, resourceType)

	if valueStr == "" {
		// klog.Infof("CalculateResourceValue: valueStr is empty, returning currentValue: %v", currentValue)
		return currentValue
	}

	newValue := currentValue.DeepCopy()
	switch algo {
	case CalculatorAlgoRatio:
		// klog.Infof("CalculateResourceValue: Using CalculatorAlgoRatio")
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			klog.Warningf("Error parsing calculator ratio value: %s", err.Error())
			// klog.Infof("CalculateResourceValue: Error parsing ratio, returning currentValue: %v", currentValue)
			return currentValue
		}
		// klog.Infof("CalculateResourceValue: Parsed ratio value: %v", value)

		switch resourceType {
		case ResourceTypeCPU:
			// klog.Infof("CalculateResourceValue: Calculating for CPU")
			currentMilliValue := currentValue.MilliValue()
			newMilliValue := int64(float64(currentMilliValue) * value)
			newValue = *resource.NewMilliQuantity(newMilliValue, resource.DecimalSI)
			// klog.Infof("CalculateResourceValue: New CPU value: %v", newValue)
		case ResourceTypeMemory:
			// klog.Infof("CalculateResourceValue: Calculating for Memory")
			newValue = *resource.NewQuantity(int64(float64(currentValue.Value())*value), resource.BinarySI)
			// klog.Infof("CalculateResourceValue: New Memory value: %v", newValue)
		}
	case CalculatorAlgoMargin:
		// klog.Infof("CalculateResourceValue: Using CalculatorAlgoMargin")
		// klog.Infof("CalculateResourceValue Original: Parsed margin value: %v", newValue)
		value, err := resource.ParseQuantity(valueStr)
		if err != nil {
			klog.Warningf("Error parsing calculator margin value: %s", err.Error())
			// klog.Infof("CalculateResourceValue: Error parsing margin, returning currentValue: %v", currentValue)
			return currentValue
		}
		// klog.Infof("CalculateResourceValue: Parsed margin value: %v", value)
		newValue.Add(value)
		// klog.Infof("CalculateResourceValue: New value after adding margin: %v", newValue)
	}

	// klog.Infof("CalculateResourceValue: End - returning newValue: %v", newValue)
	return newValue
}

func CalculateCpuToMemory(cpu resource.Quantity) resource.Quantity {
	// klog.Infof("CalculateCpuToMemory: Start - cpu: %v", cpu)

	// Convert CPU to milli-units to ensure proper calculation
	cpuMilliValue := cpu.MilliValue()
	// klog.Infof("CalculateCpuToMemory: CPU in milli-units: %d", cpuMilliValue)

	// Convert CPU to bytes (1 CPU = 1 GB)
	cpuToMemoryBytes := cpuMilliValue * 1_000_000 // 1 milliCPU = 1 MB = 1_000_000 bytes
	// klog.Infof("CalculateCpuToMemory: Converted CPU to memory bytes: %d", cpuToMemoryBytes)

	// Create a new resource.Quantity representing the memory
	totalMemory := *resource.NewQuantity(cpuToMemoryBytes, resource.BinarySI)
	// klog.Infof("CalculateCpuToMemory: End - returning totalMemory: %v", totalMemory)

	return totalMemory
}
