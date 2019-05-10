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
	channel := make(chan map[string]interface{}, 1)

	diff(left, right, "root", channel)

	ret := <-channel

	return ret["root"]
}

func diff(left interface{}, right interface{}, key string,
	ret <-chan map[string]interface{}) {
	if DEBUG {
		fmt.Printf("make-Diff %s %s \n", left, right)
	}

	switch leftCasted := left.(type) {

	// array
	case []interface{}:
		diffArray(leftCasted, right, key, ret)

	// object
	case map[string]interface{}:
		diffObject(leftCasted, right, key, ret)

	// string
	case string:
		diffString(leftCasted, right, key, ret)

	// ints
	case float64:
		diffNumber(leftCasted, right, key, ret)

	// booleans
	case bool:
		diffBool(leftCasted, right, key, ret)
	}
}

// ----- DIFFS PER TYPE

func diffArray(left []interface{}, right interface{}, key string,
	ret <-chan map[string]interface{}) {
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
		diffArrayByID(left, rightArr, key, ret)
	} else {
		diffArrayByPos(left, rightArr, key, ret)
	}
}

func diffArrayByPos(left []interface{}, right []interface{},
	key string, ret <-chan map[string]interface{}) {

	retLocal := make(map[string]interface{})

	for k, v2 := range left {
		// TODO convert properly
		kStr := fmt.Sprintf(`%d`, k)
		// remove if right is shorter
		if len(right) <= k {
			retLocal[kStr] = removeValue(left[k])
			continue
		}
		diff(v2, right[k], &retLocal, kStr)
	}
	// add new elements from right
	// TODO channel
	for k, v2 := range right {
		// TODO convert properly
		kStr := fmt.Sprintf(`%d`, k)
		// skip all indexes from the left
		if len(left) >= k {
			continue
		}
		retLocal[kStr] = addValue(v2)
	}

	// store in the final json
	(*ret)[key] = retLocal
}

func diffArrayByID(left []interface{}, right []interface{},
	key string, ret <-chan map[string]interface{}) {
	// index by ID
	leftIndex := indexByID(left)
	rightIndex := indexByID(right)
	// indicated IDs which has already been marked as moved (to avoid dups)
	// TODO temp
	//skipMove := map[int]bool{}

	retLocal := make(map[string]interface{})

	// scan the left for changes against the right
	// TODO channel
	for id, k := range leftIndex {
		// TODO convert properly
		kStr := fmt.Sprintf(`%d`, k)
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
		//		// TODO convert properly
		//		rightIndex := fmt.Sprintf(`%d`, rightIndex[id])
		//		// use the index from the RIGHT side
		//		retLocal[rightIndex] = moveValue(leftIndex[id])
		//	}
		//	continue
		//}
	}

	// add new elements from the right
	// TODO channel
	for id, k := range rightIndex {
		// skip if set on the left
		if _, isset := leftIndex[id]; isset {
			continue
		}
		// TODO convert properly
		kStr := fmt.Sprintf(`%d`, k)
		retLocal[kStr] = addValue(right[rightIndex[id]])
	}

	// store in the final json
	(*ret)[key] = retLocal
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
	key string, ret <-chan map[string]interface{}) {
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

	channels := make(chan map[string]interface{}, 1)

	retLocal := make(map[string]interface{})

	// scan the left for changes against the right
	// TODO channel
	for k, v2 := range left {
		diff(v2, rightObj[k], &retLocal, k)
	}

	// add new elements from the right
	// TODO channel
	for k, val := range rightObj {
		// skip if set in on the left
		if _, isset := left[k]; isset {
			continue
		}
		retLocal[k] = addValue(val)
	}

	for i := 0; i < 2; i++ {
		select {
		case msg1 := <-c1:
			fmt.Println("received", msg1)
		case msg2 := <-c2:
			fmt.Println("received", msg2)
		}
	}

	// store in the final json
	(*ret)[key] = retLocal
}

func diffString(left string, right interface{}, key string,
	ret <-chan map[string]interface{}) {
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

func diffNumber(left float64, right interface{}, key string,
	ret <-chan map[string]interface{}) {
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

func diffBool(left bool, right interface{}, key string,
	ret <-chan map[string]interface{}) {
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
