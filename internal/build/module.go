package build

import (
	"bytes"
	"errors"
	"fmt"
	"go/version"
	"os"
	"path/filepath"
)

var errModuleNotFound = errors.New("go.mod not found")

// requiredToolchain returns the containing module toolchain when guest code must be built.
func requiredToolchain(directory string, required bool) (string, error) {
	if !required {
		return "", nil
	}
	filename, module, err := findModuleFile(directory)
	if err != nil {
		return "", err
	}
	toolchain, err := moduleToolchain(module)
	if err != nil {
		return "", fmt.Errorf("%s: %w", filename, err)
	}
	return toolchain, nil
}

// moduleToolchain returns the Go toolchain selected by a module's go directive.
func moduleToolchain(module []byte) (string, error) {
	for len(module) > 0 {
		line := module
		if newline := bytes.IndexByte(module, '\n'); newline >= 0 {
			line = module[:newline]
			module = module[newline+1:]
		} else {
			module = nil
		}

		if comment := bytes.Index(line, []byte("//")); comment >= 0 {
			line = line[:comment]
		}
		fields := bytes.Fields(line)
		if len(fields) == 0 || !bytes.Equal(fields[0], []byte("go")) {
			continue
		}
		if len(fields) != 2 {
			return "", fmt.Errorf("go.mod has an invalid go directive")
		}

		languageVersion := string(fields[1])
		if !version.IsValid("go" + languageVersion) {
			return "", fmt.Errorf("go.mod has an invalid go directive")
		}
		toolchain := "go" + languageVersion
		// Modern language versions omit the patch suffix required by toolchain downloads.
		if version.Compare(toolchain, "go1.21") >= 0 && toolchain == version.Lang(toolchain) {
			toolchain += ".0"
		}
		return toolchain, nil
	}
	return "", fmt.Errorf("go.mod has no go directive")
}

// findModuleFile locates the nearest containing go.mod and returns its contents.
func findModuleFile(directory string) (string, []byte, error) {
	current, err := filepath.Abs(directory)
	if err != nil {
		return "", nil, err
	}

	for {
		filename := filepath.Join(current, "go.mod")
		data, err := readRegular(filename, 1<<20)
		if err == nil {
			return filename, data, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", nil, err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", nil, fmt.Errorf("%w for %s", errModuleNotFound, directory)
		}
		current = parent
	}
}
