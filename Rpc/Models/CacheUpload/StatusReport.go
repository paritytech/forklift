package CacheUpload

import (
	"time"
)

type StatusReport struct {
	Status       Status
	CrateName    string
	UploadTime   time.Duration
	CompressTime time.Duration
	PackTime     time.Duration
}

type ForkliftCacheStatusReport struct {
	Total             int
	Uploaded          int
	UploadedWithRetry int
	Failed            int
	TotalUploadTime   time.Duration
	TotalCompressTime time.Duration
	TotalPackTime     time.Duration
}

/*
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
}*/
