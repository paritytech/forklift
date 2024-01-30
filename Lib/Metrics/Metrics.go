package Metrics

import (
	"context"
	"forklift/Lib"
	"forklift/Lib/Logging"
	"forklift/Rpc/Models/CacheUpload"
	"forklift/Rpc/Models/CacheUsage"
	promwrite "github.com/castai/promwrite"
	"time"
)

func PushMetrics(
	usageReport *CacheUsage.ForkliftCacheStatusReport,
	uploadReport *CacheUpload.ForkliftCacheStatusReport,
	extraLabels map[string]string) {

	var logger = Logging.CreateLogger("Server", 4, nil)

	if !Lib.AppConfig.Metrics.Enabled {
		return
	}

	if Lib.AppConfig.Metrics.PushEndpoint == "" {
		logger.Error("Metrics push endpoint is not set")
		return
	}

	var client = promwrite.NewClient(Lib.AppConfig.Metrics.PushEndpoint)

	_, err := client.Write(context.Background(), &promwrite.WriteRequest{
		TimeSeries: append(
			createUsageTimeSeries(usageReport, extraLabels),
			createUploadTimeSeries(uploadReport, extraLabels)...),
	})

	if err != nil {
		logger.Errorf("Failed to write metrics: %s", err)
	} else {
		logger.Infof("Metrics sent")
	}
}

func createUploadTimeSeries(report *CacheUpload.ForkliftCacheStatusReport, extraLabels map[string]string) []promwrite.TimeSeries {
	var timeNow = time.Now()

	var cacheHitBase = NewIndicator("forklift_wrapper_caching_cache_hit")
	cacheHitBase.Time = timeNow
	cacheHitBase.SetLabels(extraLabels)

	var timeSeries = []promwrite.TimeSeries{
		// upload
		NewIndicatorFull("forklift_uploader_uploading_status", timeNow, float64(report.Uploaded),
			map[string]string{
				"status": "uploaded",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_uploader_uploading_status", timeNow, float64(report.UploadedWithRetry),
			map[string]string{
				"status": "warning",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_uploader_uploading_status", timeNow, float64(report.Failed),
			map[string]string{
				"status": "fail",
			},
			extraLabels,
		).ToTimeSeries(),

		// time
		NewIndicatorFull("forklift_uploader_uploading_time_total", timeNow, float64(report.TotalUploaderWorkTime.Milliseconds()),
			map[string]string{},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_uploader_uploading_time_task", timeNow, float64(report.TotalPackTime.Milliseconds()),
			map[string]string{
				"task": "pack",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_uploader_uploading_time_task", timeNow, float64(report.TotalCompressTime.Milliseconds()),
			map[string]string{
				"task": "compress",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_uploader_uploading_time_task", timeNow, float64(report.TotalUploadTime.Milliseconds()),
			map[string]string{
				"task": "upload",
			},
			extraLabels,
		).ToTimeSeries(),

		// network
		NewIndicatorFull("forklift_uploader_uploading_network_downloaded", timeNow, float64(report.TotalUploadSize),
			map[string]string{},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_uploader_uploading_network_avg_speed", timeNow, float64(report.AverageUploadSpeedBps),
			map[string]string{},
			extraLabels,
		).ToTimeSeries(),
	}

	return timeSeries
}

func createUsageTimeSeries(report *CacheUsage.ForkliftCacheStatusReport, extraLabels map[string]string) []promwrite.TimeSeries {
	var timeNow = time.Now()

	var cacheHitBase = NewIndicator("forklift_wrapper_caching_cache_hit")
	cacheHitBase.Time = timeNow
	cacheHitBase.SetLabels(extraLabels)

	var timeSeries = []promwrite.TimeSeries{
		// hit
		NewIndicatorFull("forklift_wrapper_caching_cache_hit", timeNow, float64(report.CacheHit),
			map[string]string{
				"status": "hit",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_cache_hit", timeNow, float64(report.CacheHitWithRetry),
			map[string]string{
				"status": "warning",
			},
			extraLabels,
		).ToTimeSeries(),

		// miss
		NewIndicatorFull("forklift_wrapper_caching_cache_miss", timeNow, float64(report.CacheMiss),
			map[string]string{
				"status": "miss",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_cache_miss", timeNow, float64(report.CacheFetchFailed),
			map[string]string{
				"status": "fail",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_cache_miss", timeNow, float64(report.DependencyRebuilt),
			map[string]string{
				"status": "dep_rebuilt",
			},
			extraLabels,
		).ToTimeSeries(),

		// timeNow total
		NewIndicatorFull("forklift_wrapper_caching_time_total", timeNow, float64(report.TotalForkliftTime.Milliseconds()),
			map[string]string{},
			extraLabels,
		).ToTimeSeries(),

		// timeNow tasks
		NewIndicatorFull("forklift_wrapper_caching_time_task", timeNow, float64(report.TotalDownloadTime.Milliseconds()),
			map[string]string{
				"task": "download",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_time_task", timeNow, float64(report.TotalDecompressTime.Milliseconds()),
			map[string]string{
				"task": "decompress",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_time_task", timeNow, float64(report.TotalUnpackTime.Milliseconds()),
			map[string]string{
				"task": "unpack",
			},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_time_task", timeNow, float64(report.TotalRustcTime.Milliseconds()),
			map[string]string{
				"task": "rustc",
			},
			extraLabels,
		).ToTimeSeries(),
		// network
		NewIndicatorFull("forklift_wrapper_caching_network_downloaded", timeNow, float64(report.TotalDownloadSize),
			map[string]string{},
			extraLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_network_avg_speed", timeNow, float64(report.AverageDownloadSpeedBps),
			map[string]string{},
			extraLabels,
		).ToTimeSeries(),
	}

	return timeSeries
}
