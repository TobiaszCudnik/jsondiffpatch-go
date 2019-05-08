// Reference implementation:
// https://stackblitz.com/edit/jsondiffpatch

package jsondiffpatch

import (
	"encoding/json"
	"fmt"
)

var DEBUG = false

// TODO command line param
var BY_ID = false
var TEXT_DIFF_MIN_LENGTH = 60

// types of operations
// TODO enum?
var REMOVED = 0
var TEXT_DIFF = 2
var MOVED = 3

// TODO enable once diffmatchpatch implements unidiff
//var diffMatchPathInstance *diffmatchpatch.DiffMatchPatch
//
//func getDiffMatchPatch() *diffmatchpatch.DiffMatchPatch {
//	if diffMatchPathInstance != nil {
//		return diffMatchPathInstance
//	}
//	// cache
//	diffMatchPathInstance = diffmatchpatch.New()
//	return diffMatchPathInstance
//}

func DiffStrings(left string, right string) string {
	return DiffBytes([]byte(left), []byte(right))
}

func DiffBytes(left []byte, right []byte) string {
	var leftJson interface{}
	var rightJson interface{}

	if err := json.Unmarshal(left, &leftJson); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(right, &rightJson); err != nil {
		panic(err)
	}

	return Diff(leftJson, rightJson)
}

func Diff(left interface{}, right interface{}) string {
	if DEBUG {
		fmt.Printf("make-Diff %s %s \n", left, right)
	}

	switch leftCasted := left.(type) {

	// array
	case []interface{}:
		return diffArray(leftCasted, right)

	// object
	case map[string]interface{}:
		return diffObject(leftCasted, right)

	// string
	case string:
		return diffString(leftCasted, right)

	// ints
	case float64:
		return diffNumber(leftCasted, right)

	// booleans
	case bool:
		return diffBool(leftCasted, right)
	}

	// TODO return nil
	return ""
}

// ----- DIFFS PER TYPE

func diffArray(left []interface{}, right interface{}) string {
	if DEBUG {
		fmt.Printf("array %s\n", left)
	}

	rightArr, rightOk := right.([]interface{})
	if !rightOk {
		// right isn't an array
		return changeValue(left, right)
	}

	// both are arrays, diff by IDs or positions
	if BY_ID {
		return diffArrayByID(left, rightArr)
	} else {
		return diffArrayByPos(left, rightArr)
	}
}

func diffArrayByPos(left []interface{}, right []interface{}) string {
	// init the Diff as an array change
	ret := ``
	for k, v2 := range left {
		// remove if right is shorter
		if len(right) <= k {
			removeJson := removeValue(left[k])
			ret += fmt.Sprintf(`, "_%d": %s`, k, removeJson)
			continue
		}
		diff := Diff(v2, right[k])
		if diff == "" {
			continue
		}
		ret += fmt.Sprintf(`, "%d": %s`, k, diff)
	}
	// add new elements from right
	for k, v2 := range right {
		// skip all indexes from the left
		if len(left) >= k {
			continue
		}
		addJson := addValue(v2)
		ret += fmt.Sprintf(`, "%d": %s`, k, addJson)
	}
	if ret == "" {
		return ""
	}
	return fmt.Sprintf(`{"_t": "a" %s}`, ret)
}

func diffArrayByID(left []interface{}, right []interface{}) string {
	// index by ID
	leftIndex := indexByID(left)
	rightIndex := indexByID(right)
	// indicated IDs which has already been marked as moved (to avoid dups)
	skipMove := map[int]bool{}

	// init the ret as an array change
	ret := ""
	// scan the left for changes against the right
	for id, k := range leftIndex {
		// delete if not on the right
		if _, isset := rightIndex[id]; !isset {
			removeJson := removeValue(left[leftIndex[id]])
			ret += fmt.Sprintf(`, "_%d": %s`, k, removeJson)
			continue
		}
		// Diff elements
		diff := Diff(left[leftIndex[id]], right[rightIndex[id]])
		if diff == "" {
			// move if indexes differ
			_, isset := skipMove[leftIndex[id]]
			if !isset && leftIndex[id] != rightIndex[id] {
				// use the index from the RIGHT side
				moveJson := moveValue(rightIndex[id], leftIndex[id])
				ret += fmt.Sprintf(`, %s`, moveJson)
				skipMove[rightIndex[id]] = true
			}
			continue
		}
		// use the index from the RIGHT side
		ret += fmt.Sprintf(`, "%d": %s`, rightIndex[id], diff)
	}

	// add new elements from the right
	for id, k := range rightIndex {
		// skip if set on the left
		if _, isset := leftIndex[id]; isset {
			continue
		}
		addJson := addValue(right[rightIndex[id]])
		ret += fmt.Sprintf(`, "%d": %s`, k, addJson)
	}

	if ret == "" {
		return ""
	}
	return fmt.Sprintf(`{"_t": "a" %s}`, ret)
}

