package pluginpackage

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testManifest = `api_version: 1
id: io.example.test
name: Test
version: 1.0.0
modules:
  - type: renderer-extension
    id: test
    stage: preprocess
permissions: []
`

// testArchive handles the test archive operation.
func testArchive(t *testing.T, manifest string, extras ...archiveEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entries := append([]archiveEntry{{"README.md", []byte("# Test\n"), 0644}, {"plugin.yaml", []byte(manifest), 0644}, {"plugin.wasm", []byte{0, 'a', 's', 'm', 1, 0, 0, 0}, 0644}}, extras...)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(entry.mode)
		writer, err := archive.CreateHeader(header)
		require.NoError(t, err)
		_, err = writer.Write(entry.data)
		require.NoError(t, err)
	}
	require.NoError(t, archive.Close())
	return buffer.Bytes()
}

func testArchiveWithoutWASM(t *testing.T, manifest string, extras ...archiveEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entries := append([]archiveEntry{{"README.md", []byte("# Test\n"), 0644}, {"plugin.yaml", []byte(manifest), 0644}}, extras...)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(entry.mode)
		writer, err := archive.CreateHeader(header)
		require.NoError(t, err)
		_, err = writer.Write(entry.data)
		require.NoError(t, err)
	}
	require.NoError(t, archive.Close())
	return buffer.Bytes()
}

// archiveEntry groups the state and data associated with archive entry.
type archiveEntry struct {
	// name stores the value associated with name.
	name string
	// data contains the ordered values associated with data.
	data []byte
	// mode stores the value associated with mode.
	mode fs.FileMode
}

// TestPackageReadAndDefensiveCopies verifies package read and defensive copies behavior.
func TestPackageReadAndDefensiveCopies(t *testing.T) {
	data := testArchive(t, testManifest, archiveEntry{"assets/plugin.css", []byte("body{}"), 0644})
	pkg, err := Read(data)
	require.NoError(t, err)
	assert.Equal(t, "io.example.test", pkg.Manifest().ID)
	assert.Equal(t, "# Test\n", pkg.README())
	assert.Equal(t, []string{"plugin.css"}, pkg.AssetNames())
	content, err := pkg.Asset("plugin.css")
	require.NoError(t, err)
	content[0] = '!'
	original, err := pkg.Asset("plugin.css")
	require.NoError(t, err)
	assert.Equal(t, "body{}", string(original))
	manifest := pkg.Manifest()
	manifest.Modules[0].ID = "changed"
	assert.Equal(t, "test", pkg.Manifest().Modules[0].ID)
	wasm := pkg.WASM()
	wasm[0] = 1
	assert.Equal(t, byte(0), pkg.WASM()[0])
	_, err = pkg.Asset("../plugin.wasm")
	assert.ErrorIs(t, err, fs.ErrInvalid)
	_, err = pkg.Asset("missing")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestDeclarativePackageDoesNotRequireWASM(t *testing.T) {
	manifest := `api_version: 1
id: io.example.syntax
name: Syntax
version: 1.0.0
modules:
  - type: markdown-syntax
    id: grammar
    syntax: strikethrough
permissions: []
`
	pkg, err := Read(testArchiveWithoutWASM(t, manifest))
	require.NoError(t, err)
	assert.False(t, pkg.Manifest().RequiresWASM())
	assert.Empty(t, pkg.WASM())

	_, err = Read(testArchiveWithoutWASM(t, testManifest))
	require.ErrorContains(t, err, "missing or invalid plugin.wasm")

	_, err = Read(testArchiveWithoutWASM(t, manifest, archiveEntry{"plugin.wasm", []byte("not wasm"), 0644}))
	require.ErrorContains(t, err, "invalid plugin.wasm")
}

// TestPackageRejectsUnsafeEntries verifies package rejects unsafe entries behavior.
func TestPackageRejectsUnsafeEntries(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "assets/../escape", "assets\\escape", "assets/a/../../escape", "C:/escape", "assets/./file", "other.txt", "plugin.yaml"} {
		t.Run(name, func(t *testing.T) {
			_, err := Read(testArchive(t, testManifest, archiveEntry{name, []byte("bad"), 0644}))
			require.Error(t, err)
		})
	}
	for _, mode := range []fs.FileMode{fs.ModeSymlink | 0644, fs.ModeNamedPipe | 0644} {
		_, err := Read(testArchive(t, testManifest, archiveEntry{"assets/link", []byte("/etc/passwd"), mode}))
		require.Error(t, err, "mode %v", mode)
	}
	_, err := Read(testArchive(t, testManifest, archiveEntry{"assets/link/", nil, fs.ModeSymlink | 0644}))
	require.Error(t, err)
	_, err = Read(testArchive(t, testManifest, archiveEntry{"assets/parent", nil, 0644}, archiveEntry{"assets/parent/child", nil, 0644}))
	require.Error(t, err)
}

