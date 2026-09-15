// Package build implements development-time validation and standard Go/WASI packaging.
// It is a host tool, not a guest capability.
package build

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kumbuka-me/sdk/pluginpackage"
)

// Build validates and compiles a project, then writes a deterministic package.
// Browser sources must already be compiled to the manifest-declared assets.
func Build(ctx context.Context, directory, destination string) error {
	manifest, err := Validate(directory)
	if err != nil {
		return err
	}
	temporary, err := os.MkdirTemp("", "kumbuka-plugin-build-*")
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(temporary) }()

	if manifest.RequiresWASM() {
		command := exec.CommandContext(ctx,
			"go", "build", "-trimpath", "-buildvcs=false", "-buildmode=c-shared", "-ldflags=-s -w -buildid=", "-o", filepath.Join(temporary, "plugin.wasm"), ".")
		command.Dir = directory

		// Build with the Go version declared by the containing module. A plugin may
		// live in its own module or in a shared repository-level module.
		_, module, err := findModuleFile(directory)
		if err != nil {
			return err
		}

		toolchain := ""
		for _, line := range strings.Split(string(module), "\n") {
			if value, ok := strings.CutPrefix(strings.TrimSpace(line), "go "); ok {
				toolchain = "go" + strings.TrimSpace(value)
				break
			}
		}
		if toolchain == "" {
			return fmt.Errorf("no Go toolchain in %s", directory)
		}

		environment := make([]string, 0, len(os.Environ()))
		for _, value := range os.Environ() {
			key, _, _ := strings.Cut(value, "=")
			switch key {
			case "GOOS", "GOARCH", "GOFLAGS", "GOWORK", "GOTOOLCHAIN", "CGO_ENABLED":
				continue
			}
			environment = append(environment, value)
		}

		command.Env = append(environment, "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN="+toolchain)
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("build %s: %w\n%s", directory, err, output)
		}
	}

	files := map[string]string{
		"README.md":   filepath.Join(directory, "README.md"),
		"plugin.yaml": filepath.Join(directory, "plugin.yaml"),
	}
	if manifest.RequiresWASM() {
		files["plugin.wasm"] = filepath.Join(temporary, "plugin.wasm")
	}
	assets := filepath.Join(directory, "assets")
	if _, err := os.Stat(assets); err == nil {
		if err := filepath.WalkDir(assets, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("plugin asset %s is a symlink", path)
			}
			if entry.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(directory, path)
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() || info.Size() > pluginpackage.MaxAssetBytes {
				return fmt.Errorf("invalid or oversized asset: %s", path)
			}
			files[filepath.ToSlash(relative)] = path
			return nil
		}); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if len(files) > pluginpackage.MaxFiles {
		return fmt.Errorf("too many plugin files")
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		content, err := os.ReadFile(files[name])
		if err != nil {
			return err
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o644)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := writer.Write(content); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}

	if _, err := pluginpackage.Read(buffer.Bytes()); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	previous, _ := os.ReadFile(destination)
	if bytes.Equal(previous, buffer.Bytes()) {
		return nil
	}
	return os.WriteFile(destination, buffer.Bytes(), 0o644)
}
