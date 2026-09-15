//go:build wasip1 && wasm

package sdk

import (
	"encoding/json"
	"unsafe"
)

// The host serializes calls. These slices keep the ABI buffers alive until the
// next invocation. No host pointers or Go values cross the WASM boundary.
var input, output []byte

//go:wasmexport kumbuka_api_version
func apiVersion() uint32 { return Version }

//go:wasmexport kumbuka_alloc
func allocate(size uint32) uint32 {
	if size == 0 || size > 4<<20 {
		return 0
	}
	input = make([]byte, size)
	return uint32(uintptr(unsafe.Pointer(&input[0])))
}

//go:wasmexport kumbuka_transform
func invoke(pointer, length uint32) uint64 {
	var result Result
	if !validRequestBuffer(pointer, length) {
		result.Error = "invalid request buffer"
	} else {
		var request Request
		if err := json.Unmarshal(input, &request); err != nil {
			result.Error = "invalid request JSON"
		} else {
			result = Dispatch(request)
		}
	}
	output = encodeResult(result)
	return uint64(len(output))<<32 | uint64(uintptr(unsafe.Pointer(&output[0])))
}

func validRequestBuffer(pointer, length uint32) bool {
	if len(input) == 0 {
		return false
	}
	return pointer == uint32(uintptr(unsafe.Pointer(&input[0]))) && uint64(length) == uint64(len(input))
}