// Returns a map of ID -> position (index)
func indexByID(array []interface{}) map[string]int {
	index := make(map[string]int)
	for k, val := range array {
		valMap, _ := val.(map[string]interface{})
		id := valMap["id"].(string)
		index[id] = k
	}
	return index
}

func diffObject(left map[string]interface{}, right interface{}) string {
	if DEBUG {
		fmt.Printf("object-left  %s\n", left)
	}

	rightObj, rightOk := right.(map[string]interface{})

	if !rightOk { // right isnt an object
		return changeValue(left, right)
	}
	if DEBUG {
		fmt.Printf("object-right %s \n", rightObj)
	}

	// init the ret
	first := true
	ret := ""
	// scan the left for changes against the right
	for k, v2 := range left {
		diff := Diff(v2, rightObj[k])
		if diff == "" {
			continue
		}
		if first == false {
			ret += ", "
		}
		ret += fmt.Sprintf(`"%s": %s`, k, diff)
		first = false
	}

	// add new elements from the right
	for k, val := range rightObj {
		// skip if set in on the left
		if _, isset := left[k]; isset {
			continue
		}
		if first == false {
			ret += ", "
		}
		addJson := addValue(val)
		ret += fmt.Sprintf(`"%s": %s`, k, addJson)
		first = false
	}

	if ret == "" {
		return ""
	}
	return fmt.Sprintf(`{%s}`, ret)
}

func diffString(left string, right interface{}) string {
	if DEBUG {
		fmt.Printf("string %s\n", left)
	}

	// removed
	if right == nil {
		return removeValue(left)
	}

	rightStr, rightOk := right.(string)

	if !rightOk {
		// right isnt a string
		return changeValue(left, right)
	} else if left != rightStr {
		// strings differ
		// TODO enable once diffmatchpatch implements unidiff
		//diffLongText := len(rightStr) >= TEXT_DIFF_MIN_LENGTH ||
		//	len(leftCasted) >= TEXT_DIFF_MIN_LENGTH
		//
		//if diffLongText {
		//	dmp := getDiffMatchPatch()
		//	Diff := dmp.DiffMain(leftCasted, rightStr, false)
		//	json, _ := json.Marshal(Diff)
		//	return fmt.Sprintf(`[%s, 0, %d]`, json, TEXT_DIFF)
		//} else {
		return changeValue(left, right)
		//}
	}

	return ""
}

func diffNumber(left float64, right interface{}) string {
	if DEBUG {
		fmt.Printf("int %d\n", left)
	}

	// removed
	if right == nil {
		return removeValue(left)
	}

	rightInt, rightOk := right.(float64)

	// right isnt an int or the values differ
	if !rightOk || left != rightInt {
		return changeValue(left, right)
	}

	return ""
}

func diffBool(left bool, right interface{}) string {
	if DEBUG {
		fmt.Printf("bool %s\n", left)
	}

	// removed
	if right == nil {
		return removeValue(left)
	}

	rightBool, rightOk := right.(bool)

	// right isnt a bool or the values differ
	if !rightOk || left != rightBool {
		return changeValue(left, right)
	}

	return ""
}

// ----- OPERATIONS

func changeValue(left interface{}, right interface{}) string {
	leftJson, _ := json.Marshal(left)
	rightJson, _ := json.Marshal(right)
	return fmt.Sprintf("[%s, %s]", leftJson, rightJson)
}

func removeValue(left interface{}) string {
	leftJson, _ := json.Marshal(left)
	return fmt.Sprintf("[%s, 0, %d]", leftJson, REMOVED)
}

func addValue(left interface{}) string {
	leftJson, _ := json.Marshal(left)
	return fmt.Sprintf("[%s]", leftJson)
}

func moveValue(oldIndex int, newIndex int) string {
	return fmt.Sprintf(`"_%d": ["", %d, %d]`, oldIndex, newIndex, MOVED)
}
