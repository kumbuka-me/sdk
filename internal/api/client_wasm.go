//go:build wasip1 && wasm

package api

import (
	"encoding/json"
	"fmt"
	"runtime"
	"unsafe"
)

// hostCall invokes Kumbuka's imported capability function from a WASM guest.
//
//go:wasmimport kumbuka_v1 call
func hostCall(request, length, response, capacity uint32) uint32

// Call transports one capability request using guest-owned buffers. The host
// never calls a guest allocator while the Go reactor is suspended.
func Call(method string, params, result any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	request, err := json.Marshal(CapabilityRequest{Method: method, Params: data})
	if err != nil {
		return err
	}
	response := make([]byte, 4<<20)
	length := hostCall(uint32(uintptr(unsafe.Pointer(&request[0]))), uint32(len(request)), uint32(uintptr(unsafe.Pointer(&response[0]))), uint32(len(response)))
	runtime.KeepAlive(request)
	if length == 0 || uint64(length) > uint64(len(response)) {
		return fmt.Errorf("invalid host response")
	}
	var envelope CapabilityResponse
	if err := json.Unmarshal(response[:length], &envelope); err != nil {
		return err
	}
	if envelope.Error != "" {
		return fmt.Errorf("host: %s", envelope.Error)
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(envelope.Value, result)
}
