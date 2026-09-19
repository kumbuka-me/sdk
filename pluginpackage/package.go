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
	"slices"
	"strings"
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
	manifest := p.manifest
	manifest.Modules = slices.Clone(manifest.Modules)
	for index := range manifest.Modules {
		manifest.Modules[index].Requires = slices.Clone(manifest.Modules[index].Requires)
		manifest.Modules[index].Fields = slices.Clone(manifest.Modules[index].Fields)
		for fieldIndex := range manifest.Modules[index].Fields {
			field := &manifest.Modules[index].Fields[fieldIndex]
			field.Options = slices.Clone(field.Options)
		}
		manifest.Modules[index].Usage = slices.Clone(manifest.Modules[index].Usage)
	}
	manifest.Requires = slices.Clone(manifest.Requires)
	manifest.Permissions = slices.Clone(manifest.Permissions)
	return manifest
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

// read validates an archive and constructs an immutable package representation.
func read(data []byte) (*Package, error) {
	files, err := readArchive(data)
	if err != nil {
		return nil, err
	}
	return packageFromFiles(data, files)
}

// readArchive validates archive structure and returns bounded file contents by path.
func readArchive(data []byte) (map[string][]byte, error) {
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
		name, content, isFile, err := readArchiveEntry(file, seen, MaxExpandedBytes-total)
		if err != nil {
			return nil, err
		}
		if !isFile {
			continue
		}
		files[name] = content
		total += len(content)
	}
	if err := validateArchiveHierarchy(seen, files); err != nil {
		return nil, err
	}
	return files, nil
}

// readArchiveEntry validates one ZIP entry and returns file content when applicable.
func readArchiveEntry(file *zip.File, seen map[string]bool, remaining int) (string, []byte, bool, error) {
	name := strings.TrimSuffix(file.Name, "/")
	if !validPath(name) || seen[name] {
		return "", nil, false, fmt.Errorf("invalid or duplicate plugin path %q", file.Name)
	}
	seen[name] = true

	if kind := file.Mode().Type(); kind != 0 && kind != fs.ModeDir {
		return "", nil, false, fmt.Errorf("plugin entry %q is not a regular file or directory", name)
	}
	if file.FileInfo().IsDir() {
		if err := validateArchiveDirectory(file, name); err != nil {
			return "", nil, false, err
		}
		return name, nil, false, nil
	}

	limit, err := entryLimit(name)
	if err != nil {
		return "", nil, false, err
	}
	content, err := readEntry(file, min(limit, remaining))
	if err != nil {
		return "", nil, false, err
	}
	return name, content, true, nil
}

// validateArchiveDirectory accepts only canonical empty directories under assets/.
func validateArchiveDirectory(file *zip.File, name string) error {
	if !strings.HasSuffix(file.Name, "/") || file.UncompressedSize64 != 0 {
		return fmt.Errorf("invalid plugin directory %q", name)
	}
	if name != "assets" && !strings.HasPrefix(name, "assets/") {
		return fmt.Errorf("unsupported plugin directory %q", name)
	}
	return nil
}

// validateArchiveHierarchy rejects entries whose ancestor is also an archive file.
func validateArchiveHierarchy(seen map[string]bool, files map[string][]byte) error {
	for name := range seen {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if _, exists := files[parent]; exists {
				return fmt.Errorf("plugin path %q has a file as parent", name)
			}
		}
	}
	return nil
}

// packageFromFiles validates required package content and constructs a Package.
func packageFromFiles(archive []byte, files map[string][]byte) (*Package, error) {
	manifest, err := ParseManifest(files["plugin.yaml"])
	if err != nil {
		return nil, err
	}

	wasm := files["plugin.wasm"]
	if err := validateWASM(manifest, wasm); err != nil {
		return nil, err
	}
	readme := files["README.md"]
	if len(bytes.TrimSpace(readme)) == 0 {
		return nil, errors.New("missing or empty README.md")
	}

	assets := collectAssets(files)
	if err := validateModuleAssets(manifest, assets); err != nil {
		return nil, err
	}
	return &Package{
		manifest: manifest,
		wasm:     wasm,
		assets:   assets,
		readme:   bytes.Clone(readme),
		digest:   sha256.Sum256(archive),
	}, nil
}

// validateWASM enforces executable and declarative package WASM requirements.
func validateWASM(manifest Manifest, wasm []byte) error {
	if manifest.RequiresWASM() && !hasWASMHeader(wasm) {
		return errors.New("missing or invalid plugin.wasm")
	}
	if !manifest.RequiresWASM() && len(wasm) != 0 && !hasWASMHeader(wasm) {
		return errors.New("invalid plugin.wasm")
	}
	return nil
}

// collectAssets indexes files below assets/ by their package-relative asset name.
func collectAssets(files map[string][]byte) map[string][]byte {
	assets := make(map[string][]byte)
	for name, content := range files {
		if asset, ok := strings.CutPrefix(name, "assets/"); ok {
			assets[asset] = content
		}
	}
	return assets
}

// validateModuleAssets verifies that every declared module asset exists and is valid.
func validateModuleAssets(manifest Manifest, assets map[string][]byte) error {
	for _, module := range manifest.Modules {
		for _, name := range []string{module.JavaScript, module.CSS, module.Asset} {
			if name == "" {
				continue
			}
			data, ok := assets[name]
			if !ok {
				return fmt.Errorf("missing module asset %s", name)
			}
			if module.Type == "icon-resource" && name == module.Asset {
				if _, err := ParseIconResource(data); err != nil {
					return fmt.Errorf("icon resource %s: %w", module.ID, err)
				}
			}
		}
	}
	return nil
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
