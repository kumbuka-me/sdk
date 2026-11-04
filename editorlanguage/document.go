// Package editorlanguage defines plugin-owned language choices for Kumbuka's code editor.
// Each plugin may supply assets/code-languages.json without registering executable code.
package editorlanguage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// AssetName is the package-relative path of a plugin's code-language declarations.
const AssetName = "code-languages.json"

const (
	maxDocumentBytes = 16 << 10
	maxLanguages     = 32
)

var languageID = regexp.MustCompile(`^[a-z][a-z0-9_+-]{0,63}$`)

// Language describes one fenced-code language offered by a plugin.
type Language struct {
	// Value is the Markdown fence info-string language identifier.
	Value string `json:"value"`
	// Label is the human-readable name shown in editor pickers.
	Label string `json:"label"`
}

// Document is the versioned declaration stored in assets/code-languages.json.
type Document struct {
	// Version selects the editor-language contract.
	Version int `json:"version"`
	// Languages contains the languages contributed by one plugin.
	Languages []Language `json:"languages"`
}

// Parse strictly validates a plugin's language declarations.
func Parse(data []byte) (Document, error) {
	var document Document
	if len(data) == 0 || len(data) > maxDocumentBytes {
		return document, errors.New("code-language document is empty or too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return Document{}, fmt.Errorf("parse code-language document: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Document{}, errors.New("code-language document must contain exactly one JSON object")
	}
	if document.Version != 1 || len(document.Languages) == 0 || len(document.Languages) > maxLanguages {
		return Document{}, errors.New("unsupported code-language document version or language count")
	}
	seen := make(map[string]bool, len(document.Languages))
	for _, language := range document.Languages {
		if !languageID.MatchString(language.Value) ||
			strings.TrimSpace(language.Label) != language.Label ||
			len(language.Label) == 0 || len(language.Label) > 128 || seen[language.Value] {
			return Document{}, fmt.Errorf("invalid or duplicate language %q", language.Value)
		}
		seen[language.Value] = true
	}
	return document, nil
}