// TestPackageRejectsUnsupportedManifest verifies package rejects unsupported manifest behavior.
func TestPackageRejectsUnsupportedManifest(t *testing.T) {
	for _, manifest := range []string{
		strings.Replace(testManifest, "api_version: 1", "api_version: 999", 1),
		strings.Replace(testManifest, "permissions: []", "permissions: [network]", 1),
		strings.Replace(testManifest, "stage: preprocess", "stage: native", 1),
		strings.Replace(testManifest, "renderer-extension", "database", 1),
		strings.Replace(testManifest, "1.0.0", "latest", 1),
		testManifest + "unknown: value\n",
		testManifest + "api_version: 1\n",
		testManifest + "---\nname: extra\n",
		testManifest + "requires: [io.example.test]\n",
	} {
		_, err := Read(testArchive(t, manifest))
		require.Error(t, err, manifest)
	}
}

// TestPackageRejectsExpansionAndEntryCountLimits verifies package rejects expansion and entry count limits behavior.
func TestPackageRejectsExpansionAndEntryCountLimits(t *testing.T) {
	_, err := Read(testArchive(t, testManifest, archiveEntry{"assets/large.css", bytes.Repeat([]byte{'x'}, MaxAssetBytes+1), 0644}))
	require.ErrorContains(t, err, "size limit")
	entries := make([]archiveEntry, MaxFiles)
	for i := range entries {
		entries[i] = archiveEntry{"assets/" + strings.Repeat("x", i+1), nil, 0644}
	}
	_, err = Read(testArchive(t, testManifest, entries...))
	require.ErrorContains(t, err, "too many")
	_, err = Read(make([]byte, MaxArchiveBytes+1))
	require.Error(t, err)
}

// TestManifestMacroAndPermissionValidation verifies manifest macro and permission validation behavior.
func TestManifestMacroAndPermissionValidation(t *testing.T) {
	manifest := strings.Replace(testManifest, "type: renderer-extension", "type: macro", 1)
	manifest = strings.Replace(manifest, "stage: preprocess", "name: pages\n    capability: pages.search", 1)
	manifest = strings.Replace(manifest, "permissions: []", "permissions: [pages:read]", 1)
	pkg, err := Read(testArchive(t, manifest))
	require.NoError(t, err)
	assert.Equal(t, "pages", pkg.Manifest().Modules[0].Name)
	for _, invalid := range []string{
		strings.Replace(manifest, "pages.search", "database.read", 1),
		strings.Replace(manifest, "name: pages", "name: ../pages", 1),
		strings.Replace(manifest, "permissions: [pages:read]", "permissions: [pages:read, pages:read]", 1),
		strings.Replace(manifest, "name: pages", "stage: preprocess\n    name: pages", 1),
	} {
		_, err := Read(testArchive(t, invalid))
		require.Error(t, err)
	}
}

func TestBrowserModulesRequireDeclaredPermissionAndPackagedAssets(t *testing.T) {
	manifest := `api_version: 1
id: io.example.browser
name: Browser
version: 1.0.0
modules:
  - type: browser-module
    id: diagrams
    javascript: plugin.js
    css: plugin.css
permissions: [browser:render]
`
	assets := []archiveEntry{{"assets/plugin.js", []byte("globalThis.kumbukaPlugin={}"), 0644}, {"assets/plugin.css", []byte("body{}"), 0644}}
	_, err := Read(testArchive(t, manifest, assets...))
	require.NoError(t, err)
	for _, invalid := range []string{
		strings.ReplaceAll(manifest, "[browser:render]", "[]"),
		strings.ReplaceAll(manifest, "javascript: plugin.js", "javascript: ../plugin.js"),
		strings.ReplaceAll(manifest, "javascript: plugin.js", "javascript: absent.js"),
		strings.ReplaceAll(manifest, "css: plugin.css", "css: absent.css"),
		strings.ReplaceAll(manifest, "type: browser-module", "type: renderer-extension\n    stage: postprocess"),
	} {
		_, err := Read(testArchive(t, invalid, assets...))
		require.Error(t, err)
	}
	_, err = Read(testArchive(t, manifest))
	require.Error(t, err)
}

