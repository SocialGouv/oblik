package controller

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func calculateNewResourceValue(currentValue resource.Quantity, algo CalculatorAlgo, valueStr string) resource.Quantity {
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

func parseDuration(durationStr string, defaultDuration time.Duration) time.Duration {
	if durationStr == "" {
		return defaultDuration
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		klog.Warningf("Error parsing duration: %s, using default: %s", err.Error(), defaultDuration)
		return defaultDuration
	}
	return duration
}

func formatMemory(quantity resource.Quantity) string {
	bytes := quantity.Value()
	kib := float64(bytes) / 1024
	mib := kib / 1024
	gib := mib / 1024
	tib := gib / 1024

	switch {
	case tib >= 1:
		return fmt.Sprintf("%.2f TiB", tib)
	case gib >= 1:
		return fmt.Sprintf("%.2f GiB", gib)
	case mib >= 1:
		return fmt.Sprintf("%.2f MiB", mib)
	case kib >= 1:
		return fmt.Sprintf("%.2f KiB", kib)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
