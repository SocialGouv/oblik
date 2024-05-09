package controller

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const MiB = 1024 * 1024
const FieldManager = "oblik"

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

// RemoveNullValues recursively removes null values from an unstructured object.
func RemoveNullValues(u *unstructured.Unstructured) {
	removeNullsFromMap(u.Object)
}

// removeNullsFromMap recursively removes null values from a map.
func removeNullsFromMap(m map[string]interface{}) {
	for k, v := range m {
		if v == nil {
			delete(m, k)
		} else {
			switch typedV := v.(type) {
			case map[string]interface{}:
				removeNullsFromMap(typedV)
				if len(typedV) == 0 { // If the map is empty after removals, delete it
					delete(m, k)
				}
			case []interface{}:
				cleanSlice := removeNullsFromSlice(typedV)
				if len(cleanSlice) == 0 { // If the slice is empty after removals, delete it
					delete(m, k)
				} else {
					m[k] = cleanSlice
				}
			}
		}
	}
}

// removeNullsFromSlice recursively removes null values from a slice.
func removeNullsFromSlice(s []interface{}) []interface{} {
	cleanSlice := make([]interface{}, 0, len(s))
	for _, v := range s {
		if v != nil {
			switch typedV := v.(type) {
			case map[string]interface{}:
				removeNullsFromMap(typedV)
				if len(typedV) > 0 { // Only add non-empty maps
					cleanSlice = append(cleanSlice, typedV)
				}
			default:
				cleanSlice = append(cleanSlice, v)
			}
		}
	}
	return cleanSlice
}

func flattenAndClean(u *unstructured.Unstructured) error {
	jsonData, err := json.Marshal(u.Object)
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonData, &u.Object)
	if err != nil {
		return err
	}

	RemoveNullValues(u)

	return nil
}
