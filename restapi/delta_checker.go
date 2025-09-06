package restapi

import (
	"reflect"
	"strings"
)

/*
 * Performs a deep comparison of two maps - the resource as recorded in state, and the resource as returned by the API.
 * Accepts a third argument that is a set of fields that are to be ignored when looking for differences.
 * Returns 1. the recordedResource overlaid with fields that have been modified in actualResource but not ignored, and 2. a bool true if there were any changes.
 */
func getDelta(recordedResource map[string]interface{}, actualResource map[string]interface{}, ignoreList []string) (modifiedResource map[string]interface{}, hasChanges bool) {
	modifiedResource = map[string]interface{}{}
	hasChanges = false

	// Keep track of keys we've already checked in actualResource to reduce work when checking keys in actualResource
	checkedKeys := map[string]struct{}{}

	for key, valRecorded := range recordedResource {

		checkedKeys[key] = struct{}{}

		// If the ignore_list contains the current key, don't compare
		if contains(ignoreList, key) {
			modifiedResource[key] = valRecorded
			continue
		}

		valActual, actualHasKey := actualResource[key]

		if valRecorded == nil {
			// A JSON null was put in input data, confirm the result is either not set or is also null
			modifiedResource[key] = valActual
			if actualHasKey && valActual != nil {
				hasChanges = true
			}
		} else if reflect.TypeOf(valRecorded).Kind() == reflect.Map {
			// If valRecorded was a map, assert both values are maps
			subMapA, okA := valRecorded.(map[string]interface{})
			subMapB, okB := valActual.(map[string]interface{})
			if !okA || !okB {
				modifiedResource[key] = valActual
				hasChanges = true
				continue
			}
			// Recursively compare
			deeperIgnoreList := _descendIgnoreList(key, ignoreList)
			if modifiedSubResource, hasChange := getDelta(subMapA, subMapB, deeperIgnoreList); hasChange {
				modifiedResource[key] = modifiedSubResource
				hasChanges = true
			} else {
				modifiedResource[key] = valRecorded
			}
		} else if reflect.TypeOf(valRecorded).Kind() == reflect.Slice {
			// Use reflection to handle slice comparison with ignoreList support
			modifiedSlice, sliceHasChanges := compareSlicesWithIgnoreListReflection(valRecorded, valActual, key, ignoreList)
			if sliceHasChanges {
				modifiedResource[key] = modifiedSlice
				hasChanges = true
			} else {
				modifiedResource[key] = valRecorded
			}
		} else if valRecorded != valActual {
			modifiedResource[key] = valActual
			hasChanges = true
		} else {
			// In this case, the recorded and actual values were the same
			modifiedResource[key] = valRecorded
		}

	}

	for key, valActual := range actualResource {
		// We may have already compared this key with recordedResource
		_, alreadyChecked := checkedKeys[key]
		if alreadyChecked {
			continue
		}

		// If the ignore_list contains the current key, don't compare.
		// Don't modify modifiedResource either - we don't want this key to be tracked
		if contains(ignoreList, key) {
			continue
		}

		// If we've gotten here, that means actualResource has an additional key that wasn't in recordedResource
		modifiedResource[key] = valActual
		hasChanges = true
	}

	return modifiedResource, hasChanges
}

/*
 * Compares two slices with ignoreList support for map elements within the slice using reflection.
 * Returns the modified slice and a boolean indicating if there were changes.
 */
func compareSlicesWithIgnoreListReflection(recordedResource interface{}, actualResource interface{}, key string, ignoreList []string) (interface{}, bool) {
	recordedValue := reflect.ValueOf(recordedResource)
	actualValue := reflect.ValueOf(actualResource)

	// Verify both are slices
	if recordedValue.Kind() != reflect.Slice || actualValue.Kind() != reflect.Slice {
		// Fallback to deep comparison if not both slices
		return actualResource, !reflect.DeepEqual(recordedResource, actualResource)
	}

	// If slices have different lengths, that's always a change
	if recordedValue.Len() != actualValue.Len() {
		return actualResource, true
	}

	hasChanges := false
	deeperIgnoreList := _descendIgnoreList(key, ignoreList)

	// Create new slice with same type as recorded
	modifiedSlice := reflect.MakeSlice(recordedValue.Type(), recordedValue.Len(), recordedValue.Len())

	for i := 0; i < recordedValue.Len(); i++ {
		recordedElement := recordedValue.Index(i).Interface()
		actualElement := actualValue.Index(i).Interface()

		// Check if both elements are maps
		mapRecorded, okRecorded := recordedElement.(map[string]interface{})
		mapActual, okActual := actualElement.(map[string]interface{})

		if okRecorded && okActual {
			// Both elements are maps, use getDelta recursively
			modifiedElement, elementHasChanges := getDelta(mapRecorded, mapActual, deeperIgnoreList)
			if elementHasChanges {
				modifiedSlice.Index(i).Set(reflect.ValueOf(modifiedElement))
				hasChanges = true
			} else {
				modifiedSlice.Index(i).Set(reflect.ValueOf(recordedElement))
			}
		} else {
			// At least one element is not a map, use simple comparison
			if !reflect.DeepEqual(recordedElement, actualElement) {
				modifiedSlice.Index(i).Set(reflect.ValueOf(actualElement))
				hasChanges = true
			} else {
				modifiedSlice.Index(i).Set(reflect.ValueOf(recordedElement))
			}
		}
	}

	return modifiedSlice.Interface(), hasChanges
}

/*
 * Modifies an ignoreList to be relative to a descended path.
 * E.g. given descendPath = "bar", and the ignoreList [foo, bar.alpha, bar.bravo], this returns [alpha, bravo]
 */
func _descendIgnoreList(descendPath string, ignoreList []string) []string {
	var newIgnoreList []string

	for _, ignorePath := range ignoreList {
		pathComponents := strings.Split(ignorePath, ".")
		// If this ignorePath starts with the descendPath, remove the first component and keep the rest
		if pathComponents[0] == descendPath {
			modifiedPath := strings.Join(pathComponents[1:], ".")
			newIgnoreList = append(newIgnoreList, modifiedPath)
		}
	}

	return newIgnoreList
}

func contains(list []string, elem string) bool {
	for _, a := range list {
		if a == elem {
			return true
		}
	}
	return false
}
