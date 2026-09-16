package pluginpackage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	MaxIconResourceIcons     = 8192
	MaxIconResourceLabel     = 256
	MaxIconResourcePathBytes = 64 << 10
	MaxIconResourcePaths     = 32
)

// IconResource is the versioned JSON payload declared by an icon-resource module.
type IconResource struct {
	// Format identifies the icon resource schema version.
	Format int `json:"format"`
	// Icons contains the validated icons exposed by the resource.
	Icons []Icon `json:"icons"`
}

// Icon is one host-rendered icon in an icon resource.
type Icon struct {
	// Name is the canonical icon identifier.
	Name string `json:"name"`
	// Label is the human-readable icon name.
	Label string `json:"label"`
	// ViewBox is the normalized SVG view box.
	ViewBox string `json:"view_box"`
	// Paths contains one or more SVG path data strings.
	Paths []string `json:"paths"`
}

// ParseIconResource strictly decodes and validates one bounded icon catalog.
func ParseIconResource(data []byte) (IconResource, error) {
	if len(data) == 0 || len(data) > MaxAssetBytes {
		return IconResource{}, errors.New("icon resource exceeds size limit or is empty")
	}

	var resource IconResource
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&resource); err != nil {
		return IconResource{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return IconResource{}, errors.New("icon resource must contain exactly one JSON document")
	}
	if resource.Format != 1 || len(resource.Icons) == 0 || len(resource.Icons) > MaxIconResourceIcons {
		return IconResource{}, errors.New("invalid icon resource")
	}

	seen := make(map[string]bool, len(resource.Icons))
	for index := range resource.Icons {
		if !normalizeAndValidateIcon(&resource.Icons[index], seen) {
			return IconResource{}, errors.New("invalid icon resource")
		}
		seen[resource.Icons[index].Name] = true
	}
	return resource, nil
}

// normalizeAndValidateIcon normalizes one icon and checks its metadata and paths.
func normalizeAndValidateIcon(icon *Icon, seen map[string]bool) bool {
	icon.Name = strings.TrimSpace(icon.Name)
	icon.Label = strings.TrimSpace(icon.Label)
	icon.ViewBox = strings.Join(strings.Fields(icon.ViewBox), " ")
	if !identifier.MatchString(icon.Name) || seen[icon.Name] {
		return false
	}
	if !validIconText(icon.Label, MaxIconResourceLabel) || !validIconViewBox(icon.ViewBox) {
		return false
	}
	return validIconPaths(icon.Paths)
}

// validIconPaths checks bounded UTF-8 SVG path data without interpreting commands.
func validIconPaths(paths []string) bool {
	if len(paths) == 0 || len(paths) > MaxIconResourcePaths {
		return false
	}
	for _, path := range paths {
		if strings.TrimSpace(path) == "" || len(path) > MaxIconResourcePathBytes ||
			!utf8.ValidString(path) || strings.ContainsRune(path, '\x00') {
			return false
		}
	}
	return true
}

// validIconText reports whether human-readable icon metadata is safe and bounded.
func validIconText(value string, limit int) bool {
	if value == "" || len(value) > limit || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

// validIconViewBox accepts four finite SVG view-box numbers with positive dimensions.
func validIconViewBox(value string) bool {
	fields := strings.Fields(value)
	if len(fields) != 4 {
		return false
	}

	var values [4]float64
	for index, field := range fields {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
		values[index] = value
	}
	return values[2] > 0 && values[3] > 0
}
