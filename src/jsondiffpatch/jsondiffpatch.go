// Reference implementation:
// https://stackblitz.com/edit/jsondiffpatch

package jsondiffpatch

import (
	"encoding/json"
	"fmt"
	"sync"
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

var lock = sync.RWMutex{}

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

	diff(left, right, &ret, "root", nil)

	return ret["root"]
}

func diff(left interface{}, right interface{}, ret *map[string]interface{},
	key string, wg *sync.WaitGroup) {
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

	if wg != nil {
		wg.Done()
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
		lock.Lock()
		(*ret)[key] = changeValue(left, right)
		lock.Unlock()
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
	var wg sync.WaitGroup

	for k, val := range left {
		// TODO convert properly
		kStr := fmt.Sprintf(`%d`, k)
		// remove if right is shorter
		if len(right) <= k {
			lock.Lock()
			retLocal[kStr] = removeValue(left[k])
			lock.Unlock()
			continue
		}
		wg.Add(1)
		go diff(val, right[k], &retLocal, kStr, &wg)
	}
	wg.Wait()
	// add new elements from right
	// TODO channel
	for k, v2 := range right {
		// TODO convert properly
		kStr := fmt.Sprintf(`%d`, k)
		// skip all indexes from the left
		if len(left) >= k {
			continue
		}
		lock.Lock()
		retLocal[kStr] = addValue(v2)
		lock.Unlock()
	}

	// store in the final json
	lock.Lock()
	(*ret)[key] = retLocal
	lock.Unlock()
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
	// TODO channel
	for id, k := range leftIndex {
		// TODO convert properly
		kStr := fmt.Sprintf(`%d`, k)
		// delete if not on the right
		if _, isset := rightIndex[id]; !isset {
			lock.Lock()
			(*ret)[kStr] = removeValue(left[leftIndex[id]])
			lock.Unlock()
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
		lock.Lock()
		retLocal[kStr] = addValue(right[rightIndex[id]])
		lock.Unlock()
	}

	// store in the final json
	lock.Lock()
	(*ret)[key] = retLocal
	lock.Unlock()
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
		lock.Lock()
		(*ret)[key] = changeValue(left, right)
		lock.Unlock()
		return
	}
	if DEBUG {
		fmt.Printf("object-right %s \n", rightObj)
	}

	retLocal := make(map[string]interface{})
	var wg sync.WaitGroup

	// scan the left for changes against the right
	// TODO channel
	for k, val := range left {
		wg.Add(1)
		go diff(val, rightObj[k], &retLocal, k, &wg)
	}
	wg.Wait()

	// add new elements from the right
	// TODO channel
	for k, val := range rightObj {
		// skip if set in on the left
		if _, isset := left[k]; isset {
			continue
		}
		lock.Lock()
		retLocal[k] = addValue(val)
		lock.Unlock()
	}

	// store in the final json
	lock.Lock()
	(*ret)[key] = retLocal
	lock.Unlock()
}

func diffString(left string, right interface{},
	ret *map[string]interface{}, key string) {
	if DEBUG {
		fmt.Printf("string %s\n", left)
	}

	// removed
	if right == nil {
		lock.Lock()
		(*ret)[key] = removeValue(left)
		lock.Unlock()
		return
	}

	rightStr, rightOk := right.(string)

	if !rightOk {
		// right isnt a string
		lock.Lock()
		(*ret)[key] = changeValue(left, right)
		lock.Unlock()
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
		lock.Lock()
		(*ret)[key] = changeValue(left, right)
		lock.Unlock()
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
		lock.Lock()
		(*ret)[key] = removeValue(left)
		lock.Unlock()
		return
	}

	rightInt, rightOk := right.(float64)

	// right isnt an int or the values differ
	if !rightOk || left != rightInt {
		lock.Lock()
		(*ret)[key] = changeValue(left, right)
		lock.Unlock()
	}
}

func diffBool(left bool, right interface{}, ret *map[string]interface{},
	key string) {
	if DEBUG {
		fmt.Printf("bool %s\n", left)
	}

	// removed
	if right == nil {
		lock.Lock()
		(*ret)[key] = removeValue(left)
		lock.Unlock()
	}

	rightBool, rightOk := right.(bool)

	// right isnt a bool or the values differ
	if !rightOk || left != rightBool {
		lock.Lock()
		(*ret)[key] = changeValue(left, right)
		lock.Unlock()
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
