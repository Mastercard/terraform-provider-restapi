package provider

import (
	"reflect"
	"strings"
)

// getDelta performs a deep comparison of two maps - the resource as recorded in state, and the resource as returned by the API.
// Accepts a third argument that is a set of fields that are to be ignored when looking for differences.
// Accepts a fourth argument ignoreServerAdditions - when true, fields added by the server (but not in recorded) will be ignored.
// Returns 1. the recordedResource overlaid with fields that have been modified in actualResource but not ignored, and 2. a bool true if there were any changes.
func getDelta(recorded map[string]interface{}, actual map[string]interface{}, ignoreList []string, ignoreServerAdditions bool) (modifiedResource map[string]interface{}, hasChanges bool) {
	modifiedResource = map[string]interface{}{}
	hasChanges = false

	// Keep track of keys we've already checked in actualResource to reduce work when checking keys in actualResource
	checkedKeys := map[string]struct{}{}

	for key, valRecorded := range recorded {

		checkedKeys[key] = struct{}{}

		// If the ignore_list contains the current key, don't compare
		if contains(ignoreList, key) {
			modifiedResource[key] = valRecorded
			continue
		}

		valActual, actualHasKey := actual[key]

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
			if modifiedSubResource, hasChange := getDelta(subMapA, subMapB, deeperIgnoreList, ignoreServerAdditions); hasChange {
				modifiedResource[key] = modifiedSubResource
				hasChanges = true
			} else {
				modifiedResource[key] = valRecorded
			}
		} else if reflect.TypeOf(valRecorded).Kind() == reflect.Slice {
			// Since we don't support ignoring differences in lists (besides ignoring the list as a
			// whole), it is safe to deep compare the two list values.
			if !reflect.DeepEqual(valRecorded, valActual) {
				modifiedResource[key] = valActual
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

	for key, valActual := range actual {
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
		// When ignoreServerAdditions is true, we don't consider server-added fields as changes
		if ignoreServerAdditions {
			continue
		}
		modifiedResource[key] = valActual
		hasChanges = true
	}

	return modifiedResource, hasChanges
}

// _descendIgnoreList modifies an ignoreList to be relative to a descended path.
// E.g. given descendPath = "bar", and the ignoreList [foo, bar.alpha, bar.bravo], this returns [alpha, bravo]
func _descendIgnoreList(descendPath string, ignoreList []string) []string {
	newIgnoreList := make([]string, len(ignoreList))

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
