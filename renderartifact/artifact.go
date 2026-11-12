// Package renderartifact specifies the passive, content-addressed SVG artifact boundary.
// A host authorizes all reads, chooses storage, and never trusts URLs supplied by a guest.
package renderartifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// SVGMediaType is the only artifact format currently accepted by the host.
	SVGMediaType = "image/svg+xml"
	// MaxSVGBytes is the largest permitted passive SVG body.
	MaxSVGBytes = 512 << 10
)

// Digest is the immutable, content-addressed identifier for one artifact body.
func Digest(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Validate checks an untrusted artifact before storage or public presentation.
func Validate(mediaType string, data []byte) error {
	if mediaType != SVGMediaType {
		return errors.New("unsupported artifact media type")
	}
	if len(data) < 16 || len(data) > MaxSVGBytes {
		return errors.New("svg exceeds artifact size bounds")
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	depth, roots := 0, 0
	var elements []string
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid svg xml: %w", err)
		}
		switch item := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				if roots != 0 || item.Name.Local != "svg" || item.Name.Space != "http://www.w3.org/2000/svg" {
					return errors.New("expected one svg root")
				}
				roots++
			}
			if unsafeElement(item.Name.Local) {
				return fmt.Errorf("unsafe svg element %q", item.Name.Local)
			}
			for _, attribute := range item.Attr {
				if unsafeAttribute(attribute) {
					return fmt.Errorf("unsafe svg attribute %q", attribute.Name.Local)
				}
			}
			depth++
			elements = append(elements, strings.ToLower(item.Name.Local))
		case xml.EndElement:
			depth--
			elements = elements[:len(elements)-1]
		case xml.Directive, xml.ProcInst:
			return errors.New("svg directives are forbidden")
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(item)) != "" {
				return errors.New("unexpected svg document text")
			}
			if depth > 0 && len(elements) > 0 && elements[len(elements)-1] == "style" && unsafeStyle(string(item)) {
				return errors.New("unsafe svg style")
			}
		}
	}
	if roots != 1 || depth != 0 {
		return errors.New("incomplete svg document")
	}
	return nil
}

// unsafeElement rejects active, embedded and linked content from an SVG image.
func unsafeElement(value string) bool {
	switch strings.ToLower(value) {
	case "script", "foreignobject", "iframe", "object", "embed", "image", "use", "a", "animate", "animatemotion", "animatetransform", "set", "audio", "video":
		return true
	default:
		return false
	}
}

// unsafeAttribute rejects executable attributes and all external SVG references.
func unsafeAttribute(attribute xml.Attr) bool {
	name := strings.ToLower(attribute.Name.Local)
	if name == "xmlns" && attribute.Value == "http://www.w3.org/2000/svg" {
		return false
	}
	if attribute.Name.Space == "xmlns" && name == "xlink" && attribute.Value == "http://www.w3.org/1999/xlink" {
		return false
	}
	if attribute.Name.Space == "xmlns" || strings.HasPrefix(name, "xmlns") {
		return true
	}
	value := strings.ToLower(strings.TrimSpace(attribute.Value))
	if strings.HasPrefix(name, "on") || name == "href" || name == "src" || name == "formaction" {
		return true
	}
	return unsafeStyle(value)
}

// unsafeStyle rejects executable CSS and non-fragment URL references.
func unsafeStyle(value string) bool {
	value = strings.ToLower(value)
	for _, bad := range []string{"javascript:", "@import", "expression(", "-moz-binding", "data:", "http:", "https:"} {
		if strings.Contains(value, bad) {
			return true
		}
	}
	for tail := value; ; {
		index := strings.Index(tail, "url(")
		if index < 0 {
			return false
		}
		tail = strings.TrimSpace(tail[index+4:])
		if !strings.HasPrefix(tail, "#") && !strings.HasPrefix(tail, "'#") && !strings.HasPrefix(tail, `"#`) {
			return true
		}
		tail = tail[1:]
	}
}
