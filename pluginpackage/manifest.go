package pluginpackage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/kumbuka-me/sdk/internal/api"
	"go.yaml.in/yaml/v3"
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
		case "renderer-extension", "macro", "code-highlighter", "widget", "exporter":
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
	// Key identifies the unique record key field. Exactly one text field must be the key.
	Key bool `yaml:"key,omitempty"`
	// MaxBytes bounds the UTF-8 encoded field value. Zero selects a host default.
	MaxBytes int `yaml:"max_bytes,omitempty"`
	// Default is the initial value shown when an administrator creates a record.
	Default string `yaml:"default,omitempty"`
	// Options contains the allowed values for a select field.
	Options []string `yaml:"options,omitempty"`
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
	// Surface selects the host placement for widget modules.
	Surface string `yaml:"surface,omitempty"`
	// Width is an optional host layout hint for widget modules.
	Width string `yaml:"width,omitempty"`
	// Order controls deterministic placement among widgets on the same surface.
	Order int `yaml:"order,omitempty"`
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
	// Policy declares a semantic rendering-policy marker shared with executable modules.
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
	// Markdown is the inserted source or opening text used by an editor-insert module.
	Markdown string `yaml:"markdown,omitempty"`
	// Suffix closes wrapped editor selections when Mode is wrap.
	Suffix string `yaml:"suffix,omitempty"`
	// Placeholder is used when an editor action has no selected text.
	Placeholder string `yaml:"placeholder,omitempty"`
	// Mode selects insert, wrap, or prefix-lines behavior for editor-insert modules.
	Mode string `yaml:"mode,omitempty"`
	// Group places an editor-insert action in insert, text, or blocks UI.
	Group string `yaml:"group,omitempty"`
	// Icon names an optional host icon for editor and page actions.
	Icon string `yaml:"icon,omitempty"`
	// Kind selects link or host-dialog behavior for page-action modules.
	Kind string `yaml:"kind,omitempty"`
	// URL is a local host path template used by page-action modules.
	URL string `yaml:"url,omitempty"`
	// Inline reports whether plain insertion should avoid block line breaks.
	Inline bool `yaml:"inline,omitempty"`
	// Usage declares cheap host-side source selectors for executable modules.
	Usage []UsageRule `yaml:"usage,omitempty"`
	// Inspect enables a reading-page inspector for content substitutions.
	Inspect bool `yaml:"inspect,omitempty"`
	// Export enables request-local export overrides for content substitutions.
	Export bool `yaml:"export,omitempty"`
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
	if err := validateManifestMetadata(m); err != nil {
		return err
	}
	permissions, err := validatePermissions(m.Permissions)
	if err != nil {
		return err
	}
	names, err := validateModules(m.Modules, permissions)
	if err != nil {
		return err
	}
	if err := validateModuleDependencies(m.Modules, names); err != nil {
		return err
	}
	if err := validateModuleReferences(m.Modules); err != nil {
		return err
	}
	return validatePluginDependencies(m.ID, m.Requires)
}

// validateManifestMetadata checks versioned identity and bounded descriptive fields.
func validateManifestMetadata(manifest Manifest) error {
	if manifest.APIVersion != api.Version {
		return fmt.Errorf("unsupported plugin API version %d", manifest.APIVersion)
	}
	if !validPluginIdentity(manifest) {
		return errors.New("invalid plugin identity or version")
	}
	if len(manifest.Provider) > 128 {
		return errors.New("plugin provider is too long")
	}
	if len(manifest.Description) > 4096 {
		return errors.New("plugin description is too long")
	}
	return nil
}

// validatePermissions returns the unique permission set declared by a manifest.
func validatePermissions(values []string) (map[string]bool, error) {
	permissions := make(map[string]bool, len(values))
	for _, permission := range values {
		if !api.ValidPermission(permission) || permissions[permission] {
			return nil, errors.New("invalid or duplicate plugin permission")
		}
		permissions[permission] = true
	}
	return permissions, nil
}

