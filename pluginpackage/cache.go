package pluginpackage

import (
	"crypto/sha256"
	"errors"
	"sync"
)

const packageCacheCapacity = 8

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
	if pkg := packageCache.lookup(digest); pkg != nil {
		return pkg, nil
	}

	pkg, err := read(data)
	if err != nil {
		return nil, err
	}
	return packageCache.store(pkg), nil
}

// lookup returns and promotes a cached package with the requested digest.
func (c *packageCacheState) lookup(digest [32]byte) *Package {
	c.mu.Lock()
	defer c.mu.Unlock()

	for index, pkg := range c.entries {
		if pkg.digest == digest {
			c.promote(index)
			return pkg
		}
	}
	return nil
}

// store inserts a validated package or returns an identical package cached concurrently.
func (c *packageCacheState) store(pkg *Package) *Package {
	c.mu.Lock()
	defer c.mu.Unlock()

	for index, cached := range c.entries {
		if cached.digest == pkg.digest {
			c.promote(index)
			return cached
		}
	}
	if len(c.entries) == packageCacheCapacity {
		c.entries = c.entries[1:]
	}
	c.entries = append(c.entries, pkg)
	return pkg
}

// promote moves one cache entry to the most-recently-used position.
func (c *packageCacheState) promote(index int) {
	pkg := c.entries[index]
	copy(c.entries[index:], c.entries[index+1:])
	c.entries[len(c.entries)-1] = pkg
}