func TestDeclarativeSyntaxAndSettingsDependencies(t *testing.T) {
	manifest := `api_version: 1
id: io.example.syntax
name: Syntax
version: 1.0.0
modules:
  - type: markdown-syntax
    id: grammar
    syntax: tables
  - type: settings
    id: colors
    name: Colors
    requires: [grammar]
permissions: []
`
	pkg, err := Read(testArchive(t, manifest))
	require.NoError(t, err)
	copy := pkg.Manifest()
	copy.Modules[1].Requires[0] = "mutated"
	assert.Equal(t, "grammar", pkg.Manifest().Modules[1].Requires[0])
	for _, invalid := range []string{strings.ReplaceAll(manifest, "syntax: tables", "syntax: unknown"), strings.ReplaceAll(manifest, "[grammar]", "[absent]"), strings.ReplaceAll(manifest, "[grammar]", "[colors]")} {
		_, err = Read(testArchive(t, invalid))
		require.Error(t, err)
	}
}

func TestContentStyleAndRenderPolicyModules(t *testing.T) {
	manifest := `api_version: 1
id: io.example.typography
name: Typography
version: 1.0.0
modules:
  - type: content-style
    id: typography
    css: plugin.css
    usage:
      - contains: "font"
  - type: render-policy
    id: operators
    policy: preserve-programming-operators
  - type: settings
    id: enabled
    name: Extra typography
    description: Enable optional typography behavior.
permissions: []
`
	pkg, err := Read(testArchive(t, manifest, archiveEntry{"assets/plugin.css", []byte(".prose{}"), 0644}))
	require.NoError(t, err)
	assert.Equal(t, "font", pkg.Manifest().Modules[0].Usage[0].Contains)
	assert.Equal(t, "preserve-programming-operators", pkg.Manifest().Modules[1].Policy)
	assert.Equal(t, "Enable optional typography behavior.", pkg.Manifest().Modules[2].Description)

	for _, invalid := range []string{
		strings.ReplaceAll(manifest, "policy: preserve-programming-operators", "policy: Invalid Policy"),
		strings.ReplaceAll(manifest, "css: plugin.css", "css: ../plugin.css"),
		strings.ReplaceAll(manifest, "type: content-style", "type: renderer-extension\n    stage: preprocess"),
	} {
		_, err := Read(testArchive(t, invalid, archiveEntry{"assets/plugin.css", []byte(".prose{}"), 0644}))
		require.Error(t, err)
	}
}

func TestCodeHighlighterModule(t *testing.T) {
	manifest := `api_version: 1
id: io.example.highlight
name: Highlight
version: 1.0.0
modules:
  - type: code-highlighter
    id: highlighter
    css: plugin.css
permissions: []
`

	pkg, err := Read(testArchive(t, manifest, archiveEntry{"assets/plugin.css", []byte(".prose .token { color: red; }"), 0644}))
	require.NoError(t, err)
	assert.Equal(t, "code-highlighter", pkg.Manifest().Modules[0].Type)

	for _, invalid := range []string{
		strings.ReplaceAll(manifest, "css: plugin.css", "css: ../plugin.css"),
		strings.ReplaceAll(manifest, "css: plugin.css", "css: absent.css"),
		strings.ReplaceAll(manifest, "type: code-highlighter", "type: code-highlighter\n    policy: syntax-highlighting"),
	} {
		_, err := Read(testArchive(t, invalid, archiveEntry{"assets/plugin.css", []byte(".prose .token { color: red; }"), 0644}))
		require.Error(t, err)
	}
}

func TestManifestUsageRules(t *testing.T) {
	manifest := strings.Replace(testManifest, "stage: preprocess", "stage: preprocess\n    usage:\n      - contains: \"!!! \"", 1)
	pkg, err := Read(testArchive(t, manifest))
	require.NoError(t, err)
	require.Len(t, pkg.Manifest().Modules[0].Usage, 1)
	assert.Equal(t, "!!! ", pkg.Manifest().Modules[0].Usage[0].Contains)

	copy := pkg.Manifest()
	copy.Modules[0].Usage[0].Contains = "mutated"
	assert.Equal(t, "!!! ", pkg.Manifest().Modules[0].Usage[0].Contains)

	for _, invalid := range []string{
		strings.Replace(manifest, "- contains: \"!!! \"", "- {}", 1),
		strings.Replace(manifest, "- contains: \"!!! \"", "- contains: \"!!! \"\n        macro: pages", 1),
		strings.Replace(manifest, "contains: \"!!! \"", "macro: ../pages", 1),
	} {
		_, err := Read(testArchive(t, invalid))
		require.Error(t, err, invalid)
	}

	substitution := strings.Replace(testManifest, "stage: preprocess", "stage: preprocess\n    usage:\n      - substitution: include", 1)
	substitutionPackage, err := Read(testArchive(t, substitution))
	require.NoError(t, err)
	assert.Equal(t, "include", substitutionPackage.Manifest().Modules[0].Usage[0].Substitution)

	browser := `api_version: 1
id: io.example.browser-usage
name: Browser usage
version: 1.0.0
modules:
  - type: browser-module
    id: browser
    javascript: plugin.js
    usage:
      - contains: diagram
permissions: [browser:render]
`
	_, err = Read(testArchive(t, browser, archiveEntry{"assets/plugin.js", []byte("globalThis.kumbukaPlugin={}"), 0644}))
	require.Error(t, err)
}