// validateModules validates module IDs, fields, and required host permissions.
func validateModules(modules []Module, permissions map[string]bool) (map[string]bool, error) {
	if len(modules) == 0 || len(modules) > 32 {
		return nil, errors.New("plugin must declare between 1 and 32 modules")
	}

	names := make(map[string]bool, len(modules))
	for _, module := range modules {
		if !identifier.MatchString(module.ID) || names[module.ID] {
			return nil, fmt.Errorf("invalid or duplicate module ID %q", module.ID)
		}
		names[module.ID] = true
		if !validModule(module) || !modulePermissionsAllowed(module, permissions) {
			return nil, fmt.Errorf("unsupported plugin module %q", module.ID)
		}
	}
	return names, nil
}

// validatePluginDependencies validates unique external plugin dependencies.
func validatePluginDependencies(pluginID string, values []string) error {
	dependencies := make(map[string]bool, len(values))
	for _, id := range values {
		if !identifier.MatchString(id) || id == pluginID || dependencies[id] {
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
	case "widget":
		return validWidgetModule(m)
	case "page-action":
		return validPageActionModule(m)
	case "exporter":
		return validExporterModule(m)
	default:
		return false
	}
}

// validModuleFields rejects fields that are only meaningful for another module type.
func validModuleFields(module Module) bool {
	return validUsageRules(module) &&
		validModuleRenderFields(module) &&
		validModuleResourceFields(module) &&
		validModuleEditorFields(module)
}

// validModuleRenderFields checks fields used by rendering and presentation module types.
func validModuleRenderFields(module Module) bool {
	if module.Type != "browser-module" && module.JavaScript != "" {
		return false
	}
	if module.Type != "widget" && module.Type != "page-action" && module.Type != "exporter" && (module.Surface != "" || module.Width != "" || module.Order != 0) {
		return false
	}
	if module.Type != "browser-module" && module.Type != "content-style" && module.Type != "code-highlighter" && module.CSS != "" {
		return false
	}
	if module.Type != "icon-resource" && module.Asset != "" {
		return false
	}
	if module.Type != "markdown-syntax" && module.Syntax != "" {
		return false
	}
	if module.Type != "render-policy" && module.Policy != "" {
		return false
	}
	if module.Type != "settings" && module.Type != "admin-resource" && module.Type != "editor-insert" && module.Type != "page-action" && module.Type != "exporter" && module.Description != "" {
		return false
	}
	if module.Type != "settings" && len(module.Requires) != 0 {
		return false
	}
	return true
}

// validModuleResourceFields checks resource, substitution, and priority fields.
func validModuleResourceFields(module Module) bool {
	if module.Type != "renderer-extension" && module.Type != "content-substitution" && module.Priority != 0 {
		return false
	}
	if module.Type != "admin-resource" && len(module.Fields) != 0 {
		return false
	}
	if module.Type != "content-substitution" && module.Type != "editor-completion" && module.Resource != "" {
		return false
	}
	if module.Type != "content-substitution" && (module.Prefix != "" || module.ValueField != "" || module.Inspect || module.Export) {
		return false
	}
	if module.Type != "content-substitution" && module.Type != "editor-completion" && (module.LabelField != "" || module.DetailField != "") {
		return false
	}
	return true
}

// validModuleEditorFields checks fields contributed by editor completion and insert modules.
func validModuleEditorFields(module Module) bool {
	if module.Type != "editor-completion" && (module.Trigger != "" || module.Replacement != "") {
		return false
	}
	if module.Type != "editor-insert" && (module.Markdown != "" || module.Suffix != "" || module.Placeholder != "" ||
		module.Mode != "" || module.Group != "" || module.Inline) {
		return false
	}
	if module.Type != "editor-insert" && module.Type != "page-action" && module.Type != "exporter" && module.Icon != "" {
		return false
	}
	if module.Type != "page-action" && (module.Kind != "" || module.URL != "") {
		return false
	}
	return true
}

// validUsageRules validates bounded, declarative source selectors.
func validUsageRules(module Module) bool {
	if len(module.Usage) == 0 {
		return true
	}
	if len(module.Usage) > 8 || !supportsUsageRules(module.Type) {
		return false
	}
	for _, rule := range module.Usage {
		if !validUsageRule(rule) {
			return false
		}
	}
	return true
}

// supportsUsageRules reports whether a module type may declare source selectors.
func supportsUsageRules(moduleType string) bool {
	switch moduleType {
	case "renderer-extension", "macro", "markdown-syntax", "code-highlighter", "content-style":
		return true
	default:
		return false
	}
}

// validUsageRule accepts exactly one valid source selector.
func validUsageRule(rule UsageRule) bool {
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
	return selectors == 1
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

// validRenderPolicyModule validates a semantic rendering-policy contribution.
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
func validAdminResourceModule(module Module) bool {
	if module.Stage != "" || module.Capability != "" || len(module.ID) > 64 ||
		len(module.Name) == 0 || len(module.Name) > 128 || len(module.Description) > 1024 ||
		len(module.Fields) == 0 || len(module.Fields) > 16 {
		return false
	}

	seen := make(map[string]bool, len(module.Fields))
	keys := 0
	for _, field := range module.Fields {
		if !validResourceField(field, seen) {
			return false
		}
		seen[field.ID] = true
		if field.Key {
			keys++
		}
	}
	return keys == 1
}

// validResourceField validates one bounded admin-resource field.
func validResourceField(field ResourceField, seen map[string]bool) bool {
	if !identifier.MatchString(field.ID) || seen[field.ID] || strings.TrimSpace(field.Name) == "" || len(field.Name) > 128 {
		return false
	}
	if field.MaxBytes < 0 || field.MaxBytes > 64<<10 || (field.Key && field.Type != "text") {
		return false
	}

	switch field.Type {
	case "text", "textarea", "url":
		return len(field.Options) == 0 && validResourceDefault(field)
	case "secret":
		return len(field.Options) == 0 && field.Default == "" && !field.Key
	case "boolean":
		return len(field.Options) == 0 && (field.Default == "" || field.Default == "true" || field.Default == "false") && !field.Key
	case "select":
		return validResourceOptions(field) && !field.Key
	default:
		return false
	}
}

// validResourceDefault validates a bounded default value for scalar resource fields.
func validResourceDefault(field ResourceField) bool {
	if !utf8.ValidString(field.Default) || strings.ContainsRune(field.Default, '\x00') {
		return false
	}
	limit := field.MaxBytes
	if limit == 0 {
		if field.Type == "textarea" {
			limit = 48 << 10
		} else {
			limit = 4096
		}
	}
	if len(field.Default) > limit {
		return false
	}
	if field.Type != "url" || field.Default == "" {
		return true
	}
	parsed, err := url.ParseRequestURI(field.Default)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

// validResourceOptions validates the bounded option set and optional default of a select field.
func validResourceOptions(field ResourceField) bool {
	if len(field.Options) == 0 || len(field.Options) > 32 {
		return false
	}
	seen := make(map[string]bool, len(field.Options))
	for _, option := range field.Options {
		if strings.TrimSpace(option) != option || option == "" || len(option) > 128 || !utf8.ValidString(option) || seen[option] {
			return false
		}
		seen[option] = true
	}
	return field.Default == "" || seen[field.Default]
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

// validEditorInsertModule validates one declarative editor action.
func validEditorInsertModule(m Module) bool {
	if m.Stage != "" || m.Capability != "" || strings.TrimSpace(m.Name) == "" || len(m.Name) > 128 ||
		len(m.Description) > 1024 || len(m.Markdown) == 0 || len(m.Markdown) > 4096 || len(m.Suffix) > 4096 ||
		len(m.Placeholder) > 256 || (m.Icon != "" && !identifier.MatchString(m.Icon)) {
		return false
	}

	mode := m.Mode
	if mode == "" {
		mode = "insert"
	}
	group := m.Group
	if group == "" {
		group = "insert"
	}
	if group != "insert" && group != "text" && group != "blocks" {
		return false
	}

	switch mode {
	case "insert":
		return m.Suffix == "" && m.Placeholder == ""
	case "wrap":
		return m.Suffix != "" && !m.Inline
	case "prefix-lines":
		return m.Suffix == "" && !m.Inline
	default:
		return false
	}
}

// validWidgetModule validates one executable widget contribution.
func validWidgetModule(m Module) bool {
	return m.Stage == "" && m.Name == "" && m.Description == "" && m.Capability == "" &&
		api.ValidWidgetSurface(m.Surface) && api.ValidWidgetWidth(m.Width) && m.Order >= -1000 && m.Order <= 1000
}

// validPageActionModule validates one host-rendered current-page navigation action.
func validPageActionModule(m Module) bool {
	return m.Stage == "" && m.Capability == "" && m.Surface == "" && m.Width == "" &&
		strings.TrimSpace(m.Name) != "" && len(m.Name) <= 128 && len(m.Description) <= 1024 &&
		(m.Icon == "" || identifier.MatchString(m.Icon)) && (m.Kind == "" || m.Kind == "link" || m.Kind == "dialog") &&
		m.Order >= -1000 && m.Order <= 1000 && validPageActionURL(m.URL)
}

// validExporterModule validates one executable page export contribution.
func validExporterModule(m Module) bool {
	return m.Stage == "" && m.Capability == "" && m.Surface == "" && m.Width == "" &&
		strings.TrimSpace(m.Name) != "" && len(m.Name) <= 128 && len(m.Description) <= 1024 &&
		(m.Icon == "" || identifier.MatchString(m.Icon)) && m.Kind == "" && m.URL == "" &&
		m.Order >= -1000 && m.Order <= 1000
}

// validPageActionURL accepts bounded local URL templates with page placeholders only.
func validPageActionURL(value string) bool {
	if len(value) == 0 || len(value) > 2048 || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") ||
		strings.ContainsAny(value, "\\\r\n\t") {
		return false
	}

	resolved := strings.ReplaceAll(value, "${slug}", "page")
	resolved = strings.ReplaceAll(resolved, "${id}", "1")
	if strings.Contains(resolved, "${") {
		return false
	}
	parsed, err := url.ParseRequestURI(resolved)
	return err == nil && parsed.Host == "" && !parsed.IsAbs()
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
		if err := validateModuleReference(module, byID); err != nil {
			return err
		}
	}
	return nil
}

// validateModuleReference checks one resource-backed module against its declared resource.
func validateModuleReference(module Module, byID map[string]Module) error {
	resource, ok := byID[module.Resource]
	if !ok || resource.Type != "admin-resource" {
		return fmt.Errorf("module %s references unknown admin resource %s", module.ID, module.Resource)
	}

	fields := make(map[string]ResourceField, len(resource.Fields))
	for _, field := range resource.Fields {
		fields[field.ID] = field
	}
	for _, fieldID := range []string{module.ValueField, module.LabelField, module.DetailField} {
		if fieldID == "" {
			continue
		}
		field, ok := fields[fieldID]
		if !ok {
			return fmt.Errorf("module %s references unknown resource field %s", module.ID, fieldID)
		}
		if field.Type == "secret" {
			return fmt.Errorf("module %s cannot expose secret resource field %s", module.ID, fieldID)
		}
	}
	if module.Type == "editor-completion" {
		key := resourceKeyField(resource)
		if !strings.Contains(module.Replacement, "${"+key.ID+"}") {
			return fmt.Errorf("editor completion %s replacement must contain ${%s}", module.ID, key.ID)
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
	requirements := make(map[string][]string, len(modules))
	for _, module := range modules {
		seen := make(map[string]bool, len(module.Requires))
		for _, dependency := range module.Requires {
			if !names[dependency] || dependency == module.ID || seen[dependency] {
				return fmt.Errorf("invalid module dependency %q", dependency)
			}
			seen[dependency] = true
		}
		requirements[module.ID] = module.Requires
	}
	return validateDependencyCycles(requirements)
}

// validateDependencyCycles rejects cycles in the module requirement graph.
func validateDependencyCycles(requirements map[string][]string) error {
	visiting := make(map[string]bool, len(requirements))
	done := make(map[string]bool, len(requirements))

	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return false
		}
		if done[id] {
			return true
		}

		visiting[id] = true
		for _, dependency := range requirements[id] {
			if !visit(dependency) {
				return false
			}
		}
		visiting[id] = false
		done[id] = true
		return true
	}

	for id := range requirements {
		if !visit(id) {
			return errors.New("module dependency cycle")
		}
	}
	return nil
}
