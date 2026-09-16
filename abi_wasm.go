//go:build wasip1 && wasm

package sdk

import (
	"encoding/json"
	"unsafe"
)

// The host serializes calls. These slices keep the ABI buffers alive until the
// next invocation. No host pointers or Go values cross the WASM boundary.
var input, output []byte

// apiVersion returns the plugin API version through the WASM ABI.
//
//go:wasmexport kumbuka_api_version
func apiVersion() uint32 { return Version }

// allocate reserves the bounded guest request buffer and returns its pointer.
//
//go:wasmexport kumbuka_alloc
func allocate(size uint32) uint32 {
	if size == 0 || size > maxWireBytes {
		return 0
	}
	input = make([]byte, size)
	return uint32(uintptr(unsafe.Pointer(&input[0])))
}

// invoke validates and dispatches one host request and returns the response buffer tuple.
//
//go:wasmexport kumbuka_transform
func invoke(pointer, length uint32) uint64 {
	output = encodeResult(dispatchInput(pointer, length))
	return uint64(len(output))<<32 | uint64(uintptr(unsafe.Pointer(&output[0])))
}

// dispatchInput decodes the current request buffer and dispatches a valid request.
func dispatchInput(pointer, length uint32) Result {
	if !validRequestBuffer(pointer, length) {
		return Result{Error: "invalid request buffer"}
	}

	var request Request
	if err := json.Unmarshal(input, &request); err != nil {
		return Result{Error: "invalid request JSON"}
	}
	return Dispatch(request)
}

// validRequestBuffer reports whether the host supplied the current allocation exactly.
func validRequestBuffer(pointer, length uint32) bool {
	if len(input) == 0 {
		return false
	}
	return pointer == uint32(uintptr(unsafe.Pointer(&input[0]))) && uint64(length) == uint64(len(input))
}
