package restapi

import (
	"reflect"
	"strings"
)

/*
 * Performs a deep comparison of two maps, returning true if there is a difference between them.
 * Accepts a third argument that is a set of fields that are to be ignored when looking for differences.
 */
func hasDelta(mapA map[string]interface{}, mapB map[string]interface{}, ignoreList []string) bool {
	// Keep a set of keys we've check in mapA, to reduce work when checking keys in mapB
	checkedKeys := map[string]struct{} {}

	for key, valA := range mapA {

		checkedKeys[key] = struct{}{}

		// If the ignore_list contains the current key, don't compare
		if contains(ignoreList, key) {
			continue
		}

		valB := mapB[key]

		// If valA was a map, assert both values are maps
		if reflect.TypeOf(valA).Kind() == reflect.Map {
			subMapA, okA := valA.(map[string]interface{})
			subMapB, okB := valB.(map[string]interface{})
			if !okA || !okB {
				return true
			}
			// Recursively compare
			if hasDelta(subMapA, subMapB, _descendIgnoreList(key, ignoreList)) {
				return true
			}
		} else if reflect.TypeOf(valA).Kind() == reflect.Slice {
			// Since we don't support ignoring differences in lists (besides ignoring the list as a
			// whole), it is safe to deep compare the two list values.
			if ! reflect.DeepEqual(valA, valB) {
				return true
			}
		} else {
			if valA != valB {
				return true
			}
		}

	}

	for key := range mapB {
		// We may have already compared this key with mapA, and confirmed there is no difference.
		_, alreadyChecked := checkedKeys[key]
		if alreadyChecked {
			continue
		}

		// If the ignore_list contains the current key, don't compare
		if contains(ignoreList, key) {
			continue
		}

		// If we've gotten here, that means mapB has an additional key that wasn't in mapA
		return true
	}

	return false
}

/*
 * Modifies an ignoreList to be relative to a descended path.
 * E.g. given descendPath = "bar", and the ignoreList [foo, bar.alpha, bar.bravo], this returns [alpha, bravo]
 */
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