func TestPackageAllowsUnsetOptionalModuleAssets(t *testing.T) {
	manifest := `api_version: 1
id: io.example.syntax
name: Syntax
version: 1.0.0
modules:
  - type: markdown-syntax
    id: grammar
    syntax: strikethrough
permissions: []
`

	pkg, err := Read(testArchiveWithoutWASM(t, manifest))
	require.NoError(t, err)
	assert.Empty(t, pkg.AssetNames())
}

func TestIconResourceModule(t *testing.T) {
	manifest := `api_version: 1
id: io.example.icons
name: Icons
version: 1.0.0
modules:
  - type: icon-resource
    id: brands
    name: Brand Icons
    asset: icons.json
permissions: []
`
	asset := archiveEntry{"assets/icons.json", []byte(`{"format":1,"icons":[{"name":"example-brand","label":"Example","view_box":"0 0 24 24","paths":["M0 0H24V24H0z"]}]}`), 0644}
	pkg, err := Read(testArchiveWithoutWASM(t, manifest, asset))
	require.NoError(t, err)
	require.Len(t, pkg.Manifest().Modules, 1)
	assert.Equal(t, "icon-resource", pkg.Manifest().Modules[0].Type)
	assert.Equal(t, "Brand Icons", pkg.Manifest().Modules[0].Name)
	assert.Equal(t, "icons.json", pkg.Manifest().Modules[0].Asset)
	assert.False(t, pkg.Manifest().RequiresWASM())

	for _, invalid := range []string{
		strings.ReplaceAll(manifest, "name: Brand Icons", "name: ''"),
		strings.ReplaceAll(manifest, "asset: icons.json", "asset: ../icons.json"),
		strings.ReplaceAll(manifest, "asset: icons.json", "asset: icons.svg"),
		strings.ReplaceAll(manifest, "type: icon-resource", "type: content-style"),
	} {
		_, err := Read(testArchiveWithoutWASM(t, invalid, asset))
		require.Error(t, err, invalid)
	}

	_, err = Read(testArchiveWithoutWASM(t, manifest))
	require.ErrorContains(t, err, "missing module asset icons.json")
}

func TestEditorInsertSupportsGenericToolbarActions(t *testing.T) {
	manifest := `api_version: 1
id: com.example.editor
name: Editor
version: 1.0.0
modules:
  - type: editor-insert
    id: strike
    name: Strikethrough
    markdown: "~~"
    suffix: "~~"
    placeholder: text
    mode: wrap
    group: text
    icon: strikethrough-lucide
  - type: editor-insert
    id: tasks
    name: Task list
    markdown: "- [ ] "
    placeholder: task
    mode: prefix-lines
    group: blocks
    icon: list-checks-lucide
permissions: []
`
	pkg, err := Read(testArchiveWithoutWASM(t, manifest))
	require.NoError(t, err)
	require.Len(t, pkg.Manifest().Modules, 2)
	assert.Equal(t, "wrap", pkg.Manifest().Modules[0].Mode)
	assert.Equal(t, "prefix-lines", pkg.Manifest().Modules[1].Mode)

	for _, invalid := range []string{
		strings.Replace(manifest, "mode: wrap", "mode: unknown", 1),
		strings.Replace(manifest, "group: text", "group: sidebar", 1),
		strings.Replace(manifest, "suffix: \"~~\"", "suffix: \"\"", 1),
		strings.Replace(manifest, "icon: strikethrough-lucide", "icon: Bad Icon", 1),
	} {
		_, err := Read(testArchiveWithoutWASM(t, invalid))
		require.Error(t, err)
	}
}

