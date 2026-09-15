package sdk

import (
	"encoding/json"
	"fmt"
)

// Handler implements a declared module's render operation.
type Handler func(Request) Result

var handlers = map[string]Handler{}

// RegisterModule registers a handler for a manifest module. Call during init.
func RegisterModule(id string, handler Handler) {
	if id == "" || handler == nil || handlers[id] != nil {
		panic("invalid or duplicate plugin module registration: " + id)
	}
	handlers[id] = handler
}

// Dispatch runs a registered handler. It also supports native unit tests without WASM.
func Dispatch(request Request) (result Result) {
	defer func() {
		if recover() != nil {
			result = Result{Error: "plugin handler panicked"}
		}
	}()
	if request.APIVersion != Version {
		return Result{Error: "unsupported plugin API version"}
	}
	handler := handlers[request.Module]
	if handler == nil {
		return Result{Error: "unregistered plugin module: " + request.Module}
	}
	return handler(request)
}

// Text returns literal intermediate output; Kumbuka still applies its sanitizer.
func Text(text string) Result { return Result{Parts: []Part{{Text: text}}} }

// Markdown asks Kumbuka to render a nested Markdown fragment.
func Markdown(source string) Result { return Result{Parts: []Part{{Markdown: &source}}} }

// Failure returns a render error.
func Failure(err error) Result {
	if err == nil {
		return Result{}
	}
	return Result{Error: err.Error()}
}

// RegisterWidget registers a widget renderer for one manifest widget module.
func RegisterWidget(id string, render func(WidgetContext) (Result, error)) {
	RegisterModule(id, func(request Request) Result {
		if request.Stage != "widget" || request.Widget == nil || !ValidWidgetSurface(request.Widget.Surface) {
			return Result{Error: "unsupported widget request"}
		}
		context := *request.Widget
		context.Features = request.Features
		result, err := render(context)
		if err != nil {
			return Failure(err)
		}
		return result
	})
}

// RegisterMacro hides the parse/render transport and invocation serialization.
func RegisterMacro[T any](id string, parse func(string) (T, bool), render func(T) (Result, error)) {
	RegisterModule(id, func(request Request) Result {
		switch request.Stage {
		case "parse":
			value, matched := parse(request.Source)
			if !matched {
				return Result{}
			}
			data, err := json.Marshal(value)
			if err != nil {
				return Failure(err)
			}
			return Result{Matched: true, Invocation: data}
		case "macro":
			var value T
			if err := json.Unmarshal(request.Invocation, &value); err != nil {
				return Failure(fmt.Errorf("decode macro arguments: %w", err))
			}
			result, err := render(value)
			if err != nil {
				return Failure(err)
			}
			return result
		default:
			return Result{Error: "unsupported macro stage"}
		}
	})
}

func encodeResult(result Result) []byte {
	data, err := json.Marshal(result)
	if err != nil {
		return []byte(`{"error":"invalid plugin response"}`)
	}
	if len(data) > 4<<20 {
		return []byte(`{"error":"plugin response exceeds wire limit"}`)
	}
	return data
}
