package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	for _, line := range strings.Split(string(module), "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "go "); ok {
			return "go" + strings.TrimSpace(value), nil
		}
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
