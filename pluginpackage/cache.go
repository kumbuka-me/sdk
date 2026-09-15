package pluginpackage

import (
	"crypto/sha256"
	"errors"
	"sync"
)

// Validated packages expose only defensive copies. Sharing those immutable
// values avoids repeated ZIP expansion for the same bundled or installed bytes.
var packageCache struct {
	sync.Mutex
	entries []*Package
}

// Read validates a package archive or returns a cloned process-local cached package.
func Read(data []byte) (*Package, error) {
	if len(data) == 0 || len(data) > MaxArchiveBytes {
		return nil, errors.New("plugin archive exceeds size limit or is empty")
	}
	digest := sha256.Sum256(data)
	packageCache.Lock()
	defer packageCache.Unlock()
	for index, pkg := range packageCache.entries {
		if pkg.digest == digest {
			copy(packageCache.entries[index:], packageCache.entries[index+1:])
			packageCache.entries[len(packageCache.entries)-1] = pkg
			return pkg, nil
		}
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
