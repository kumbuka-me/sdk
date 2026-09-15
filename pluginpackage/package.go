// Package pluginpackage reads bounded, versioned .kumbukaplugin ZIP archives.
// Archives are kept in memory and never extracted into the host filesystem.
package pluginpackage

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kumbuka-me/sdk/internal/api"
	"go.yaml.in/yaml/v3"
)

const (
	MaxArchiveBytes  = 16 << 20
	MaxExpandedBytes = 32 << 20
	MaxWASMBytes     = 16 << 20
	MaxAssetBytes    = 8 << 20
	MaxManifestBytes = 64 << 10
	MaxREADMEBytes   = 256 << 10
	MaxFiles         = 256
)

// Manifest is the complete v1 package manifest. Unknown fields and unsupported
// modules or permissions are rejected rather than silently ignored.
type Manifest struct {
	// Provider is self-declared package author metadata, not a trust grant.
	Provider string `yaml:"provider,omitempty"`
	// APIVersion selects the manifest and runtime contract version.
	APIVersion int `yaml:"api_version"`
	// ID is the globally unique plugin identifier.
	ID string `yaml:"id"`
	// Name is the human-readable name.
	Name string `yaml:"name"`
	// Version identifies the associated plugin version.
	Version string `yaml:"version"`
	// Description is optional human-readable plugin metadata.
	Description string `yaml:"description,omitempty"`
	// DefaultEnabled indicates whether the plugin starts enabled by default.
	DefaultEnabled bool `yaml:"default_enabled,omitempty"`
	// Requires lists plugin IDs that must be enabled first.
	Requires []string `yaml:"requires,omitempty"`
	// Modules declares contributions provided by the package.
	Modules []Module `yaml:"modules"`
	// Permissions declares host capabilities the package may request.
	Permissions []string `yaml:"permissions"`
}

// RequiresWASM reports whether any module in this manifest executes guest code.
// Declarative modules are handled entirely by the host and do not need a WASM guest.
func (m Manifest) RequiresWASM() bool {
	for _, module := range m.Modules {
		switch module.Type {
		case "renderer-extension", "macro", "code-highlighter":
			return true
		}
	}

	return false
}

// ResourceField declares one field in a plugin-owned admin resource.
type ResourceField struct {
	// ID identifies the field in stored records.
	ID string `yaml:"id"`
	// Name is the human-readable field label.
	Name string `yaml:"name"`
	// Type selects a bounded host-rendered input control.
	Type string `yaml:"type"`
	// Required reports whether the field may be empty.
	Required bool `yaml:"required,omitempty"`
	// Key identifies the unique record key field. Exactly one field must be the key.
	Key bool `yaml:"key,omitempty"`
	// MaxBytes bounds the UTF-8 encoded field value. Zero selects a host default.
	MaxBytes int `yaml:"max_bytes,omitempty"`
}

// UsageRule declares a cheap source selector used to avoid invoking a module
// for pages that cannot contain its syntax. Exactly one selector is set.
type UsageRule struct {
	// Contains matches literal Markdown text. False positives are safe; false negatives are not.
	Contains string `yaml:"contains,omitempty"`
	// Fence matches a fenced-code info-string language; "*" matches any fenced code block.
	Fence string `yaml:"fence,omitempty"`
	// Macro matches a standalone {{name ...}} invocation and captures its argument text.
	Macro string `yaml:"macro,omitempty"`
	// Substitution matches inline {{prefix:value}} syntax outside fenced code and captures values.
	Substitution string `yaml:"substitution,omitempty"`
}

