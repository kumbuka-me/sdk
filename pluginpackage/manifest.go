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
	// Icon is the optional host icon used for plugin identity in administration.
	Icon string `yaml:"icon,omitempty"`
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
		case "renderer-extension", "macro", "code-highlighter", "widget", "exporter", "admin-action":
			return true
		}
	}

	return false
}

// ConfigurationField declares one bounded field in host-rendered plugin configuration.
type ConfigurationField struct {
	// ID identifies the field within its settings group or resource record.
	ID string `yaml:"id"`
	// Name is the human-readable field label.
	Name string `yaml:"name"`
	// Type selects a bounded host-rendered input control.
	Type string `yaml:"type"`
	// Required reports whether the field may be empty.
	Required bool `yaml:"required,omitempty"`
	// Key identifies the unique record key field for repeatable admin resources.
	Key bool `yaml:"key,omitempty"`
	// MaxBytes bounds the UTF-8 encoded field value. Zero selects a host default.
	MaxBytes int `yaml:"max_bytes,omitempty"`
	// Default is the value used before an administrator saves an explicit value.
	Default string `yaml:"default,omitempty"`
	// Options contains the allowed values for a select field.
	Options []string `yaml:"options,omitempty"`
	// MaxItems bounds the number of rows accepted by a list field. Zero selects a host default.
	MaxItems int `yaml:"max_items,omitempty"`
	// Columns declares the structured columns rendered for each row of a list field.
	Columns []ConfigurationField `yaml:"columns,omitempty"`
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
	// Fields declares a bounded schema for structured settings or admin-resource records.
	Fields []ConfigurationField `yaml:"fields,omitempty"`
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
	// AllowedGroups bounds the toolbar groups an administrator may select.
	AllowedGroups []string `yaml:"allowed_groups,omitempty"`
	// Children lists editor-insert module IDs owned and ordered by an editor-menu.
	Children []string `yaml:"children,omitempty"`
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
	if manifest.Icon != "" && !identifier.MatchString(manifest.Icon) {
		return errors.New("invalid plugin icon")
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
	case "admin-action":
		return validAdminActionModule(m)
	case "content-substitution":
		return validContentSubstitutionModule(m)
	case "editor-completion":
		return validEditorCompletionModule(m)
	case "editor-insert":
		return validEditorInsertModule(m)
	case "editor-menu":
		return validEditorMenuModule(m)
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
		validModuleConfigurationFields(module) &&
		validModuleEditorFields(module)
}

// validModuleRenderFields checks fields used by rendering and presentation module types.
func validModuleRenderFields(module Module) bool {
	if module.JavaScript != "" && module.Type != "browser-module" {
		return false
	}
	if hasWidgetLayoutFields(module) && module.Type != "widget" {
		return false
	}
	if module.Order != 0 && !supportsModuleOrder(module.Type) {
		return false
	}
	if module.CSS != "" && !supportsCSS(module.Type) {
		return false
	}
	if module.Asset != "" && module.Type != "icon-resource" {
		return false
	}
	if module.Syntax != "" && module.Type != "markdown-syntax" {
		return false
	}
	if module.Policy != "" && module.Type != "render-policy" {
		return false
	}
	if module.Description != "" && !supportsDescription(module.Type) {
		return false
	}
	if len(module.Requires) != 0 && module.Type != "settings" {
		return false
	}
	return true
}

// hasWidgetLayoutFields reports whether a module declares widget-only surface or width metadata.
func hasWidgetLayoutFields(module Module) bool {
	return module.Surface != "" || module.Width != ""
}

// supportsModuleOrder reports whether a module type participates in ordered host presentation.
func supportsModuleOrder(moduleType string) bool {
	switch moduleType {
	case "widget", "page-action", "exporter", "editor-insert", "editor-menu":
		return true
	default:
		return false
	}
}

// supportsCSS reports whether a module type may declare a stylesheet asset.
func supportsCSS(moduleType string) bool {
	switch moduleType {
	case "browser-module", "content-style", "code-highlighter":
		return true
	default:
		return false
	}
}

// supportsDescription reports whether a module type may declare administrator-facing descriptive text.
func supportsDescription(moduleType string) bool {
	switch moduleType {
	case "settings", "admin-resource", "admin-action", "editor-insert", "editor-menu", "page-action", "exporter":
		return true
	default:
		return false
	}
}

// validModuleConfigurationFields checks structured configuration, substitution, and priority fields.
func validModuleConfigurationFields(module Module) bool {
	if module.Priority != 0 && !supportsPriority(module.Type) {
		return false
	}
	if len(module.Fields) != 0 && !supportsConfigurationFields(module.Type) {
		return false
	}
	if module.Resource != "" && !supportsResourceReference(module.Type) {
		return false
	}
	if hasSubstitutionFields(module) && module.Type != "content-substitution" {
		return false
	}
	if hasResourcePresentationFields(module) && !supportsResourcePresentationFields(module.Type) {
		return false
	}
	return true
}

// supportsPriority reports whether a module type may declare pipeline priority.
func supportsPriority(moduleType string) bool {
	return moduleType == "renderer-extension" || moduleType == "content-substitution"
}

// supportsConfigurationFields reports whether a module type may declare typed configuration fields.
func supportsConfigurationFields(moduleType string) bool {
	return moduleType == "admin-resource" || moduleType == "settings"
}

// supportsResourceReference reports whether a module type may refer to an admin resource.
func supportsResourceReference(moduleType string) bool {
	return moduleType == "content-substitution" || moduleType == "editor-completion"
}

// hasSubstitutionFields reports whether a module declares content-substitution-only fields.
func hasSubstitutionFields(module Module) bool {
	return module.Prefix != "" || module.ValueField != "" || module.Inspect || module.Export
}

// hasResourcePresentationFields reports whether a module declares resource label or detail fields.
func hasResourcePresentationFields(module Module) bool {
	return module.LabelField != "" || module.DetailField != ""
}

// supportsResourcePresentationFields reports whether a module type may declare resource presentation fields.
func supportsResourcePresentationFields(moduleType string) bool {
	return moduleType == "content-substitution" || moduleType == "editor-completion"
}

// validModuleEditorFields checks fields contributed by editor completion and insert modules.
func validModuleEditorFields(module Module) bool {
	if hasCompletionFields(module) && module.Type != "editor-completion" {
		return false
	}
	if hasEditorInsertFields(module) && module.Type != "editor-insert" {
		return false
	}
	if module.Group != "" && !supportsToolbarPlacement(module.Type) {
		return false
	}
	if hasToolbarCollectionFields(module) && !supportsToolbarPlacement(module.Type) {
		return false
	}
	if module.Icon != "" && !supportsModuleIcon(module.Type) {
		return false
	}
	if hasPageActionFields(module) && module.Type != "page-action" {
		return false
	}
	return true
}

// hasCompletionFields reports whether a module declares editor-completion-only fields.
func hasCompletionFields(module Module) bool {
	return module.Trigger != "" || module.Replacement != ""
}

// hasEditorInsertFields reports whether a module declares editor-insert-only content fields.
func hasEditorInsertFields(module Module) bool {
	return module.Markdown != "" || module.Suffix != "" || module.Placeholder != "" || module.Mode != "" || module.Inline
}

// supportsToolbarPlacement reports whether a module type may participate in editor toolbar placement.
func supportsToolbarPlacement(moduleType string) bool {
	return moduleType == "editor-insert" || moduleType == "editor-menu"
}

// hasToolbarCollectionFields reports whether a module declares toolbar group choices or child actions.
func hasToolbarCollectionFields(module Module) bool {
	return len(module.AllowedGroups) != 0 || len(module.Children) != 0
}

// supportsModuleIcon reports whether a module type may declare a host-rendered icon.
func supportsModuleIcon(moduleType string) bool {
	switch moduleType {
	case "editor-insert", "editor-menu", "page-action", "exporter", "admin-action":
		return true
	default:
		return false
	}
}

// hasPageActionFields reports whether a module declares page-action-only navigation fields.
func hasPageActionFields(module Module) bool {
	return module.Kind != "" || module.URL != ""
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
		if !validUsageContains(rule.Contains) {
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

// validUsageContains reports whether a literal usage selector is bounded, valid UTF-8 text.
func validUsageContains(value string) bool {
	return len(value) <= 128 && utf8.ValidString(value) && !strings.ContainsRune(value, '\x00')
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

// validSettingsModule validates either one boolean feature toggle or one typed singleton settings group.
func validSettingsModule(m Module) bool {
	if !validSettingsMetadata(m) {
		return false
	}
	if len(m.Fields) == 0 {
		return true
	}
	if len(m.Requires) != 0 || len(m.Fields) > 16 {
		return false
	}

	seen := make(map[string]bool, len(m.Fields))
	for _, field := range m.Fields {
		if field.Key || field.Type == "list" || !validConfigurationField(field, seen) {
			return false
		}
		seen[field.ID] = true
	}

	return true
}

// validSettingsMetadata reports whether settings module metadata is supported and bounded.
func validSettingsMetadata(module Module) bool {
	return module.Stage == "" && module.Capability == "" && validModuleDisplayMetadata(module)
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
	if !validAdminResourceMetadata(module) {
		return false
	}

	seen := make(map[string]bool, len(module.Fields))
	keys := 0
	for _, field := range module.Fields {
		if !validConfigurationField(field, seen) {
			return false
		}
		seen[field.ID] = true
		if field.Key {
			keys++
		}
	}
	return keys == 1
}

// validAdminResourceMetadata reports whether resource metadata and field counts are supported.
func validAdminResourceMetadata(module Module) bool {
	return module.Stage == "" &&
		module.Capability == "" &&
		len(module.ID) <= 64 &&
		validModuleDisplayMetadata(module) &&
		len(module.Fields) > 0 &&
		len(module.Fields) <= 16
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
	if !validEditorInsertMetadata(m) {
		return false
	}

	mode := m.Mode
	if mode == "" {
		mode = "insert"
	}
	if !validEditorInsertPlacement(m) {
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

// validEditorInsertMetadata reports whether editor action metadata and source bounds are supported.
func validEditorInsertMetadata(module Module) bool {
	return module.Stage == "" &&
		module.Capability == "" &&
		validModuleDisplayMetadata(module) &&
		len(module.Markdown) > 0 &&
		len(module.Markdown) <= 4096 &&
		len(module.Suffix) <= 4096 &&
		len(module.Placeholder) <= 256 &&
		validOptionalModuleIcon(module.Icon)
}

// validEditorInsertPlacement reports whether an editor action has a supported toolbar position.
func validEditorInsertPlacement(module Module) bool {
	return validToolbarPlacement(module.Group, module.AllowedGroups) &&
		len(module.Children) == 0 &&
		module.Order >= -1000 &&
		module.Order <= 1000
}

// validEditorMenuModule validates one host-rendered plugin-owned toolbar submenu.
func validEditorMenuModule(m Module) bool {
	return m.Stage == "" &&
		m.Capability == "" &&
		validModuleDisplayMetadata(m) &&
		validOptionalModuleIcon(m.Icon) &&
		validToolbarPlacement(m.Group, m.AllowedGroups) &&
		len(m.Children) >= 2 &&
		len(m.Children) <= 16 &&
		validModuleOrder(m.Order)
}

// validToolbarPlacement validates a preferred group and its bounded unique administrator choices.
func validToolbarPlacement(preferred string, allowed []string) bool {
	if preferred == "" {
		preferred = "insert"
	}
	if !validToolbarGroup(preferred) || len(allowed) > 5 {
		return false
	}
	seen := map[string]bool{}
	for _, group := range allowed {
		if !validToolbarGroup(group) || seen[group] {
			return false
		}
		seen[group] = true
	}
	return len(allowed) == 0 || seen[preferred]
}

// validToolbarGroup reports whether a group is one of the stable host-owned toolbar slots.
func validToolbarGroup(group string) bool {
	switch group {
	case "text", "blocks", "insert", "tools", "plugins":
		return true
	default:
		return false
	}
}

// validModuleDisplayMetadata reports whether administrator-facing module text is non-empty and bounded.
func validModuleDisplayMetadata(module Module) bool {
	return strings.TrimSpace(module.Name) != "" && len(module.Name) <= 128 && len(module.Description) <= 1024
}

// validOptionalModuleIcon reports whether an optional module icon has a valid identifier.
func validOptionalModuleIcon(icon string) bool {
	return icon == "" || identifier.MatchString(icon)
}

// validModuleOrder reports whether a host presentation order is within the supported range.
func validModuleOrder(order int) bool {
	return order >= -1000 && order <= 1000
}

// validPageActionKind reports whether kind selects a supported page action presentation.
func validPageActionKind(kind string) bool {
	return kind == "" || kind == "link" || kind == "dialog"
}

// validWidgetModule validates one executable widget contribution.
func validWidgetModule(m Module) bool {
	return m.Stage == "" && m.Name == "" && m.Description == "" && m.Capability == "" &&
		api.ValidWidgetSurface(m.Surface) && api.ValidWidgetWidth(m.Width) && validModuleOrder(m.Order)
}

// validAdminActionModule validates one executable administrator action.
func validAdminActionModule(m Module) bool {
	return m.Stage == "" &&
		m.Capability == "" &&
		m.Order == 0 &&
		validModuleDisplayMetadata(m) &&
		validOptionalModuleIcon(m.Icon)
}

// validPageActionModule validates one host-rendered current-page navigation action.
func validPageActionModule(m Module) bool {
	return m.Stage == "" &&
		m.Capability == "" &&
		validModuleDisplayMetadata(m) &&
		validOptionalModuleIcon(m.Icon) &&
		validPageActionKind(m.Kind) &&
		validModuleOrder(m.Order) &&
		validPageActionURL(m.URL)
}

// validExporterModule validates one executable page export contribution.
func validExporterModule(m Module) bool {
	return m.Stage == "" &&
		m.Capability == "" &&
		validModuleDisplayMetadata(m) &&
		validOptionalModuleIcon(m.Icon) &&
		m.Kind == "" &&
		m.URL == "" &&
		validModuleOrder(m.Order)
}

// validPageActionURL accepts bounded local URL templates with page placeholders only.
func validPageActionURL(value string) bool {
	if !validPageActionURLShape(value) {
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

// validPageActionURLShape reports whether value has a bounded local-path URL shape.
func validPageActionURLShape(value string) bool {
	return len(value) > 0 &&
		len(value) <= 2048 &&
		strings.HasPrefix(value, "/") &&
		!strings.HasPrefix(value, "//") &&
		!strings.ContainsAny(value, "\\\r\n\t")
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
	menuChild := make(map[string]string)
	for _, module := range modules {
		byID[module.ID] = module
	}
	for _, module := range modules {
		if module.Type == "editor-menu" {
			seen := map[string]bool{}
			for _, childID := range module.Children {
				child, ok := byID[childID]
				if !validEditorMenuChild(child, ok, childID, seen, menuChild) {
					return fmt.Errorf("editor menu %s has invalid child %s", module.ID, childID)
				}
				seen[childID] = true
				menuChild[childID] = module.ID
			}
		}
		if module.Resource == "" {
			continue
		}
		if err := validateModuleReference(module, byID); err != nil {
			return err
		}
	}
	return nil
}

// validEditorMenuChild reports whether a menu child is a unique unowned editor-insert module.
func validEditorMenuChild(child Module, exists bool, childID string, seen map[string]bool, menuChild map[string]string) bool {
	return exists && child.Type == "editor-insert" && !seen[childID] && menuChild[childID] == ""
}

// validateModuleReference checks one resource-backed module against its declared resource.
func validateModuleReference(module Module, byID map[string]Module) error {
	resource, ok := byID[module.Resource]
	if !ok || resource.Type != "admin-resource" {
		return fmt.Errorf("module %s references unknown admin resource %s", module.ID, module.Resource)
	}

	fields := make(map[string]ConfigurationField, len(resource.Fields))
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
func resourceKeyField(module Module) ConfigurationField {
	for _, field := range module.Fields {
		if field.Key {
			return field
		}
	}
	return ConfigurationField{}
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
