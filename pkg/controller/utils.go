package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

const MiB = 1024 * 1024

func parseQuantity(qtyStr string) (resource.Quantity, error) {
	qty, err := resource.ParseQuantity(qtyStr)
	if err != nil {
		return resource.Quantity{}, fmt.Errorf("failed to parse quantity %s: %v", qtyStr, err)
	}
	return qty, nil
}

func exceedsThreshold(recommendation, cursor string) bool {
	recQty, err := parseQuantity(recommendation)
	if err != nil {
		fmt.Println(err)
		return false
	}

	curQty, err := parseQuantity(cursor)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return recQty.Cmp(curQty) > 0
}

func subtractBaseResource(recommendation, base string) string {
	recQty, _ := resource.ParseQuantity(recommendation)
	baseQty, _ := resource.ParseQuantity(base)
	recQty.Sub(baseQty)
	return recQty.String()
}

func multiplyResource(resourceQty string, multiplier int) string {
	qty, _ := resource.ParseQuantity(resourceQty)
	qty.SetMilli(qty.MilliValue() * int64(multiplier))
	return qty.String()
}

func adjustResource(recommendation, baseResource string, ratio float64) string {
	recQty, _ := resource.ParseQuantity(recommendation)
	baseQty, _ := resource.ParseQuantity(baseResource)

	// Subtract base
	recQty.Sub(baseQty)

	// Multiply by ratio
	scaledValue := float64(recQty.ScaledValue(resource.Milli)) * ratio
	scaledQty := resource.NewMilliQuantity(int64(scaledValue), resource.DecimalSI)

	// Add base back
	scaledQty.Add(baseQty)

	return scaledQty.String()
}