// Module declares one plugin contribution in a package manifest.
type Module struct {
	// Type selects the contribution module kind.
	Type string `yaml:"type"`
	// ID identifies this module within its plugin package.
	ID string `yaml:"id"`
	// Stage selects the renderer pipeline stage when applicable.
	Stage string `yaml:"stage,omitempty"`
	// Name is the human-readable name.
	Name string `yaml:"name,omitempty"`
	// Description explains the module to administrators when applicable.
	Description string `yaml:"description,omitempty"`
	// Capability names a render-scope capability required by the module.
	Capability string `yaml:"capability,omitempty"`
	// JavaScript names the browser module JavaScript asset.
	JavaScript string `yaml:"javascript,omitempty"`
	// Syntax selects a public standard Markdown grammar.
	Syntax string `yaml:"syntax,omitempty"`
	// Requires names prerequisite modules within this package.
	Requires []string `yaml:"requires,omitempty"`
	// CSS names the stylesheet asset used by browser, content-style, or code-highlighter modules.
	CSS string `yaml:"css,omitempty"`
	// Asset names the data asset owned by declarative resource modules.
	Asset string `yaml:"asset,omitempty"`
	// Policy selects a host rendering policy for render-policy modules.
	Policy string `yaml:"policy,omitempty"`
	// Priority orders content preprocessors. Lower values run first.
	Priority int `yaml:"priority,omitempty"`
	// Fields declares a bounded schema for admin-resource records.
	Fields []ResourceField `yaml:"fields,omitempty"`
	// Resource identifies another module in this package that owns persisted records.
	Resource string `yaml:"resource,omitempty"`
	// Prefix identifies the inline macro prefix handled by a content-substitution module.
	Prefix string `yaml:"prefix,omitempty"`
	// ValueField selects the resource field inserted into Markdown.
	ValueField string `yaml:"value_field,omitempty"`
	// LabelField selects the resource field used as a human-readable label.
	LabelField string `yaml:"label_field,omitempty"`
	// DetailField selects optional descriptive resource text.
	DetailField string `yaml:"detail_field,omitempty"`
	// Trigger opens a resource-backed editor completion provider.
	Trigger string `yaml:"trigger,omitempty"`
	// Replacement formats a selected resource record into Markdown.
	Replacement string `yaml:"replacement,omitempty"`
	// Markdown is the static source inserted by an editor-insert module.
	Markdown string `yaml:"markdown,omitempty"`
	// Inline reports whether editor insertion should avoid block line breaks.
	Inline bool `yaml:"inline,omitempty"`
	// Usage declares cheap host-side source selectors for executable modules.
	Usage []UsageRule `yaml:"usage,omitempty"`
	// Inspect enables a reading-page inspector for content substitutions.
	Inspect bool `yaml:"inspect,omitempty"`
	// Export enables request-local export overrides for content substitutions.
	Export bool `yaml:"export,omitempty"`
}

// Package exposes copies of validated content so callers cannot mutate the
// package after validation. The digest covers the original ZIP bytes.
type Package struct {
	// manifest contains the validated plugin manifest.
	manifest Manifest
	// wasm contains the validated guest module bytes.
	wasm []byte
	// assets indexes validated package assets by archive-relative path.
	assets map[string][]byte
	// readme contains the package documentation rendered by the administration UI.
	readme []byte
	// digest is the SHA-256 digest of the original package archive.
	digest [32]byte
}

// Manifest returns an independent copy of the validated package manifest.
func (p *Package) Manifest() Manifest {
	m := p.manifest
	m.Modules = slices.Clone(m.Modules)
	for i := range m.Modules {
		m.Modules[i].Requires = slices.Clone(m.Modules[i].Requires)
		m.Modules[i].Fields = slices.Clone(m.Modules[i].Fields)
		m.Modules[i].Usage = slices.Clone(m.Modules[i].Usage)
	}
	m.Requires = slices.Clone(m.Requires)
	m.Permissions = slices.Clone(m.Permissions)
	return m
}

// WASM returns a copy of the validated guest module bytes.
func (p *Package) WASM() []byte { return bytes.Clone(p.wasm) }

// README returns the validated package documentation.
func (p *Package) README() string { return string(p.readme) }

// Digest returns the package content digest.
func (p *Package) Digest() [32]byte { return p.digest }

