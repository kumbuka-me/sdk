package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kumbuka-me/sdk/pluginpackage"
)

// project contains validated source files ready to be packaged.
type project struct {
	// manifest contains the validated plugin manifest.
	manifest pluginpackage.Manifest
	// files maps package-relative paths to their validated contents.
	files map[string][]byte
	// toolchain is the Go toolchain required for executable plugins.
	toolchain string
}

// Validate checks the exact runtime manifest schema and project assets before compilation.
func Validate(directory string) (pluginpackage.Manifest, error) {
	project, err := loadProject(directory)
	if err != nil {
		return pluginpackage.Manifest{}, err
	}
	return project.manifest, nil
}

// loadProject validates a project once and retains the bounded files needed for packaging.
func loadProject(directory string) (project, error) {
	manifestData, err := readRegular(filepath.Join(directory, "plugin.yaml"), pluginpackage.MaxManifestBytes)
	if err != nil {
		return project{}, err
	}
	manifest, err := pluginpackage.ParseManifest(manifestData)
	if err != nil {
		return project{}, err
	}

	readme, err := readRegular(filepath.Join(directory, "README.md"), pluginpackage.MaxREADMEBytes)
	if err != nil {
		return project{}, err
	}
	toolchain, err := requiredToolchain(directory, manifest.RequiresWASM())
	if err != nil {
		return project{}, err
	}

	files := map[string][]byte{
		"README.md":   readme,
		"plugin.yaml": manifestData,
	}
	loader := assetLoader{directory: directory, files: files, reservesWASM: manifest.RequiresWASM()}
	if err := loader.load(); err != nil {
		return project{}, err
	}
	if err := validateModuleAssets(manifest, files); err != nil {
		return project{}, err
	}

	return project{manifest: manifest, files: files, toolchain: toolchain}, nil
}

// assetLoader validates and collects package assets during one directory walk.
type assetLoader struct {
	// directory is the plugin project root.
	directory string
	// files receives package-relative asset contents.
	files map[string][]byte
	// total tracks expanded asset bytes loaded so far.
	total int
	// reservesWASM accounts for the compiled guest module in the package file limit.
	reservesWASM bool
}

// load validates the assets directory and walks its contents when present.
func (l *assetLoader) load() error {
	assets := filepath.Join(l.directory, "assets")
	info, err := os.Lstat(assets)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("assets must be a real directory")
	}
	return filepath.WalkDir(assets, l.visit)
}

// visit validates and loads one entry from the plugin assets tree.
func (l *assetLoader) visit(filename string, entry os.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("asset symlinks are forbidden: %s", filename)
	}
	if entry.IsDir() {
		return nil
	}

	data, err := readRegular(filename, pluginpackage.MaxAssetBytes)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(l.directory, filename)
	if err != nil {
		return err
	}
	l.files[filepath.ToSlash(relative)] = data
	l.total += len(data)

	fileCount := len(l.files)
	if l.reservesWASM {
		fileCount++
	}
	if fileCount > pluginpackage.MaxFiles || l.total > pluginpackage.MaxExpandedBytes {
		return fmt.Errorf("plugin assets exceed package limits")
	}
	return nil
}

// validateModuleAssets verifies manifest-declared assets against the loaded project files.
func validateModuleAssets(manifest pluginpackage.Manifest, files map[string][]byte) error {
	for _, module := range manifest.Modules {
		for _, name := range []string{module.JavaScript, module.CSS, module.Asset} {
			if name == "" {
				continue
			}

			data, ok := files["assets/"+name]
			if !ok {
				return fmt.Errorf("module %s asset %s: %w", module.ID, name, os.ErrNotExist)
			}
			if module.Type == "icon-resource" && name == module.Asset {
				if _, err := pluginpackage.ParseIconResource(data); err != nil {
					return fmt.Errorf("module %s icon resource: %w", module.ID, err)
				}
			}
		}
	}
	return nil
}

// readRegular rejects special files and applies limits before reading.
func readRegular(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("invalid or oversized regular file: %s", path)
	}
	return os.ReadFile(path)
}
