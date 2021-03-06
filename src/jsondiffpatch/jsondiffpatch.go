// Reference implementation:
// https://stackblitz.com/edit/jsondiffpatch

package jsondiffpatch

import (
	"encoding/json"
	"fmt"
	"strconv"
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

func DiffStrings(left string, right string) interface{} {
	return DiffBytes([]byte(left), []byte(right))
}

func DiffBytes(left []byte, right []byte) interface{} {
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

func Diff(left interface{}, right interface{}) interface{} {
	ret := make(map[string]interface{})

	diff(left, right, &ret, "root")

	return ret["root"]
}

func diff(left interface{}, right interface{}, ret *map[string]interface{},
	key string) {
	if DEBUG {
		fmt.Printf("make-Diff %s %s \n", left, right)
	}

	switch leftCasted := left.(type) {

	// array
	case []interface{}:
		diffArray(leftCasted, right, ret, key)

	// object
	case map[string]interface{}:
		diffObject(leftCasted, right, ret, key)

	// string
	case string:
		diffString(leftCasted, right, ret, key)

	// ints
	case float64:
		diffNumber(leftCasted, right, ret, key)

	// booleans
	case bool:
		diffBool(leftCasted, right, ret, key)
	}
}

// ----- DIFFS PER TYPE

func diffArray(left []interface{}, right interface{}, ret *map[string]interface{},
	key string) {
	if DEBUG {
		fmt.Printf("array %s\n", left)
	}

	rightArr, rightOk := right.([]interface{})
	if !rightOk {
		// right isn't an array
		(*ret)[key] = changeValue(left, right)
		return
	}

	// both are arrays, diff by IDs or positions
	if BY_ID {
		diffArrayByID(left, rightArr, ret, key)
	} else {
		diffArrayByPos(left, rightArr, ret, key)
	}
}

func diffArrayByPos(left []interface{}, right []interface{},
	ret *map[string]interface{}, key string) {

	retLocal := make(map[string]interface{})

	for k, v2 := range left {
		kStr := strconv.Itoa(k)
		// remove if right is shorter
		if len(right) <= k {
			retLocal[kStr] = removeValue(left[k])
			continue
		}
		diff(v2, right[k], &retLocal, kStr)
	}
	// add new elements from right
	for k, v2 := range right {
		kStr := strconv.Itoa(k)
		// skip all indexes from the left
		if len(left) >= k {
			continue
		}
		retLocal[kStr] = addValue(v2)
	}

	// store in the final json
	if len(retLocal) > 0 {
		(*ret)[key] = retLocal
	}
}

func diffArrayByID(left []interface{}, right []interface{},
	ret *map[string]interface{}, key string) {
	// index by ID
	leftIndex := indexByID(left)
	rightIndex := indexByID(right)
	// indicated IDs which has already been marked as moved (to avoid dups)
	// TODO temp
	//skipMove := map[int]bool{}

	retLocal := make(map[string]interface{})

	// scan the left for changes against the right
	for id, k := range leftIndex {
		kStr := strconv.Itoa(k)
		// delete if not on the right
		if _, isset := rightIndex[id]; !isset {
			(*ret)[kStr] = removeValue(left[leftIndex[id]])
			continue
		}
		// diff elements
		// TODO check if any diff happened
		// TODO temp
		//diffed := diff(left[leftIndex[id]], right[rightIndex[id]], retLocal, kStr)
		//if diffed == false {
		//	// move if indexes differ
		//	_, isset := skipMove[leftIndex[id]]
		//	if !isset && leftIndex[id] != rightIndex[id] {
		//		rightIndex := strconv.Itoa(k)
		//		// use the index from the RIGHT side
		//		retLocal[rightIndex] = moveValue(leftIndex[id])
		//	}
		//	continue
		//}
	}

	// add new elements from the right
	for id, k := range rightIndex {
		// skip if set on the left
		if _, isset := leftIndex[id]; isset {
			continue
		}
		kStr := strconv.Itoa(k)
		retLocal[kStr] = addValue(right[rightIndex[id]])
	}

	// store in the final json
	if len(retLocal) > 0 {
		(*ret)[key] = retLocal
	}
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

func diffObject(left map[string]interface{}, right interface{},
	ret *map[string]interface{}, key string) {
	if DEBUG {
		fmt.Printf("object-left  %s\n", left)
	}

	rightObj, rightOk := right.(map[string]interface{})

	// right isnt an object
	if !rightOk {
		(*ret)[key] = changeValue(left, right)
		return
	}
	if DEBUG {
		fmt.Printf("object-right %s \n", rightObj)
	}

	retLocal := make(map[string]interface{})

	// scan the left for changes against the right
	for k, v2 := range left {
		diff(v2, rightObj[k], &retLocal, k)
	}

	// add new elements from the right
	for k, val := range rightObj {
		// skip if set in on the left
		if _, isset := left[k]; isset {
			continue
		}
		retLocal[k] = addValue(val)
	}

	// store in the final json
	if len(retLocal) > 0 {
		(*ret)[key] = retLocal
	}
}

func diffString(left string, right interface{},
	ret *map[string]interface{}, key string) {
	if DEBUG {
		fmt.Printf("string %s\n", left)
	}

	// removed
	if right == nil {
		(*ret)[key] = removeValue(left)
		return
	}

	rightStr, rightOk := right.(string)

	if !rightOk {
		// right isnt a string
		(*ret)[key] = changeValue(left, right)
	} else if left != rightStr {
		// strings differ
		// TODO enable once diffmatchpatch implements unidiff
		//diffLongText := len(rightStr) >= TEXT_DIFF_MIN_LENGTH ||
		//	len(leftCasted) >= TEXT_DIFF_MIN_LENGTH
		//
		//if diffLongText {
		//	dmp := getDiffMatchPatch()
		//	ret[key] = dmp.DiffMain(leftCasted, rightStr, false)
		//} else {
		(*ret)[key] = changeValue(left, right)
		//}
	}
}

func diffNumber(left float64, right interface{}, ret *map[string]interface{},
	key string) {
	if DEBUG {
		fmt.Printf("int %d\n", left)
	}

	// removed
	if right == nil {
		(*ret)[key] = removeValue(left)
		return
	}

	rightInt, rightOk := right.(float64)

	// right isnt an int or the values differ
	if !rightOk || left != rightInt {
		(*ret)[key] = changeValue(left, right)
	}
}

func diffBool(left bool, right interface{}, ret *map[string]interface{},
	key string) {
	if DEBUG {
		fmt.Printf("bool %s\n", left)
	}

	// removed
	if right == nil {
		(*ret)[key] = removeValue(left)
	}

	rightBool, rightOk := right.(bool)

	// right isnt a bool or the values differ
	if !rightOk || left != rightBool {
		(*ret)[key] = changeValue(left, right)
	}
}

// ----- OPERATIONS

func changeValue(left interface{}, right interface{}) []interface{} {
	ret := make([]interface{}, 2)
	ret[0] = left
	ret[1] = right
	return ret
}

func removeValue(left interface{}) []interface{} {
	ret := make([]interface{}, 3)
	ret[0] = left
	ret[1] = 0
	ret[2] = REMOVED
	return ret
}

func addValue(left interface{}) []interface{} {
	ret := make([]interface{}, 1)
	ret[0] = left
	return ret
}

func moveValue(newIndex int) []interface{} {
	ret := make([]interface{}, 3)
	ret[0] = ""
	ret[1] = newIndex
	ret[2] = MOVED
	return ret
}