// AssetNames returns validated asset paths in deterministic order.
func (p *Package) AssetNames() []string { return slices.Sorted(maps.Keys(p.assets)) }

// Asset returns a copy of one validated plugin asset.
func (p *Package) Asset(name string) ([]byte, error) {
	if !validPath(name) {
		return nil, fs.ErrInvalid
	}
	content, ok := p.assets[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return bytes.Clone(content), nil
}

// Read validates paths, types, decompression limits, CRCs, and manifest fields
// before returning any package. Asset names are relative to assets/.
func read(data []byte) (*Package, error) {
	if len(data) == 0 || len(data) > MaxArchiveBytes {
		return nil, errors.New("plugin archive exceeds size limit or is empty")
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("read plugin archive: %w", err)
	}
	if len(archive.File) > MaxFiles {
		return nil, errors.New("too many plugin archive entries")
	}

	files := make(map[string][]byte)
	seen := make(map[string]bool)
	total := 0

	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		if !validPath(name) || seen[name] {
			return nil, fmt.Errorf("invalid or duplicate plugin path %q", file.Name)
		}
		seen[name] = true
		if kind := file.Mode().Type(); kind != 0 && kind != fs.ModeDir {
			return nil, fmt.Errorf("plugin entry %q is not a regular file or directory", name)
		}
		if file.FileInfo().IsDir() {
			if !strings.HasSuffix(file.Name, "/") || file.UncompressedSize64 != 0 {
				return nil, fmt.Errorf("invalid plugin directory %q", name)
			}
			if name != "assets" && !strings.HasPrefix(name, "assets/") {
				return nil, fmt.Errorf("unsupported plugin directory %q", name)
			}
			continue
		}
		limit, err := entryLimit(name)
		if err != nil {
			return nil, err
		}
		content, err := readEntry(file, min(limit, MaxExpandedBytes-total))
		if err != nil {
			return nil, err
		}
		total += len(content)
		files[name] = content
	}

	// A file cannot also be an ancestor directory of another entry.
	for name := range seen {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if _, exists := files[parent]; exists {
				return nil, fmt.Errorf("plugin path %q has a file as parent", name)
			}
		}
	}

	manifest, err := ParseManifest(files["plugin.yaml"])
	if err != nil {
		return nil, err
	}

	wasm := files["plugin.wasm"]
	if manifest.RequiresWASM() {
		if !hasWASMHeader(wasm) {
			return nil, errors.New("missing or invalid plugin.wasm")
		}
	} else if len(wasm) != 0 && !hasWASMHeader(wasm) {
		return nil, errors.New("invalid plugin.wasm")
	}

	readme := files["README.md"]
	if len(bytes.TrimSpace(readme)) == 0 {
		return nil, errors.New("missing or empty README.md")
	}

	assets := make(map[string][]byte)
	for name, content := range files {
		if asset, ok := strings.CutPrefix(name, "assets/"); ok {
			assets[asset] = content
		}
	}

	for _, module := range manifest.Modules {
		for _, name := range []string{module.JavaScript, module.CSS, module.Asset} {
			data, ok := assets[name]
			if !ok {
				return nil, fmt.Errorf("missing module asset %s", name)
			}
			if module.Type == "icon-resource" && name == module.Asset {
				if _, err := ParseIconResource(data); err != nil {
					return nil, fmt.Errorf("icon resource %s: %w", module.ID, err)
				}
			}
		}
	}

	return &Package{manifest: manifest, wasm: wasm, assets: assets, readme: bytes.Clone(readme), digest: sha256.Sum256(data)}, nil
}

// hasWASMHeader reports whether data starts with the WebAssembly magic and version bytes.
func hasWASMHeader(data []byte) bool {
	header := []byte{'\x00', 'a', 's', 'm', 1, 0, 0, 0}
	return len(data) >= len(header) && bytes.Equal(data[:len(header)], header)
}

