package utils

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

func ParseDuration(durationStr string, defaultDuration time.Duration) time.Duration {
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

func FormatMemory(quantity resource.Quantity) string {
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
