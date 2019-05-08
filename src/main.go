// Reference implementation:
// https://stackblitz.com/edit/jsondiffpatch

package main

import (
	"./jsondiffpatch"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// TODO command line param
var FIXTURE = "1"

func main() {

	// TODO command line param
	left := fileToJSON(fmt.Sprintf(`test/fixtures/%s/left.json`, FIXTURE))
	right := fileToJSON(fmt.Sprintf(`test/fixtures/%s/right.json`, FIXTURE))

	// make the diff
	diff_json := jsondiffpatch.Diff(left, right)

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
