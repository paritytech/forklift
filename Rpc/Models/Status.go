package Models

import "fmt"

type CrateCacheStatus int

const (
	Undefined CrateCacheStatus = iota
	CacheUsed
	CacheMiss
	DependencyRebuilt
)

type CrateCacheStatusReport struct {
	CacheStatus CrateCacheStatus
	CrateName   string
}

type ForkliftCacheStatusReport struct {
	TotalCrates       int
	FromCache         int
	CacheMiss         int
	DependencyRebuilt int
	CacheMissCrates   []string
}

func (s ForkliftCacheStatusReport) String() string {
	return fmt.Sprintf("Cache report:\nTotal crates processed: %d\nFrom cache: %d\nCache miss: %d\nDependency rebuilt: %d\n", s.TotalCrates, s.FromCache, s.CacheMiss, s.DependencyRebuilt)
}
