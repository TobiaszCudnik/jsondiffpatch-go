package main

import (
	"../jsondiffpatch"
	"encoding/json"
	"fmt"
	"syscall/js"
)

func diff(this js.Value, i []js.Value) interface{} {
	left := i[0].String()
	right := i[1].String()
	diff := jsondiffpatch.DiffStrings(left, right)
	diffJson, _ := json.Marshal(diff)
	diffString := fmt.Sprintf("%s", diffJson)
	println(diffString)

	return nil
}

func registerCallbacks() {
	js.Global().Set("diff", js.FuncOf(diff))
}

func main() {
	c := make(chan struct{}, 0)

	println("WASM Go Initialized")
	// register functions
	registerCallbacks()
	<-c
}
