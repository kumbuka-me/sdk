//go:build wasip1 && wasm

package api

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

const maxCapabilityResponseBytes = 4 << 20

var (
	capabilityCallMu         sync.Mutex
	capabilityResponseBuffer []byte
)

// hostCall invokes Kumbuka's imported capability function from a WASM guest.
//
//go:wasmimport kumbuka_v1 call
func hostCall(request, length, response, capacity uint32) uint32

// Call transports one capability request through guest-owned WASM buffers.
func Call(method string, params, result any) error {
	request, err := encodeCapabilityRequest(method, params)
	if err != nil {
		return err
	}

	// The response buffer is guest-owned and reused across calls. Keep the host call
	// and decode together so concurrent plugin goroutines cannot overwrite it.
	capabilityCallMu.Lock()
	defer capabilityCallMu.Unlock()

	response, err := callHost(request)
	if err != nil {
		return err
	}
	return decodeCapabilityResponse(response, result)
}

// encodeCapabilityRequest serializes method parameters directly into the host request envelope.
func encodeCapabilityRequest(method string, params any) ([]byte, error) {
	return json.Marshal(struct {
		// Method selects the host capability operation.
		Method string `json:"method"`
		// Params contains method-specific request parameters.
		Params any `json:"params,omitempty"`
	}{Method: method, Params: params})
}

// callHost invokes the imported host function and returns the bounded response bytes.
func callHost(request []byte) ([]byte, error) {
	if cap(capabilityResponseBuffer) < maxCapabilityResponseBytes {
		capabilityResponseBuffer = make([]byte, maxCapabilityResponseBytes)
	}
	response := capabilityResponseBuffer[:maxCapabilityResponseBytes]
	length := hostCall(
		uint32(uintptr(unsafe.Pointer(&request[0]))),
		uint32(len(request)),
		uint32(uintptr(unsafe.Pointer(&response[0]))),
		uint32(len(response)),
	)
	// Keep request alive until the host has finished reading the guest-owned buffer.
	runtime.KeepAlive(request)
	if length == 0 || uint64(length) > uint64(len(response)) {
		return nil, fmt.Errorf("invalid host response")
	}
	return response[:length], nil
}

// decodeCapabilityResponse validates the host envelope and decodes its optional value.
func decodeCapabilityResponse(response []byte, result any) error {
	var envelope CapabilityResponse
	if err := json.Unmarshal(response, &envelope); err != nil {
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
