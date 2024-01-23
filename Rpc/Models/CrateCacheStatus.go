package Models

type CrateCacheStatus int

const (
	Undefined CrateCacheStatus = iota
	CacheUsed
	CacheMiss
	DependencyRebuilt
	CacheUsedWithRetry
	CacheFetchFailed
)