// validPath reports whether an archive path is safe and canonical.
func validPath(name string) bool {
	return len(name) <= 512 && name != "." && fs.ValidPath(name) && !strings.ContainsAny(name, "\\:\x00")
}

// entryLimit returns the maximum decompressed size allowed for one archive entry.
func entryLimit(name string) (int, error) {
	switch {
	case name == "plugin.yaml":
		return MaxManifestBytes, nil
	case name == "README.md":
		return MaxREADMEBytes, nil
	case name == "plugin.wasm":
		return MaxWASMBytes, nil
	case strings.HasPrefix(name, "assets/"):
		return MaxAssetBytes, nil
	default:
		return 0, fmt.Errorf("unsupported plugin entry %q", name)
	}
}

// readEntry reads one ZIP entry while enforcing size and integrity limits.
func readEntry(file *zip.File, limit int) ([]byte, error) {
	if file.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("plugin entry %q exceeds size limit", file.Name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}

	defer func() { _ = reader.Close() }()
	content, err := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	if err != nil {
		return nil, fmt.Errorf("read plugin entry %q: %w", file.Name, err)
	}
	if len(content) > limit {
		return nil, fmt.Errorf("plugin entry %q exceeds size limit", file.Name)
	}

	return content, nil
}

// ParseManifest strictly decodes and validates plugin manifest bytes.
func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if len(data) == 0 || len(data) > MaxManifestBytes {
		return manifest, errors.New("plugin manifest exceeds size limit or is empty")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("read plugin manifest: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return manifest, errors.New("plugin manifest must contain exactly one YAML document")
	}

	return manifest, manifest.Validate()
}

var (
	identifier = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	version    = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
)

// Validate checks manifest identity, permissions, modules, and dependency declarations.
func (m Manifest) Validate() error {
	if m.APIVersion != api.Version {
		return fmt.Errorf("unsupported plugin API version %d", m.APIVersion)
	}
	if !validPluginIdentity(m) {
		return errors.New("invalid plugin identity or version")
	}
	if len(m.Provider) > 128 {
		return errors.New("plugin provider is too long")
	}
	if len(m.Description) > 4096 {
		return errors.New("plugin description is too long")
	}

	permissions := make(map[string]bool)
	for _, permission := range m.Permissions {
		if !api.ValidPermission(permission) || permissions[permission] {
			return errors.New("invalid or duplicate plugin permission")
		}
		permissions[permission] = true
	}
	if len(m.Modules) == 0 || len(m.Modules) > 32 {
		return errors.New("plugin must declare between 1 and 32 modules")
	}

	names := make(map[string]bool)
	for _, module := range m.Modules {
		if !identifier.MatchString(module.ID) || names[module.ID] {
			return fmt.Errorf("invalid or duplicate module ID %q", module.ID)
		}
		names[module.ID] = true
		if !validModule(module) || !modulePermissionsAllowed(module, permissions) {
			return fmt.Errorf("unsupported plugin module %q", module.ID)
		}
	}

	if err := validateModuleDependencies(m.Modules, names); err != nil {
		return err
	}
	if err := validateModuleReferences(m.Modules); err != nil {
		return err
	}

	dependencies := make(map[string]bool)
	for _, id := range m.Requires {
		if !identifier.MatchString(id) || id == m.ID || dependencies[id] {
			return fmt.Errorf("invalid plugin dependency %q", id)
		}
		dependencies[id] = true
	}

	return nil
}

// validPluginIdentity reports whether the manifest identity fields are well formed.
func validPluginIdentity(m Manifest) bool {
	return identifier.MatchString(m.ID) &&
		strings.TrimSpace(m.Name) != "" &&
		len(m.Name) <= 128 &&
		version.MatchString(m.Version)
}

// modulePermissionsAllowed reports whether the manifest grants permissions required by a module type.
func modulePermissionsAllowed(m Module, permissions map[string]bool) bool {
	if m.Type == "browser-module" && !permissions["browser:render"] {
		return false
	}
	if m.Capability == "" {
		return true
	}
	permission, known := api.PermissionFor(m.Capability)
	return known && (permission == "" || permissions[permission])
}

