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

func calculateNewLimitValue(currentValue resource.Quantity, algo CalculatorAlgo, valueStr string) resource.Quantity {
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		klog.Warningf("Error parsing calculator value: %s", err.Error())
		return currentValue
	}

	newValue := currentValue.DeepCopy()
	switch algo {
	case CalculatorAlgoRatio:
		if currentValue.Format == resource.DecimalSI { // Handles CPU
			currentMilliValue := currentValue.MilliValue()
			newMilliValue := int64(float64(currentMilliValue) * value)
			newValue = *resource.NewMilliQuantity(newMilliValue, currentValue.Format)
		} else { // Handles memory
			newValue = *resource.NewQuantity(int64(float64(currentValue.Value())*value), currentValue.Format)
		}
	case CalculatorAlgoMargin:
		newValue.Add(resource.MustParse(fmt.Sprintf("%.0fm", value*1000)))
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
