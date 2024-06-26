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

func CalculateResourceValue(currentValue resource.Quantity, algo CalculatorAlgo, valueStr string) resource.Quantity {
	if valueStr == "" {
		return currentValue
	}
	newValue := currentValue.DeepCopy()
	switch algo {
	case CalculatorAlgoRatio:
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			klog.Warningf("Error parsing calculator ratio value: %s", err.Error())
			return currentValue
		}
		if currentValue.Format == resource.DecimalSI { // Handles CPU
			currentMilliValue := currentValue.MilliValue()
			newMilliValue := int64(float64(currentMilliValue) * value)
			newValue = *resource.NewMilliQuantity(newMilliValue, currentValue.Format)
		} else { // Handles memory
			newValue = *resource.NewQuantity(int64(float64(currentValue.Value())*value), currentValue.Format)
		}
	case CalculatorAlgoMargin:
		value, err := resource.ParseQuantity(valueStr)
		if err != nil {
			klog.Warningf("Error parsing calculator margin value: %s", err.Error())
			return currentValue
		}
		newValue.Add(value)
	}

	return newValue
}

func CalculateCpuToMemory(cpu resource.Quantity) resource.Quantity {
	// Convert CPU to milli-units to ensure proper calculation
	cpuMilliValue := cpu.MilliValue()
	// Convert CPU to bytes (1 CPU = 1 GB)
	cpuToMemoryBytes := cpuMilliValue * 1_000_000 // 1 milliCPU = 1 MB = 1_000_000 bytes
	// Create a new resource.Quantity representing the memory
	totalMemory := *resource.NewQuantity(cpuToMemoryBytes, resource.BinarySI)
	return totalMemory
}
