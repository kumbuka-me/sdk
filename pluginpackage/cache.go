package pluginpackage

import (
	"crypto/sha256"
	"errors"
	"sync"
)

// packageCacheState stores recently validated immutable packages by content digest.
type packageCacheState struct {
	// mu serializes cache lookup and replacement.
	mu sync.Mutex
	// entries stores least-recently-used packages first.
	entries []*Package
}

// Validated packages expose only defensive copies. Sharing immutable package
// instances avoids repeated ZIP expansion for the same bundled or installed bytes.
var packageCache packageCacheState

// Read validates a package archive or returns its process-local cached package.
func Read(data []byte) (*Package, error) {
	if len(data) == 0 || len(data) > MaxArchiveBytes {
		return nil, errors.New("plugin archive exceeds size limit or is empty")
	}

	digest := sha256.Sum256(data)
	packageCache.mu.Lock()
	defer packageCache.mu.Unlock()

	for index, pkg := range packageCache.entries {
		if pkg.digest != digest {
			continue
		}
		copy(packageCache.entries[index:], packageCache.entries[index+1:])
		packageCache.entries[len(packageCache.entries)-1] = pkg
		return pkg, nil
	}

	pkg, err := read(data)
	if err != nil {
		return nil, err
	}
	if len(packageCache.entries) == 8 {
		packageCache.entries = packageCache.entries[1:]
	}
	packageCache.entries = append(packageCache.entries, pkg)
	return pkg, nil
}
