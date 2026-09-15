package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kumbuka-me/sdk/pluginpackage"
)

// Validate checks the exact runtime manifest schema and project assets before compilation.
func Validate(directory string) (pluginpackage.Manifest, error) {
	var empty pluginpackage.Manifest
	data, err := readRegular(filepath.Join(directory, "plugin.yaml"), pluginpackage.MaxManifestBytes)
	if err != nil {
		return empty, err
	}
	manifest, err := pluginpackage.ParseManifest(data)
	if err != nil {
		return empty, err
	}
	if _, err := readRegular(filepath.Join(directory, "README.md"), pluginpackage.MaxREADMEBytes); err != nil {
		return empty, err
	}
	if manifest.RequiresWASM() {
		if _, _, err := findModuleFile(directory); err != nil {
			return empty, err
		}
	}
	// Walk every packaged asset, including dependencies referenced by JS/CSS.
	count, total := 2, 0
	assets := filepath.Join(directory, "assets")
	if info, err := os.Lstat(assets); err == nil {
		if !info.IsDir() {
			return empty, fmt.Errorf("assets must be a real directory")
		}
		err = filepath.WalkDir(assets, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("asset symlinks are forbidden: %s", path)
			}
			if entry.IsDir() {
				return nil
			}
			data, err := readRegular(path, pluginpackage.MaxAssetBytes)
			if err != nil {
				return err
			}
			count++
			total += len(data)
			if count > pluginpackage.MaxFiles || total > pluginpackage.MaxExpandedBytes {
				return fmt.Errorf("plugin assets exceed package limits")
			}
			return nil
		})
		if err != nil {
			return empty, err
		}
	} else if !os.IsNotExist(err) {
		return empty, err
	}
	for _, module := range manifest.Modules {
		for _, name := range []string{module.JavaScript, module.CSS} {
			if name != "" {
				if _, err := readRegular(filepath.Join(assets, filepath.FromSlash(name)), pluginpackage.MaxAssetBytes); err != nil {
					return empty, fmt.Errorf("module %s asset: %w", module.ID, err)
				}
			}
		}
	}
	return manifest, nil
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
