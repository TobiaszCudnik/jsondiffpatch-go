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
var FIXTURE = "2"

func main() {

	// TODO command line param
	left := fileToJSON(fmt.Sprintf(`test/fixtures/%s/left.json`, FIXTURE))
	right := fileToJSON(fmt.Sprintf(`test/fixtures/%s/right.json`, FIXTURE))

	// make the diff
	diff := jsondiffpatch.Diff(left, right)

	// pprint
	ppJson, _ := json.MarshalIndent(diff, "", "  ")
	fmt.Printf("%s\n", ppJson)

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
