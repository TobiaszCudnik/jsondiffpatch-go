// Reference implementation:
// https://stackblitz.com/edit/jsondiffpatch

package jsondiffpatch

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
)

var DEBUG = false

// TODO command line param
var BY_ID = false
var TEXT_DIFF_MIN_LENGTH = 60
var GOROUTINE_GROUP_SIZE = 12

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

	diff(left, right, &ret, "root", nil)

	return ret["root"]
}

func diff(left interface{}, right interface{}, ret *map[string]interface{},
	key string, lock *sync.RWMutex) {
	if DEBUG {
		fmt.Printf("make-Diff %s %s \n", left, right)
	}

	if lock == nil {
		lock = &sync.RWMutex{}
	}

	switch leftCasted := left.(type) {

	// array
	case []interface{}:
		diffArray(leftCasted, right, ret, key, lock)

	// object
	case map[string]interface{}:
		diffObject(leftCasted, right, ret, key, lock)

	// string
	case string:
		diffString(leftCasted, right, ret, key, lock)

	// ints
	case float64:
		diffNumber(leftCasted, right, ret, key, lock)

	// booleans
	case bool:
		diffBool(leftCasted, right, ret, key, lock)
	}
}

// ----- DIFFS PER TYPE

func diffArray(left []interface{}, right interface{}, ret *map[string]interface{},
	key string, lock *sync.RWMutex) {
	if DEBUG {
		fmt.Printf("array %s\n", left)
	}

	rightArr, rightOk := right.([]interface{})
	if !rightOk {
		// right isn't an array
		value := changeValue(left, right)
		lock.Lock()
		(*ret)[key] = value
		lock.Unlock()
		return
	}

	// both are arrays, diff by IDs or positions
	if BY_ID {
		diffArrayByID(left, rightArr, ret, key, lock)
	} else {
		diffArrayByPos(left, rightArr, ret, key, lock)
	}
}

func diffArrayByPos(left []interface{}, right []interface{},
	ret *map[string]interface{}, key string, lock *sync.RWMutex) {

	retLocal := make(map[string]interface{})
	lockLocal := sync.RWMutex{}

	if len(left) < GOROUTINE_GROUP_SIZE {
		diffArrayPartByPos(left, right, 0, len(left),
			&retLocal, nil, &lockLocal)
	} else {
		var wg sync.WaitGroup
		// go through the array in parts
		for i := 0; i <= len(left); i += GOROUTINE_GROUP_SIZE {
			wg.Add(1)
			go diffArrayPartByPos(left, right, i, i+GOROUTINE_GROUP_SIZE,
				&retLocal, &wg, &lockLocal)
		}

		// wait for all goroutine groups
		wg.Wait()
	}

	// add new elements from right
	for k, v2 := range right {
		kStr := strconv.Itoa(k)
		// skip all indexes from the left
		if len(left) >= k {
			continue
		}
		value := addValue(v2)
		retLocal[kStr] = value
	}

	// store in the final json
	if len(retLocal) > 0 {
		lock.Lock()
		(*ret)[key] = retLocal
		lock.Unlock()
	}
}

func diffArrayPartByPos(left []interface{}, right []interface{}, start int,
	end int, ret *map[string]interface{}, wg *sync.WaitGroup,
	lock *sync.RWMutex) {

	for k := start; k < end && k < len(left); k++ {
		k := k
		val := left[k]
		kStr := strconv.Itoa(k)
		// remove if right is shorter
		if len(right) <= k {
			value := removeValue(left[k])
			lock.Lock()
			(*ret)[kStr] = value
			lock.Unlock()
			continue
		}
		diff(val, right[k], ret, kStr, lock)
	}

	if wg != nil {
		wg.Done()
	}
}