// validModule validates one declared module type, stage, and identifier.
func validModule(m Module) bool {
	if !validModuleFields(m) {
		return false
	}

	switch m.Type {
	case "markdown-syntax":
		return validMarkdownSyntaxModule(m)
	case "code-highlighter":
		return validCodeHighlighterModule(m)
	case "settings":
		return validSettingsModule(m)
	case "content-style":
		return validContentStyleModule(m)
	case "render-policy":
		return validRenderPolicyModule(m)
	case "browser-module":
		return validBrowserModule(m)
	case "renderer-extension":
		return validRendererModule(m)
	case "macro":
		return validMacroModule(m)
	case "admin-resource":
		return validAdminResourceModule(m)
	case "content-substitution":
		return validContentSubstitutionModule(m)
	case "editor-completion":
		return validEditorCompletionModule(m)
	case "editor-insert":
		return validEditorInsertModule(m)
	case "icon-resource":
		return validIconResourceModule(m)
	default:
		return false
	}
}

// validModuleFields rejects fields that are only meaningful for another module type.
func validModuleFields(m Module) bool {
	if !validUsageRules(m) {
		return false
	}
	if m.Type != "browser-module" && m.JavaScript != "" {
		return false
	}
	if m.Type != "browser-module" && m.Type != "content-style" && m.Type != "code-highlighter" && m.CSS != "" {
		return false
	}
	if m.Type != "icon-resource" && m.Asset != "" {
		return false
	}
	if m.Type != "markdown-syntax" && m.Syntax != "" {
		return false
	}
	if m.Type != "render-policy" && m.Policy != "" {
		return false
	}
	if m.Type != "settings" && m.Type != "admin-resource" && m.Type != "editor-insert" && m.Description != "" {
		return false
	}
	if m.Type != "settings" && len(m.Requires) != 0 {
		return false
	}
	if m.Type != "renderer-extension" && m.Type != "content-substitution" && m.Priority != 0 {
		return false
	}
	if m.Type != "admin-resource" && len(m.Fields) != 0 {
		return false
	}
	if m.Type != "content-substitution" && m.Type != "editor-completion" && m.Resource != "" {
		return false
	}
	if m.Type != "content-substitution" && m.Prefix != "" {
		return false
	}
	if m.Type != "content-substitution" && m.ValueField != "" {
		return false
	}
	if m.Type != "content-substitution" && m.Type != "editor-completion" && m.LabelField != "" {
		return false
	}
	if m.Type != "content-substitution" && m.Type != "editor-completion" && m.DetailField != "" {
		return false
	}
	if m.Type != "editor-completion" && m.Trigger != "" {
		return false
	}
	if m.Type != "editor-completion" && m.Replacement != "" {
		return false
	}
	if m.Type != "editor-insert" && m.Markdown != "" {
		return false
	}
	if m.Type != "editor-insert" && m.Inline {
		return false
	}
	if m.Type != "content-substitution" && (m.Inspect || m.Export) {
		return false
	}
	return true
}

// validUsageRules validates bounded, declarative source selectors.
func validUsageRules(m Module) bool {
	if len(m.Usage) == 0 {
		return true
	}
	switch m.Type {
	case "renderer-extension", "macro", "markdown-syntax", "code-highlighter":
	default:
		return false
	}
	if len(m.Usage) > 8 {
		return false
	}
	for _, rule := range m.Usage {
		selectors := 0
		if rule.Contains != "" {
			selectors++
			if len(rule.Contains) > 128 || !utf8.ValidString(rule.Contains) || strings.ContainsRune(rule.Contains, '\x00') {
				return false
			}
		}
		if rule.Fence != "" {
			selectors++
			if rule.Fence != "*" && !identifier.MatchString(rule.Fence) {
				return false
			}
		}
		if rule.Macro != "" {
			selectors++
			if !identifier.MatchString(rule.Macro) {
				return false
			}
		}
		if rule.Substitution != "" {
			selectors++
			if !identifier.MatchString(rule.Substitution) {
				return false
			}
		}
		if selectors != 1 {
			return false
		}
	}
	return true
}

