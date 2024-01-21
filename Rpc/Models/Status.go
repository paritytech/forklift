package Models

import "fmt"

type CrateCacheStatus int

const (
	Undefined CrateCacheStatus = iota
	CacheUsed
	CacheMiss
	DependencyRebuilt
	CacheUsedWithRetry
	CacheFetchFailed
)

type CrateCacheStatusReport struct {
	CacheStatus CrateCacheStatus
	CrateName   string
}

type ForkliftCacheStatusReport struct {
	TotalCrates        int
	CacheUsed          int
	CacheMiss          int
	DependencyRebuilt  int
	CacheUsedWithRetry int
	CacheFetchFailed   int
	CacheMissCrates    []string
}

func (s ForkliftCacheStatusReport) String() string {
	return fmt.Sprintf(
		"Cache report:\n"+
			"      Total crates processed: %d\n"+
			"      From cache: %d\n"+
			"      From cache with retry: %d\n"+
			"      Cache miss: %d\n"+
			"      Dependency rebuilt: %d\n"+
			"      Cache package fetch fail: %d\n",
		s.TotalCrates, s.CacheUsed, s.CacheUsedWithRetry, s.CacheMiss, s.DependencyRebuilt, s.CacheFetchFailed)
}
