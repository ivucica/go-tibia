package main

import (
	"fmt"
	"io/ioutil"

	wasmer "github.com/wasmerio/wasmer-go/wasmer"

	gowasmer "github.com/mattn/gowasmer" // "gowasmer is a port of the wasm_exec.js file, for Go. It assumes the WebAssembly runtime is wasmer-go"
	// "Alternatively, to avoid using gowasmer, you can compile your Go program to WebAssembly with TinyGo as follows:
	//
	// $ tinygo build -o module.wasm -target wasi .
	// "
	//wasmadapter "github.com/go-wasm-adapter/go-wasm" // found on this path, but declares its own path as the one below:
	//wasmadapter "github.com/vedhavyas/go-wasm" // This redirects to the one above. But they're both unusable.
)

func main() {
	// TODO: use paths/full package instead
	wasmBytes, _ := ioutil.ReadFile("datafiles/html/main.wasm")

	var result interface{}
	choice := 3
	switch choice {
	case 0: // wasmer directly, lowlevel. no wasmi.
		// "Missing import: `go`.`debug`".
		// Need an implementation of wasm_exec.js.

		var sayHello wasmer.NativeFunction
		var err error
		sayHello = basicWasmer(wasmBytes)
		// Calls that exported function with Go standard values. The WebAssembly
		// types are inferred and values are casted automatically.
		//result, _ := sum(5, 37)
		result, err = sayHello()
		if err != nil {
			panic(err)
		}
	case 1: // gowasmer
		// "panic: syscall/js: call of Value.Get on undefined"
		//
		// In Go 1.19's fs_js.go:20. We likely need an implementation of fs which gowasmer does not provide.
		//
		// Looks like:
		// 19: var jsFS = js.Global().Get("fs")
		// 20: var constants = jsFS.Get("constants")
		var sayHello nativeFunc
		sayHello = goWasmer(wasmBytes)
		result = sayHello([]interface{}{})

	/*
		case 2: // go-wasm-adapter
			// sayHello is not a function pointer, we have to use CallFunc.
			// We cannot get wasmer via b.instance since it's unexported.
			//
			// Unusable: does not build with Go 1.19
			// In file included from _cgo_export.c:4:
			// cgo-gcc-export-header-prolog:54:47: error: conflicting types for ‘_’
			// cgo-gcc-export-header-prolog:54:36: note: previous definition of ‘_’ was here
			var err error
			bridge := wasmAdapter(wasmBytes)
			result, err := bridge.CallFunc(b, []interface{}{})
			if err != nil {
				panic(err)
			}
	*/
	case 3: // use wasi
		// use a downloaded wasi file since local dev environment is on 1.19, so we can't get a wasip1 version of main.wasm built
		wasmBytes, _ = ioutil.ReadFile("wasmerio-example-module.wasm")

		var sayHello wasmer.NativeFunction
		var err error
		sayHello, err = wasiWasmer(wasmBytes)
		check(err)

		result, err = sayHello()
		check(err)

	default:
		panic("unknown case hardcoded")
	}

	fmt.Println(result) // 42!
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type nativeFunc = func([]interface{}) interface{}

// basicWasmer sets up wasmer, and returns an instance of ExportedSayHello inside of it.
//
// Note: type NativeFunction = func(...interface{}) (interface{}, error)
func basicWasmer(wasmBytes []byte) wasmer.NativeFunction {
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// Compiles the module
	module, _ := wasmer.NewModule(store, wasmBytes)

	// Instantiates the module
	importObject := wasmer.NewImportObject()
	instance, err := wasmer.NewInstance(module, importObject)
	if err != nil {
		panic(err)
	}

	// Gets the `ExportedSayHello` exported function from the WebAssembly instance.
	sayHello, err := instance.Exports.GetFunction("ExportedSayHello")
	if err != nil {
		panic(err)
	}

	return sayHello
}

// goWasmer sets up gowasmer's GoInstance, containin a wasmer, and returns an instance of ExportedSayHello inside of it.
//
// This is needed because otherwise we lack the wasm_exec.js.
//
// Unclear how this can work given syscall/js requires "fs" which is not implemented here.
func goWasmer(wasmBytes []byte) nativeFunc {
	fmt.Println("starting gowasmer")
	inst, err := gowasmer.NewInstance(wasmBytes)
	if err != nil {
		panic(err)
	}

	fmt.Println("getting func")
	return inst.Get("ExportedSayHello").(nativeFunc)
}

// wasmAdapter has fs: https://github.com/go-wasm-adapter/go-wasm
//
// However:
// In file included from _cgo_export.c:4:
// cgo-gcc-export-header-prolog:54:47: error: conflicting types for ‘_’
// cgo-gcc-export-header-prolog:54:36: note: previous definition of ‘_’ was here
/*
func wasmAdapter(wasmBytes []byte) *wasmadapter.Bridge {
	b, err := wasm.BridgeFromBytes("gotwebfe", wasmBytes) //.BridgeFromFile("test", "./examples/function-wasm/main.wasm", nil)
	if err != nil {
		panic(err)
	}

	return b.
}
*/

// wasi version of wasmer-go should have fs.
func wasiWasmer(wasmBytes []byte) (wasmer.NativeFunction, error) {
	// taken from https://github.com/wasmerio/wasmer-go/blob/49b556ba29cc8816ffe91f09e0846151b2dc0810/examples/wasi/main.go
	store := wasmer.NewStore(wasmer.NewEngine())
	module, _ := wasmer.NewModule(store, wasmBytes)
	wasiEnv, _ := wasmer.NewWasiStateBuilder("wasi-program").
		// Choose according to your actual situation
		// Argument("--foo").
		// Environment("ABC", "DEF").
		// MapDirectory("./", ".").
		Finalize()
	importObject, err := wasiEnv.GenerateImportObject(store, module)
	check(err)

	instance, err := wasmer.NewInstance(module, importObject)
	check(err)

	start, err := instance.Exports.GetWasiStartFunction()
	check(err)
	start()

	//HelloWorld, err := instance.Exports.GetFunction("HelloWorld")
	//result, _ := HelloWorld()

	//return instance.Exports.GetFunction("ExportedSayHello")
	return instance.Exports.GetFunction("HelloWorld")
}
