package sdk

import (
	"encoding/json"
	"fmt"
)

const maxWireBytes = 4 << 20

// Handler implements a declared module's render operation.
type Handler func(Request) Result

var handlers = map[string]Handler{}

// RegisterModule registers a handler for a manifest module. Call during init.
func RegisterModule(id string, handler Handler) {
	if id == "" || handler == nil {
		panic("invalid or duplicate plugin module registration: " + id)
	}
	if _, exists := handlers[id]; exists {
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
	RegisterModule(id, widgetHandler(render, nil))
}

// RegisterWidgetWithCommands registers a widget renderer plus host-mediated command handler.
func RegisterWidgetWithCommands(
	id string,
	render func(WidgetContext) (Result, error),
	command func(WidgetCommandContext) (WidgetCommandResult, error),
) {
	RegisterModule(id, widgetHandler(render, command))
}

// RegisterExporter registers one page exporter for a manifest exporter module.
func RegisterExporter(id string, export func(ExportContext) (ExportFile, error)) {
	RegisterModule(id, exporterHandler(export))
}

// exporterHandler adapts a typed exporter to the generic module handler contract.
func exporterHandler(export func(ExportContext) (ExportFile, error)) Handler {
	if export == nil {
		return nil
	}
	return func(request Request) Result {
		if request.Stage != "export" || request.Export == nil {
			return Result{Error: "unsupported exporter request"}
		}

		context := *request.Export
		context.Features = request.Features
		file, err := export(context)
		if err != nil {
			return Failure(err)
		}
		return Result{File: &file}
	}
}

// widgetHandler adapts a typed widget renderer to the generic module handler contract.
func widgetHandler(
	render func(WidgetContext) (Result, error),
	command func(WidgetCommandContext) (WidgetCommandResult, error),
) Handler {
	if render == nil {
		return nil
	}
	return func(request Request) Result {
		switch request.Stage {
		case "widget":
			return renderWidgetRequest(request, render)
		case "widget-command":
			return renderWidgetCommand(request, command)
		default:
			return Result{Error: "unsupported widget request"}
		}
	}
}

// renderWidgetRequest validates and invokes one typed widget render request.
func renderWidgetRequest(request Request, render func(WidgetContext) (Result, error)) Result {
	if request.Widget == nil || !ValidWidgetSurface(request.Widget.Surface) {
		return Result{Error: "unsupported widget request"}
	}
	context := *request.Widget
	context.Features = request.Features
	result, err := render(context)
	if err != nil {
		return Failure(err)
	}
	return result
}

// renderWidgetCommand validates and invokes one typed widget command request.
func renderWidgetCommand(
	request Request,
	command func(WidgetCommandContext) (WidgetCommandResult, error),
) Result {
	if command == nil || request.WidgetCommand == nil ||
		!ValidWidgetSurface(request.WidgetCommand.Surface) || request.WidgetCommand.Action == "" {
		return Result{Error: "unsupported widget command"}
	}
	context := *request.WidgetCommand
	context.Features = request.Features
	result, err := command(context)
	if err != nil {
		return Failure(err)
	}
	return Result{WidgetCommand: &result}
}

// RegisterMacro hides the parse/render transport and invocation serialization.
func RegisterMacro[T any](id string, parse func(string) (T, bool), render func(T) (Result, error)) {
	RegisterModule(id, macroHandler(parse, render))
}

// macroHandler adapts typed macro parse and render callbacks to module stages.
func macroHandler[T any](parse func(string) (T, bool), render func(T) (Result, error)) Handler {
	if parse == nil || render == nil {
		return nil
	}
	return func(request Request) Result {
		switch request.Stage {
		case "parse":
			return parseMacroRequest(request.Source, parse)
		case "macro":
			return renderMacroRequest(request.Invocation, render)
		default:
			return Result{Error: "unsupported macro stage"}
		}
	}
}

// parseMacroRequest serializes a matched typed macro invocation for the host.
func parseMacroRequest[T any](source string, parse func(string) (T, bool)) Result {
	value, matched := parse(source)
	if !matched {
		return Result{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return Failure(err)
	}
	return Result{Matched: true, Invocation: data}
}

// renderMacroRequest decodes a typed macro invocation and renders it.
func renderMacroRequest[T any](invocation json.RawMessage, render func(T) (Result, error)) Result {
	var value T
	if err := json.Unmarshal(invocation, &value); err != nil {
		return Failure(fmt.Errorf("decode macro arguments: %w", err))
	}
	result, err := render(value)
	if err != nil {
		return Failure(err)
	}
	return result
}

// encodeResult serializes one bounded result into the plugin wire response.
func encodeResult(result Result) []byte {
	data, err := json.Marshal(result)
	if err != nil {
		return []byte(`{"error":"invalid plugin response"}`)
	}
	if len(data) > maxWireBytes {
		return []byte(`{"error":"plugin response exceeds wire limit"}`)
	}
	return data
}
