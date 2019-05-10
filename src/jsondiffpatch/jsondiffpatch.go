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
//var BY_ID = false
//var TEXT_DIFF_MIN_LENGTH = 60

// types of operations
// TODO enum?
var REMOVED = 0

//var TEXT_DIFF = 2
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
	channel := make(chan fieldChan)

	go diff(left, right, "root", channel)

	field := <-channel

	close(channel)

	return field.value
}

type fieldChan struct {
	key   string
	value interface{}
}

func diff(left interface{}, right interface{}, key string, ret chan<- fieldChan) {
	if DEBUG {
		fmt.Printf("make-Diff %s %s \n", left, right)
	}

	field := fieldChan{key: key, value: nil}

	switch leftCasted := left.(type) {

	// object
	case map[string]interface{}:
		field.value = diffObject(leftCasted, right)

	// array
	case []interface{}:
		field.value = diffArray(leftCasted, right)

	// string
	case string:
		field.value = diffString(leftCasted, right)

	// ints
	case float64:
		field.value = diffNumber(leftCasted, right)

	// booleans
	case bool:
		field.value = diffBool(leftCasted, right)
	}

	ret <- field
}

// ----- DIFFS PER TYPE

func diffArray(left []interface{}, right interface{}) interface{} {
	if DEBUG {
		fmt.Printf("array %s\n", left)
	}

	rightArr, rightOk := right.([]interface{})
	if !rightOk {
		// right isn't an array
		return changeValue(left, right)
	}

	// both are arrays, diff by IDs or positions
	//if BY_ID {
	//	return diffArrayByID(left, rightArr)
	//} else {
	return diffArrayByPos(left, rightArr)
	//}
}

func diffArrayByPos(left []interface{}, right []interface{}) interface{} {

	ret := make(map[string]interface{})
	keys := make([]int, 0)

	for k := range left {
		//k := k
		//val := val
		kStr := strconv.Itoa(k)
		// remove if right is shorter
		if len(right) <= k {
			ret[kStr] = removeValue(left[k])
			continue
		}
		keys = append(keys, k)
	}

	// init the channel
	fieldsChan := make(chan fieldChan, len(keys))
	for _, k := range keys {
		go diff(left[k], right[k], strconv.Itoa(k), fieldsChan)
	}

	// get the results from the channel
	channelToFields(len(keys), fieldsChan, &ret)

	// add new elements from right
	// TODO channel?
	for k, val := range right {
		// skip all indexes from the left
		if len(left) >= k {
			continue
		}
		ret[strconv.Itoa(k)] = addValue(val)
	}

	return ret
}

func channelToFields(counter int, fields chan fieldChan,
	ret *map[string]interface{}) {
	for i := 0; i < counter; i++ {
		field := <-fields
		if field.value != nil {
			(*ret)[field.key] = field.value
		}
	}
	close(fields)
}

//func diffArrayByID(left []interface{}, right []interface{},
//	key string, ret interface{}) {
//	// index by ID
//	leftIndex := indexByID(left)
//	rightIndex := indexByID(right)
//	// indicated IDs which has already been marked as moved (to avoid dups)
//	// TODO temp
//	//skipMove := map[int]bool{}
//
//	retLocal := make(map[string]interface{})
//
//	// scan the left for changes against the right
//	// TODO channel
//	for id, k := range leftIndex {
//		// TODO convert properly
//		kStr := fmt.Sprintf(`%d`, k)
//		// delete if not on the right
//		if _, isset := rightIndex[id]; !isset {
//			(*ret)[kStr] = removeValue(left[leftIndex[id]])
//			continue
//		}
//		// diff elements
//		// TODO check if any diff happened
//		// TODO temp
//		//diffed := diff(left[leftIndex[id]], right[rightIndex[id]], retLocal, kStr)
//		//if diffed == false {
//		//	// move if indexes differ
//		//	_, isset := skipMove[leftIndex[id]]
//		//	if !isset && leftIndex[id] != rightIndex[id] {
//		//		// TODO convert properly
//		//		rightIndex := fmt.Sprintf(`%d`, rightIndex[id])
//		//		// use the index from the RIGHT side
//		//		retLocal[rightIndex] = moveValue(leftIndex[id])
//		//	}
//		//	continue
//		//}
//	}
//
//	// add new elements from the right
//	// TODO channel
//	for id, k := range rightIndex {
//		// skip if set on the left
//		if _, isset := leftIndex[id]; isset {
//			continue
//		}
//		// TODO convert properly
//		kStr := fmt.Sprintf(`%d`, k)
//		retLocal[kStr] = addValue(right[rightIndex[id]])
//	}
//
//	// store in the final json
//	(*ret)[key] = retLocal
//}
//
//// Returns a map of ID -> position (index)
//func indexByID(array []interface{}) map[string]int {
//	index := make(map[string]int)
//	for k, val := range array {
//		valMap, _ := val.(map[string]interface{})
//		id := valMap["id"].(string)
//		index[id] = k
//	}
//	return index
//}

func diffObject(left map[string]interface{}, right interface{}) interface{} {
	if DEBUG {
		fmt.Printf("object-left  %s\n", left)
	}

	rightObj, rightOk := right.(map[string]interface{})

	// right isnt an object
	if !rightOk {
		return changeValue(left, right)
	}
	if DEBUG {
		fmt.Printf("object-right %s \n", rightObj)
	}

	// init the channel and its counter
	fieldsChan := make(chan fieldChan, len(left))
	ret := make(map[string]interface{})

	// scan the left for changes against the right
	for k, val := range left {
		//k := k
		//val := val
		go diff(val, rightObj[k], k, fieldsChan)
	}

	// get the results from the channel
	channelToFields(len(left), fieldsChan, &ret)

	// add new elements from the right
	// TODO channel?
	for k, val := range rightObj {
		// skip if set in on the left
		if _, isset := left[k]; isset {
			continue
		}
		ret[k] = addValue(val)
	}

	return ret
}

func diffString(left string, right interface{}) interface{} {
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
		//	ret[key] = dmp.DiffMain(leftCasted, rightStr, false)
		//} else {
		return changeValue(left, right)
		//}
	}

	return nil
}

func diffNumber(left float64, right interface{}) interface{} {
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

	return nil
}

func diffBool(left bool, right interface{}) interface{} {
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

	return nil
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
