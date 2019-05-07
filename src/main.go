package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var DEBUG = false
var BY_ID = false
var TEXT_DIFF_MIN_LENGTH = 60

var TEXT_DIFF = 2
var MOVED = 3

func main() {

	// TODO command line
	left := fileToJSON(`test/fixtures/1/left.json`)
	right := fileToJSON(`test/fixtures/1/right.json`)

	// make the diff
	diff_json := makeDiff(left, right)

	// pretty print the result
	var diff interface{}
	if err := json.Unmarshal([]byte(diff_json), &diff); err != nil {
		panic(err)
	}
	pp_json, _ := json.MarshalIndent(diff, "", "  ")
	fmt.Printf("%s\n", pp_json)

}

func fileToJSON(path string) interface{} {
	ioutil.ReadFile(path)

	content, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var ret interface{}

	if err := json.Unmarshal(content, &ret); err != nil {
		panic(err)
	}

	return ret
}

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

func makeDiff(left interface{}, right interface{}) string {
	if DEBUG {
		fmt.Printf("make-diff %s %s \n", left, right)
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
		if DEBUG {
			fmt.Printf("string %s\n", leftCasted)
		}

		rightStr, rightOk := right.(string)

		if !rightOk {
			// right isnt a string
			return diffChange(left, right)
		} else if leftCasted != rightStr {
			// strings differ
			// TODO enable once diffmatchpatch implements unidiff
			//diffLongText := len(rightStr) >= TEXT_DIFF_MIN_LENGTH ||
			//	len(leftCasted) >= TEXT_DIFF_MIN_LENGTH
			//
			//if diffLongText {
			//	dmp := getDiffMatchPatch()
			//	diff := dmp.DiffMain(leftCasted, rightStr, false)
			//	json, _ := json.Marshal(diff)
			//	return fmt.Sprintf(`[%s, 0, %d]`, json, TEXT_DIFF)
			//} else {
			return diffChange(left, right)
			//}
		}

	// ints
	case int:
		if DEBUG {
			fmt.Printf("int %d\n", leftCasted)
		}

		rightInt, rightOk := right.(int)

		// right isnt an int or the values differ
		if !rightOk || leftCasted != rightInt {
			return diffChange(left, right)
		}

	// booleans
	case bool:
		if DEBUG {
			fmt.Printf("bool %s\n", leftCasted)
		}

		rightBool, rightOk := right.(bool)

		// right isnt a bool or the values differ
		if !rightOk || leftCasted != rightBool {
			return diffChange(left, right)
		}
	}
	return ""
}

func diffChange(left interface{}, right interface{}) string {
	leftJson, _ := json.Marshal(left)
	rightJson, _ := json.Marshal(right)
	return fmt.Sprintf("[%s, %s]", leftJson, rightJson)
}

// TODO diff els with the .id field
func diffArray(left []interface{}, right interface{}) string {
	if DEBUG {
		fmt.Printf("array %s\n", left)
	}

	rightArr, rightOk := right.([]interface{})
	if !rightOk { // right isn't an array
		leftJson, _ := json.Marshal(left)
		rightJson, _ := json.Marshal(right)
		return fmt.Sprintf("[%s, %s]", leftJson, rightJson)
	}
	if BY_ID {
		return diffArrayByID(left, rightArr)
	} else {
		return diffArrayByPos(left, rightArr)
	}
}

func diffArrayByPos(left []interface{}, right []interface{}) string {
	// init the diff as an array change
	ret := ``
	for k, v2 := range left {
		// remove if right is shorter
		if len(right) <= k {
			json, _ := json.Marshal(left[k])
			ret += fmt.Sprintf(`, "_%d": [%s, 0, 0]`, k, json)
			continue
		}
		diff := makeDiff(v2, right[k])
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
		json, _ := json.Marshal(v2)
		ret += fmt.Sprintf(`, "%d": [%s]`, k, json)
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
	for id, k := range leftIndex {
		// delete if not on right
		if _, isset := rightIndex[id]; !isset {
			json, _ := json.Marshal(left[leftIndex[id]])
			ret += fmt.Sprintf(`, "_%d": [%s, 0, 0]`, k, json)
			continue
		}
		// diff elements
		diff := makeDiff(left[leftIndex[id]], right[rightIndex[id]])
		if diff == "" {
			// move if indexes differ
			_, isset := skipMove[leftIndex[id]]
			if !isset && leftIndex[id] != rightIndex[id] {
				// use the index from the RIGHT side
				ret += fmt.Sprintf(`, "_%d": ["", %d, %d]`, rightIndex[id],
					leftIndex[id], MOVED)
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
		json, _ := json.Marshal(right[rightIndex[id]])
		ret += fmt.Sprintf(`, "%d": [%s]`, k, json)
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
		leftJson, _ := json.Marshal(left)
		rightJson, _ := json.Marshal(right)
		return fmt.Sprintf("[%s, %s]", leftJson, rightJson)
	}
	if DEBUG {
		fmt.Printf("object-right %s \n", rightObj)
	}

	// init the ret
	first := true
	ret := ""
	for k, v2 := range left {
		diff := makeDiff(v2, rightObj[k])
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
		json, _ := json.Marshal(val)
		ret += fmt.Sprintf(`"%s": [%s]`, k, json)
		first = false
	}

	if ret == "" {
		return ""
	}
	return fmt.Sprintf(`{%s}`, ret)
}