// validMarkdownSyntaxModule validates fields specific to a Markdown syntax declaration.
func validMarkdownSyntaxModule(m Module) bool {
	return api.ValidSyntax(m.Syntax) && m.Stage == "" && m.Name == "" && m.Capability == ""
}

// validCodeHighlighterModule validates an exclusive fenced-code highlighter declaration.
func validCodeHighlighterModule(m Module) bool {
	validCSS := m.CSS == "" || (validPath(m.CSS) && strings.HasSuffix(m.CSS, ".css"))
	return m.Stage == "" && m.Name == "" && m.Capability == "" && validCSS
}

// validSettingsModule validates fields specific to a settings declaration.
func validSettingsModule(m Module) bool {
	return m.Stage == "" && m.Capability == "" && len(m.Name) > 0 && len(m.Name) <= 128 && len(m.Description) <= 1024
}

// validContentStyleModule validates a stylesheet scoped to rendered page content.
func validContentStyleModule(m Module) bool {
	return m.Stage == "" && m.Name == "" && m.Capability == "" &&
		validPath(m.CSS) && strings.HasSuffix(m.CSS, ".css")
}

// validRenderPolicyModule validates a host rendering-policy contribution.
func validRenderPolicyModule(m Module) bool {
	return m.Stage == "" && m.Name == "" && m.Capability == "" && api.ValidRenderPolicy(m.Policy)
}

// validBrowserModule validates fields specific to an isolated browser module.
func validBrowserModule(m Module) bool {
	validJavaScript := validPath(m.JavaScript) && strings.HasSuffix(m.JavaScript, ".js")
	validCSS := m.CSS == "" || (validPath(m.CSS) && strings.HasSuffix(m.CSS, ".css"))

	return m.Stage == "" && m.Name == "" && m.Capability == "" && validJavaScript && validCSS
}

// validRendererModule validates fields specific to a renderer pipeline declaration.
func validRendererModule(m Module) bool {
	validStage := m.Stage == "preprocess" || m.Stage == "postprocess" || m.Stage == "content-preprocess"
	if m.Stage != "content-preprocess" && m.Priority != 0 {
		return false
	}
	return validStage && m.Name == ""
}

// validMacroModule validates fields specific to a macro declaration.
func validMacroModule(m Module) bool {
	if m.Capability != "" {
		if _, ok := api.PermissionFor(m.Capability); !ok {
			return false
		}
	}

	return identifier.MatchString(m.Name) && m.Stage == ""
}

// validAdminResourceModule validates one host-rendered plugin record collection.
func validAdminResourceModule(m Module) bool {
	if m.Stage != "" || m.Capability != "" || len(m.ID) > 64 || len(m.Name) == 0 || len(m.Name) > 128 || len(m.Description) > 1024 || len(m.Fields) == 0 || len(m.Fields) > 16 {
		return false
	}
	seen := make(map[string]bool)
	keys := 0
	for _, field := range m.Fields {
		if !identifier.MatchString(field.ID) || seen[field.ID] || strings.TrimSpace(field.Name) == "" || len(field.Name) > 128 {
			return false
		}
		seen[field.ID] = true
		if field.Type != "text" && field.Type != "textarea" {
			return false
		}
		if field.MaxBytes < 0 || field.MaxBytes > 64<<10 {
			return false
		}
		if field.Key {
			keys++
			if field.Type != "text" {
				return false
			}
		}
	}
	return keys == 1
}