func TestWidgetModule(t *testing.T) {
	manifest := `api_version: 1
id: io.example.widget
name: Widget
version: 1.0.0
modules:
  - type: widget
    id: details
    surface: page.details
    width: wide
    order: 20
permissions:
  - pages:read
`
	pkg, err := Read(testArchive(t, manifest))
	require.NoError(t, err)
	require.Len(t, pkg.Manifest().Modules, 1)
	assert.Equal(t, "page.details", pkg.Manifest().Modules[0].Surface)
	assert.Equal(t, "wide", pkg.Manifest().Modules[0].Width)
	assert.Equal(t, 20, pkg.Manifest().Modules[0].Order)
	assert.True(t, pkg.Manifest().RequiresWASM())

	for _, invalid := range []string{
		strings.Replace(manifest, "page.details", "unknown", 1),
		strings.Replace(manifest, "    surface: page.details\n", "", 1),
		strings.Replace(manifest, "    surface: page.details", "    surface: page.details\n    stage: postprocess", 1),
		strings.Replace(manifest, "width: wide", "width: huge", 1),
		strings.Replace(manifest, "order: 20", "order: 1001", 1),
	} {
		_, err := Read(testArchive(t, invalid))
		require.Error(t, err)
	}
}

func TestAdminActionModule(t *testing.T) {
	manifest := `api_version: 1
id: io.example.admin-action
name: Admin action
version: 1.0.0
modules:
  - type: admin-action
    id: refresh
    name: Refresh cache
    description: Mark cached content stale.
    icon: refresh-cw-lucide
permissions: []
`

	pkg, err := Read(testArchive(t, manifest))
	require.NoError(t, err)
	require.Len(t, pkg.Manifest().Modules, 1)
	assert.Equal(t, "admin-action", pkg.Manifest().Modules[0].Type)
	assert.True(t, pkg.Manifest().RequiresWASM())

	for _, invalid := range []string{
		strings.Replace(manifest, "name: Refresh cache", "name: ''", 1),
		strings.Replace(manifest, "icon: refresh-cw-lucide", "icon: Bad Icon", 1),
		strings.Replace(manifest, "description: Mark cached content stale.", "stage: preprocess\n    description: Mark cached content stale.", 1),
	} {
		_, err := Read(testArchive(t, invalid))
		require.Error(t, err, invalid)
	}
}

func TestPageActionModule(t *testing.T) {
	manifest := `api_version: 1
id: io.example.actions
name: Actions
version: 1.0.0
modules:
  - type: page-action
    id: history
    name: Revision history
    description: Open revision history.
    icon: history-lucide
    kind: dialog
    url: /revisions/${slug}?page=${id}
    order: 25
permissions: []
`

	pkg, err := Read(testArchiveWithoutWASM(t, manifest))
	require.NoError(t, err)
	require.Len(t, pkg.Manifest().Modules, 1)
	assert.Equal(t, "page-action", pkg.Manifest().Modules[0].Type)
	assert.False(t, pkg.Manifest().RequiresWASM())

	for _, invalid := range []string{
		strings.Replace(manifest, "/revisions/${slug}?page=${id}", "https://example.com/history", 1),
		strings.Replace(manifest, "/revisions/${slug}?page=${id}", "/revisions/${unknown}", 1),
		strings.Replace(manifest, "icon: history-lucide", "icon: Bad Icon", 1),
	} {
		_, err := Read(testArchiveWithoutWASM(t, invalid))
		require.Error(t, err)
	}
}

func TestExporterModule(t *testing.T) {
	manifest := `api_version: 1
id: io.example.export
name: Export
version: 1.0.0
modules:
  - type: exporter
    id: text
    name: Plain text
    description: Export the current page as text.
    icon: file-down-lucide
    order: 10
permissions: []
`

	pkg, err := Read(testArchive(t, manifest))
	require.NoError(t, err)
	require.Len(t, pkg.Manifest().Modules, 1)
	assert.Equal(t, "exporter", pkg.Manifest().Modules[0].Type)
	assert.True(t, pkg.Manifest().RequiresWASM())

	for _, invalid := range []string{
		strings.Replace(manifest, "name: Plain text", "name: ''", 1),
		strings.Replace(manifest, "icon: file-down-lucide", "icon: Bad Icon", 1),
		strings.Replace(manifest, "order: 10", "order: 1001", 1),
		strings.Replace(manifest, "order: 10", "url: /download\n    order: 10", 1),
	} {
		_, err := Read(testArchive(t, invalid))
		require.Error(t, err, invalid)
	}
}
