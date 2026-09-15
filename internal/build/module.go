package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var errModuleNotFound = errors.New("go.mod not found")

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
