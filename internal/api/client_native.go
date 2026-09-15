//go:build !wasip1 || !wasm

package api

import "errors"

// Call reports that Kumbuka host capabilities are unavailable to native plugin code.
func Call(string, any, any) error { return errors.New("host capabilities require the WASM runtime") }
