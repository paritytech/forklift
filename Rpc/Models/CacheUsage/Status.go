package CacheUsage

type Status int

const (
	Undefined Status = iota
	CacheHit
	CacheMiss
	DependencyRebuilt
	CacheHitWithRetry
	CacheFetchFailed
)
