package main

import (
	"../jsondiffpatch"
	"syscall/js"
)

func diff(this js.Value, i []js.Value) interface{} {
	left := i[0].String()
	right := i[1].String()
	diff := jsondiffpatch.DiffStrings(left, right)
	println(diff)

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
