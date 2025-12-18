package restapi

import (
	"reflect"
	"strings"
)

/*
 * Performs a deep comparison of two maps - the resource as recorded in state, and the resource as returned by the API.
 * Accepts a third argument that is a set of fields that are to be ignored when looking for differences.
 * Returns 1. the recordedResource overlaid with fields that have been modified in actualResource but not ignored, and 2. a bool true if there were any changes.
 *
 * Ignore list syntax:
 *   - "field" - ignore a top-level field
 *   - "parent.child" - ignore a nested field
 *   - "list[].field" - ignore "field" in all items of "list" (when list contains objects)
 *   - "@odata.etag" - keys containing dots are matched exactly first, before trying path descent
 */
func getDelta(recordedResource map[string]interface{}, actualResource map[string]interface{}, ignoreList []string) (modifiedResource map[string]interface{}, hasChanges bool) {
	modifiedResource = map[string]interface{}{}
	hasChanges = false

	// Keep track of keys we've already checked in actualResource to reduce work when checking keys in actualResource
	checkedKeys := map[string]struct{}{}

	for key, valRecorded := range recordedResource {

		checkedKeys[key] = struct{}{}

		// If the ignore_list contains the current key, don't compare
		// Try exact match first (handles keys containing dots like @odata.etag)
		if containsExact(ignoreList, key) {
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
			// Check if any ignore paths apply to this list using the [] syntax
			listIgnoreList := _getListIgnoreList(key, ignoreList)

			if len(listIgnoreList) > 0 {
				// We have ignore paths that apply to list items, compare element by element
				modifiedList, listHasChanges := _compareLists(valRecorded, valActual, listIgnoreList)
				if listHasChanges {
					modifiedResource[key] = modifiedList
					hasChanges = true
				} else {
					modifiedResource[key] = valRecorded
				}
			} else {
				// No list-specific ignores, do deep compare as before
				if !reflect.DeepEqual(valRecorded, valActual) {
					modifiedResource[key] = valActual
					hasChanges = true
				} else {
					modifiedResource[key] = valRecorded
				}
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
		// Try exact match first (handles keys containing dots like @odata.etag)
		if containsExact(ignoreList, key) {
			continue
		}

		// If we've gotten here, that means actualResource has an additional key that wasn't in recordedResource
		modifiedResource[key] = valActual
		hasChanges = true
	}

	return modifiedResource, hasChanges
}

/*
 * Compares two slices element by element, applying the ignore list to each element.
 * Returns the modified list and whether there were any changes.
 */
func _compareLists(valRecorded interface{}, valActual interface{}, ignoreList []string) (modifiedList interface{}, hasChanges bool) {
	// Try to get both as slices of interface{}
	sliceRecorded, okRecorded := _toInterfaceSlice(valRecorded)
	sliceActual, okActual := _toInterfaceSlice(valActual)

	if !okRecorded || !okActual {
		// Can't compare as slices with ignores, fall back to deep equal
		return valActual, !reflect.DeepEqual(valRecorded, valActual)
	}

	// If lengths differ, there's definitely a change
	if len(sliceRecorded) != len(sliceActual) {
		return valActual, true
	}

	resultList := make([]interface{}, len(sliceRecorded))
	hasChanges = false

	for i := 0; i < len(sliceRecorded); i++ {
		itemRecorded := sliceRecorded[i]
		itemActual := sliceActual[i]

		// If both items are maps, recursively compare with the ignore list
		mapRecorded, okRecordedMap := itemRecorded.(map[string]interface{})
		mapActual, okActualMap := itemActual.(map[string]interface{})

		if okRecordedMap && okActualMap {
			modifiedItem, itemHasChanges := getDelta(mapRecorded, mapActual, ignoreList)
			if itemHasChanges {
				resultList[i] = modifiedItem
				hasChanges = true
			} else {
				resultList[i] = itemRecorded
			}
		} else {
			// Not maps, just compare directly
			if !reflect.DeepEqual(itemRecorded, itemActual) {
				resultList[i] = itemActual
				hasChanges = true
			} else {
				resultList[i] = itemRecorded
			}
		}
	}

	return resultList, hasChanges
}

/*
 * Converts various slice types to []interface{} for uniform handling.
 */
func _toInterfaceSlice(val interface{}) ([]interface{}, bool) {
	if val == nil {
		return nil, false
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Slice {
		return nil, false
	}

	result := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		result[i] = rv.Index(i).Interface()
	}
	return result, true
}

/*
 * Gets the ignore list for items within a list.
 * For an ignore path like "myList[].field", when called with key="myList",
 * this returns ["field"].
 */
func _getListIgnoreList(key string, ignoreList []string) []string {
	result := []string{}
	listPrefix := key + "[]"

	for _, ignorePath := range ignoreList {
		if ignorePath == listPrefix {
			// Ignoring the entire list items (shouldn't normally happen, but handle it)
			continue
		}
		if strings.HasPrefix(ignorePath, listPrefix+".") {
			// Strip "key[]." prefix to get the path within each item
			remainder := ignorePath[len(listPrefix)+1:]
			result = append(result, remainder)
		}
	}

	return result
}

/*
 * Modifies an ignoreList to be relative to a descended path.
 * E.g. given descendPath = "bar", and the ignoreList [foo, bar.alpha, bar.bravo], this returns [alpha, bravo]
 *
 * This function handles the case where keys might contain dots.
 * It first tries to match the exact key as a prefix, then falls back to treating dots as separators.
 */
func _descendIgnoreList(descendPath string, ignoreList []string) []string {
	result := []string{}
	prefix := descendPath + "."

	for _, ignorePath := range ignoreList {
		// Check if this path starts with "descendPath."
		if strings.HasPrefix(ignorePath, prefix) {
			remainder := ignorePath[len(prefix):]
			if remainder != "" {
				result = append(result, remainder)
			}
		}
	}

	return result
}

/*
 * Checks if the list contains the exact element.
 * This is used for matching keys that might contain dots.
 */
func containsExact(list []string, elem string) bool {
	for _, a := range list {
		if a == elem {
			return true
		}
	}
	return false
}

// Deprecated: Use containsExact instead. Kept for clarity.
func contains(list []string, elem string) bool {
	return containsExact(list, elem)
}
