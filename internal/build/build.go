// Package build implements development-time validation and standard Go/WASI packaging.
// It is a host tool, not a guest capability.
package build

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
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
	project, err := loadProject(directory)
	if err != nil {
		return err
	}
	if project.manifest.RequiresWASM() {
		wasm, err := compileWASM(ctx, directory, project.toolchain)
		if err != nil {
			return err
		}
		project.files["plugin.wasm"] = wasm
	}

	archive, err := writeArchive(project.files)
	if err != nil {
		return err
	}
	if _, err := pluginpackage.Read(archive); err != nil {
		return err
	}
	return writePackage(destination, archive)
}

// compileWASM builds one plugin with the validated Go toolchain for its containing module.
func compileWASM(ctx context.Context, directory, toolchain string) ([]byte, error) {
	temporary, err := os.MkdirTemp("", "kumbuka-plugin-build-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(temporary) }()

	outputPath := filepath.Join(temporary, "plugin.wasm")
	command := exec.CommandContext(
		ctx,
		"go", "build",
		"-trimpath",
		"-buildvcs=false",
		"-buildmode=c-shared",
		"-ldflags=-s -w -buildid=",
		"-o", outputPath,
		".",
	)
	command.Dir = directory
	command.Env = wasmBuildEnvironment(toolchain)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build %s: %w\n%s", directory, err, output)
	}
	return readRegular(outputPath, pluginpackage.MaxWASMBytes)
}

// wasmBuildEnvironment returns a clean environment for deterministic WASI compilation.
func wasmBuildEnvironment(toolchain string) []string {
	environment := make([]string, 0, len(os.Environ())+6)
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		switch key {
		case "GOOS", "GOARCH", "GOFLAGS", "GOWORK", "GOTOOLCHAIN", "CGO_ENABLED":
			continue
		}
		environment = append(environment, value)
	}
	return append(
		environment,
		"GOOS=wasip1",
		"GOARCH=wasm",
		"CGO_ENABLED=0",
		"GOFLAGS=",
		"GOWORK=off",
		"GOTOOLCHAIN="+toolchain,
	)
}

// writeArchive serializes validated package files into a deterministic ZIP archive.
func writeArchive(files map[string][]byte) ([]byte, error) {
	if len(files) > pluginpackage.MaxFiles {
		return nil, fmt.Errorf("too many plugin files")
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o644)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := writer.Write(files[name]); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// writePackage creates the destination directory and avoids rewriting identical packages.
func writePackage(destination string, archive []byte) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	previous, _ := os.ReadFile(destination)
	if bytes.Equal(previous, archive) {
		return nil
	}
	return os.WriteFile(destination, archive, 0o644)
}
