// Reference implementation:
// https://stackblitz.com/edit/jsondiffpatch

package main

import (
	"../../src/jsondiffpatch"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

var TRIES = 100 * 1000

//var TRIES = 1

func main() {

	// TODO command line param
	left := fileToJSON(fmt.Sprintf(`test/fixtures/1/left.json`))
	right := fileToJSON(fmt.Sprintf(`test/fixtures/1/right.json`))

	start := time.Now()
	for i := 0; i < TRIES; i++ {
		// make the diff
		jsondiffpatch.Diff(left, right)
	}
	t := time.Now()
	elapsed := t.Sub(start)
	result := elapsed.Nanoseconds() / 1000

	fmt.Printf("Tries: %d\n", TRIES)
	fmt.Printf("Time: %d (micro secs)\n", result)
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
