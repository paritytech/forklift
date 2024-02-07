package CacheUpload

import (
	"fmt"
	"time"
)

type StatusReport struct {
	Status         Status
	CrateName      string
	WorkTime       time.Duration
	UploadTime     time.Duration
	CompressTime   time.Duration
	PackTime       time.Duration
	UploadSize     int64
	UploadSpeedBps int64
}

type ForkliftCacheStatusReport struct {
	Total                 int
	Uploaded              int
	UploadedWithRetry     int
	Failed                int
	TotalUploaderWorkTime time.Duration
	TotalUploadTime       time.Duration
	TotalCompressTime     time.Duration
	TotalPackTime         time.Duration
	TotalUploadSize       int64
	AverageUploadSpeedBps int64
}

func (s ForkliftCacheStatusReport) String() string {
	return fmt.Sprintf(
		"   Cache upload report:\n"+
			"      Total uploads:          %d\n"+
			"      Uploaded:               %d\n"+
			"      With retry:             %d\n"+
			"      Failed:                 %d\n"+
			"      Pack time:              %s\n"+
			"      Compress time:          %s\n"+
			"      Upload time:            %s\n"+
			"      Uploaded size:          %d bytes\n"+
			"      Average upload speed:   %d bps\n",
		s.Total,
		s.Uploaded,
		s.UploadedWithRetry,
		s.Failed,
		s.TotalPackTime.Truncate(time.Millisecond),
		s.TotalCompressTime.Truncate(time.Millisecond),
		s.TotalUploadTime.Truncate(time.Millisecond),
		s.TotalUploadSize,
		s.AverageUploadSpeedBps,
	)
}