func diffArrayByID(left []interface{}, right []interface{},
	ret *map[string]interface{}, key string, lock *sync.RWMutex) {
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
		kStr := strconv.Itoa(k)
		// delete if not on the right
		if _, isset := rightIndex[id]; !isset {
			value := removeValue(left[leftIndex[id]])
			lock.Lock()
			(*ret)[kStr] = value
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
		value := addValue(right[rightIndex[id]])
		lock.Lock()
		retLocal[kStr] = value
		lock.Unlock()
	}

	// store in the final json
	if len(retLocal) > 0 {
		lock.Lock()
		(*ret)[key] = retLocal
		lock.Unlock()
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
	ret *map[string]interface{}, key string, lock *sync.RWMutex) {
	if DEBUG {
		fmt.Printf("object-left  %s\n", left)
	}

	rightObj, rightOk := right.(map[string]interface{})

	// right isnt an object
	if !rightOk {
		value := changeValue(left, right)
		lock.Lock()
		(*ret)[key] = value
		lock.Unlock()
		return
	}
	if DEBUG {
		fmt.Printf("object-right %s \n", rightObj)
	}

	retLocal := make(map[string]interface{})
	lockLocal := sync.RWMutex{}

	if len(left) < GOROUTINE_GROUP_SIZE {
		keys := make([]string, 0, len(left))
		for k := range left {
			keys = append(keys, k)
		}
		diffObjectPart(left, rightObj, keys, &retLocal, nil, &lockLocal)
	} else {
		var wg sync.WaitGroup
		keys := make([]string, GOROUTINE_GROUP_SIZE)
		i := 0
		// scan the left for changes against the right
		for k := range left {
			// add a key to the group
			keys[i] = k
			i++
			// parse in groups
			if i != GOROUTINE_GROUP_SIZE {
				continue
			}
			wg.Add(1)
			go diffObjectPart(left, rightObj, keys, &retLocal, &wg, &lockLocal)
			keys = make([]string, GOROUTINE_GROUP_SIZE)
			i = 0
		}

		// parse remaining keys (if any)
		if len(keys) > 0 {
			wg.Add(1)
			go diffObjectPart(left, rightObj, keys, &retLocal, &wg, &lockLocal)
		}

		// wait for all goroutine groups
		wg.Wait()
	}

	// add new elements from the right
	for k, val := range rightObj {
		// skip if set in on the left
		if _, isset := left[k]; isset {
			continue
		}
		value := addValue(val)
		retLocal[k] = value
	}

	// store in the final json
	if len(retLocal) > 0 {
		lock.Lock()
		(*ret)[key] = retLocal
		lock.Unlock()
	}
}

func diffObjectPart(left map[string]interface{}, right map[string]interface{},
	keys []string, ret *map[string]interface{}, wg *sync.WaitGroup,
	lock *sync.RWMutex) {

	// add new elements from the right
	for _, k := range keys {
		diff(left[k], right[k], ret, k, lock)
	}

	if wg != nil {
		wg.Done()
	}
}

func diffString(left string, right interface{},
	ret *map[string]interface{}, key string, lock *sync.RWMutex) {
	if DEBUG {
		fmt.Printf("string %s\n", left)
	}

	// removed
	if right == nil {
		value := removeValue(left)
		lock.Lock()
		(*ret)[key] = value
		lock.Unlock()
		return
	}

	rightStr, rightOk := right.(string)

	if !rightOk {
		// right isnt a string
		value := changeValue(left, right)
		lock.Lock()
		(*ret)[key] = value
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
		value := changeValue(left, right)
		lock.Lock()
		(*ret)[key] = value
		lock.Unlock()
		//}
	}
}

func diffNumber(left float64, right interface{}, ret *map[string]interface{},
	key string, lock *sync.RWMutex) {
	if DEBUG {
		fmt.Printf("int %d\n", left)
	}

	// removed
	if right == nil {
		value := removeValue(left)
		lock.Lock()
		(*ret)[key] = value
		lock.Unlock()
		return
	}

	rightInt, rightOk := right.(float64)

	// right isnt an int or the values differ
	if !rightOk || left != rightInt {
		value := changeValue(left, right)
		lock.Lock()
		(*ret)[key] = value
		lock.Unlock()
	}
}

func diffBool(left bool, right interface{}, ret *map[string]interface{},
	key string, lock *sync.RWMutex) {
	if DEBUG {
		fmt.Printf("bool %s\n", left)
	}

	// removed
	if right == nil {
		value := removeValue(left)
		lock.Lock()
		(*ret)[key] = value
		lock.Unlock()
	}

	rightBool, rightOk := right.(bool)

	// right isnt a bool or the values differ
	if !rightOk || left != rightBool {
		value := changeValue(left, right)
		lock.Lock()
		(*ret)[key] = value
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