// validContentSubstitutionModule validates a resource-backed inline Markdown substitution.
func validContentSubstitutionModule(m Module) bool {
	return m.Stage == "" && m.Name == "" && m.Capability == "" && identifier.MatchString(m.Resource) &&
		identifier.MatchString(m.Prefix) && identifier.MatchString(m.ValueField) &&
		(m.LabelField == "" || identifier.MatchString(m.LabelField)) &&
		(m.DetailField == "" || identifier.MatchString(m.DetailField)) &&
		m.Priority >= -10000 && m.Priority <= 10000
}

// validEditorCompletionModule validates a resource-backed completion provider.
func validEditorCompletionModule(m Module) bool {
	return m.Stage == "" && m.Name == "" && m.Capability == "" && identifier.MatchString(m.Resource) &&
		len(m.Trigger) > 0 && len(m.Trigger) <= 16 && len(m.Replacement) > 0 && len(m.Replacement) <= 512 &&
		identifier.MatchString(m.LabelField) && (m.DetailField == "" || identifier.MatchString(m.DetailField))
}

// validEditorInsertModule validates one static editor insertion action.
func validEditorInsertModule(m Module) bool {
	return m.Stage == "" && m.Capability == "" && strings.TrimSpace(m.Name) != "" && len(m.Name) <= 128 &&
		len(m.Description) <= 1024 && len(m.Markdown) > 0 && len(m.Markdown) <= 4096
}

// validIconResourceModule validates one declarative icon-pack contribution.
func validIconResourceModule(m Module) bool {
	name := strings.TrimSpace(m.Name)
	return m.Stage == "" && m.Capability == "" && validIconText(name, 128) &&
		validPath(m.Asset) && strings.HasSuffix(m.Asset, ".json")
}

// validateModuleReferences checks relationships between declarative modules in one package.
func validateModuleReferences(modules []Module) error {
	byID := make(map[string]Module, len(modules))
	for _, module := range modules {
		byID[module.ID] = module
	}
	for _, module := range modules {
		if module.Resource == "" {
			continue
		}
		resource, ok := byID[module.Resource]
		if !ok || resource.Type != "admin-resource" {
			return fmt.Errorf("module %s references unknown admin resource %s", module.ID, module.Resource)
		}
		fields := make(map[string]bool, len(resource.Fields))
		for _, field := range resource.Fields {
			fields[field.ID] = true
		}
		for _, field := range []string{module.ValueField, module.LabelField, module.DetailField} {
			if field != "" && !fields[field] {
				return fmt.Errorf("module %s references unknown resource field %s", module.ID, field)
			}
		}
		if module.Type == "editor-completion" {
			key := resourceKeyField(resource)
			if !strings.Contains(module.Replacement, "${"+key.ID+"}") {
				return fmt.Errorf("editor completion %s replacement must contain ${%s}", module.ID, key.ID)
			}
		}
	}
	return nil
}

// resourceKeyField returns the validated unique key field for a resource module.
func resourceKeyField(module Module) ResourceField {
	for _, field := range module.Fields {
		if field.Key {
			return field
		}
	}
	return ResourceField{}
}

// validateModuleDependencies validates settings-module dependency references and cycles.
func validateModuleDependencies(modules []Module, names map[string]bool) error {
	for _, module := range modules {
		seen := map[string]bool{}
		for _, dependency := range module.Requires {
			if !names[dependency] || dependency == module.ID || seen[dependency] {
				return fmt.Errorf("invalid module dependency %q", dependency)
			}
			seen[dependency] = true
		}
	}
	// Detect cycles before a package can reach the registry.
	visiting, done := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return false
		}
		if done[id] {
			return true
		}
		visiting[id] = true
		for _, module := range modules {
			if module.ID == id {
				for _, dep := range module.Requires {
					if !visit(dep) {
						return false
					}
				}
			}
		}
		visiting[id] = false
		done[id] = true
		return true
	}
	for id := range names {
		if !visit(id) {
			return errors.New("module dependency cycle")
		}
	}

	return nil
}
