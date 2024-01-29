package CacheUsage

import (
	"fmt"
	"time"
)

type StatusReport struct {
	Status           Status
	CrateName        string
	DownloadTime     time.Duration
	DecompressTime   time.Duration
	UnpackTime       time.Duration
	RustcTime        time.Duration
	DownloadSize     int64
	DownloadSpeedBps int64
}

type ForkliftCacheStatusReport struct {
	TotalCrates             int
	CacheHit                int
	CacheMiss               int
	DependencyRebuilt       int
	CacheHitWithRetry       int
	CacheFetchFailed        int
	CacheMissCrates         []string
	TotalForkliftTime       time.Duration
	TotalDownloadTime       time.Duration
	TotalDecompressTime     time.Duration
	TotalUnpackTime         time.Duration
	TotalRustcTime          time.Duration
	TotalDownloadSize       int64
	AverageDownloadSpeedBps int64
}

func (s ForkliftCacheStatusReport) String() string {
	return fmt.Sprintf(
		"Cache report:\n"+
			"      Total crates processed: %d\n"+
			"      Cache hit:              %d\n"+
			"      Cache hit with retry:   %d\n"+
			"      Cache miss:             %d\n"+
			"      Dependency rebuilt:     %d\n"+
			"      Cache fetch fail:       %d\n"+
			"      Total forklift time:    %s\n"+
			"      Download time:          %s\n"+
			"      Decompress time:        %s\n"+
			"      Unpack time:            %s\n"+
			"      Rustc time:             %s\n"+
			"      Total download size:    %d bytes\n"+
			"      Average download speed: %d bps\n",
		s.TotalCrates,
		s.CacheHit,
		s.CacheHitWithRetry,
		s.CacheMiss,
		s.DependencyRebuilt,
		s.CacheFetchFailed,
		s.TotalForkliftTime,
		s.TotalDownloadTime.Truncate(time.Millisecond),
		s.TotalDecompressTime.Truncate(time.Millisecond),
		s.TotalUnpackTime.Truncate(time.Millisecond),
		s.TotalRustcTime.Truncate(time.Millisecond),
		s.TotalDownloadSize,
		s.AverageDownloadSpeedBps,
	)
}
