// Package staticexport defines the optional, browser-based artifact export contract for plugins.
// The declaration is an ordinary plugin asset, so hosts that do not support exports remain compatible.
package staticexport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// DeclarationAsset is the optional browser asset advertising static export modules.
	DeclarationAsset = "static-exports.json"
	// Version is the current static export contract version.
	Version = 1
	// SVGMediaType is the supported passive vector image output format.
	SVGMediaType = "image/svg+xml"
	// MaxSVGBytes bounds each generated vector image.
	MaxSVGBytes         = 1 << 20
	maxDeclarationBytes = 16 << 10
)

// Declaration advertises the browser modules that can export generated assets.
type Declaration struct {
	// Version identifies the versioned export contract.
	Version int `json:"version"`
	// Modules lists the exporting browser modules in this plugin package.
	Modules []Module `json:"modules"`
}

// Module describes the export capability of an existing browser module.
type Module struct {
	// ID matches the browser-module ID in the plugin manifest.
	ID string `json:"id"`
	// MediaType identifies the generated artifact format.
	MediaType string `json:"media_type"`
}

// Result is returned by a browser module's optional exportStatic(root, context) function.
type Result struct {
	// Content contains generated UTF-8 artifact bytes, or is empty when no export is available.
	Content string `json:"content"`
	// Alt is an optional accessible description of the passive image.
	Alt string `json:"alt,omitempty"`
	// Alignment is an optional left, center, or right alignment preference.
	Alignment string `json:"alignment,omitempty"`
}

// ParseDeclaration reads a bounded JSON declaration, rejecting unknown fields and unsupported exports.
func ParseDeclaration(data []byte) (Declaration, error) {
	if len(data) == 0 || len(data) > maxDeclarationBytes {
		return Declaration{}, errors.New("static export declaration is missing or too large")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var declaration Declaration
	if err := decoder.Decode(&declaration); err != nil {
		return Declaration{}, fmt.Errorf("decode static export declaration: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Declaration{}, errors.New("static export declaration contains trailing data")
	}
	if declaration.Version != Version {
		return Declaration{}, fmt.Errorf("unsupported static export version %d", declaration.Version)
	}
	if len(declaration.Modules) == 0 || len(declaration.Modules) > 32 {
		return Declaration{}, errors.New("static export declaration must contain 1 to 32 modules")
	}

	seen := make(map[string]bool, len(declaration.Modules))
	for _, module := range declaration.Modules {
		if !validModuleID(module.ID) || seen[module.ID] || module.MediaType != SVGMediaType {
			return Declaration{}, fmt.Errorf("invalid static export module %q", module.ID)
		}
		seen[module.ID] = true
	}
	return declaration, nil
}

// ValidateResult checks bounded presentation metadata and the declared output format.
func ValidateResult(result Result, mediaType string) error {
	if mediaType != SVGMediaType {
		return fmt.Errorf("unsupported static artifact media type %q", mediaType)
	}
	if len(result.Alt) > 512 || strings.ContainsAny(result.Alt, "\x00\r\n") {
		return errors.New("invalid static artifact alternative text")
	}
	if result.Alignment != "" && result.Alignment != "left" && result.Alignment != "center" && result.Alignment != "right" {
		return errors.New("invalid static artifact alignment")
	}
	return ValidateSVG([]byte(result.Content))
}

// validModuleID accepts the same conservative ASCII identifiers used by browser modules.
func validModuleID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, character := range id {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-' || character == '_' {
			continue
		}
		return false
	}
	return true
}
