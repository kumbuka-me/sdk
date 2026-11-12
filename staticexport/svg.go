package staticexport

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

// ValidateSVG rejects active elements, external resources, and oversized SVG content.
func ValidateSVG(data []byte) error {
	if len(data) == 0 || len(data) > MaxSVGBytes {
		return errors.New("svg outside supported size range")
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	root, depth := false, 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		switch element := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				if root || element.Name.Local != "svg" || element.Name.Space != "http://www.w3.org/2000/svg" {
					return errors.New("expected svg root")
				}
				root = true
			}
			if forbiddenSVGElement(strings.ToLower(element.Name.Local)) {
				return errors.New("active svg element")
			}
			for _, attribute := range element.Attr {
				name := strings.ToLower(attribute.Name.Local)
				value := strings.ToLower(strings.TrimSpace(attribute.Value))
				if strings.HasPrefix(name, "on") || name == "href" || name == "src" || unsafeSVGResources(value) {
					return errors.New("unsafe svg attribute")
				}
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.Directive, xml.ProcInst:
			return errors.New("svg directives and processing instructions are not supported")
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(element)) != "" {
				return errors.New("unexpected svg text")
			}
			if unsafeSVGResources(string(element)) {
				return errors.New("unsafe svg resources")
			}
		}
		if depth < 0 {
			return errors.New("invalid svg nesting")
		}
	}
	if !root || depth != 0 {
		return errors.New("incomplete svg document")
	}
	return nil
}

// forbiddenSVGElement reports whether an SVG element can trigger active content or resource loading.
func forbiddenSVGElement(name string) bool {
	switch name {
	case "script", "foreignobject", "iframe", "object", "embed", "image", "use", "a", "animate", "animatetransform", "animatemotion", "set", "audio", "video":
		return true
	}
	return false
}

// unsafeSVGResources rejects external CSS imports, executable strings, and non-local CSS references.
func unsafeSVGResources(source string) bool {
	value := strings.ToLower(strings.TrimSpace(source))
	for _, forbidden := range []string{"javascript:", "@import", "expression(", "-moz-binding", "url(http", "url(//", "url(data:"} {
		if strings.Contains(value, forbidden) {
			return true
		}
	}
	for tail := value; ; {
		index := strings.Index(tail, "url(")
		if index < 0 {
			break
		}
		tail = strings.TrimSpace(tail[index+4:])
		if strings.HasPrefix(tail, "'#") || strings.HasPrefix(tail, "\"#") {
			tail = tail[2:]
		} else if strings.HasPrefix(tail, "#") {
			tail = tail[1:]
		} else {
			return true
		}
		if end := strings.IndexByte(tail, ')'); end >= 0 {
			tail = tail[end+1:]
		} else {
			return true
		}
	}
	return false
}
