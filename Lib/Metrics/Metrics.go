package Metrics

import (
	"context"
	"forklift/Lib"
	"forklift/Lib/Logging"
	"forklift/Rpc/Models/CacheUsage"
	promwrite "github.com/castai/promwrite"
	"time"
)

func PushMetrics(report *CacheUsage.ForkliftCacheStatusReport, commonLabels map[string]string) {

	var logger = Logging.CreateLogger("Server", 4, nil)

	if !Lib.AppConfig.Metrics.Enabled {
		return
	}

	if Lib.AppConfig.Metrics.PushEndpoint == "" {
		logger.Error("Metrics push endpoint is not set")
		return
	}

	//var _ = createTimeSeries(report, commonLabels)
	var client = promwrite.NewClient(Lib.AppConfig.Metrics.PushEndpoint)

	_, err := client.Write(context.Background(), &promwrite.WriteRequest{
		TimeSeries: createTimeSeries(report, commonLabels),
	})

	if err != nil {
		logger.Errorf("Failed to write metrics: %s", err)
	} else {
		logger.Infof("Metrics sent to %s", Lib.AppConfig.Metrics.PushEndpoint)
	}
}

func createTimeSeries(report *CacheUsage.ForkliftCacheStatusReport, commonLabels map[string]string) []promwrite.TimeSeries {
	var timeNow = time.Now()

	var cacheHitBase = NewIndicator("forklift_wrapper_caching_cache_hit")
	cacheHitBase.Time = timeNow
	cacheHitBase.SetLabels(commonLabels)

	var timeSeries = []promwrite.TimeSeries{
		// hit
		NewIndicatorFull("forklift_wrapper_caching_cache_hit", timeNow, float64(report.CacheHit),
			map[string]string{
				"status": "hit",
			},
			commonLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_cache_hit", timeNow, float64(report.CacheHitWithRetry),
			map[string]string{
				"status": "warning",
			},
			commonLabels,
		).ToTimeSeries(),

		// miss
		NewIndicatorFull("forklift_wrapper_caching_cache_miss", timeNow, float64(report.CacheMiss),
			map[string]string{
				"status": "miss",
			},
			commonLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_cache_miss", timeNow, float64(report.CacheFetchFailed),
			map[string]string{
				"status": "fail",
			},
			commonLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_cache_miss", timeNow, float64(report.DependencyRebuilt),
			map[string]string{
				"status": "dep_rebuilt",
			},
			commonLabels,
		).ToTimeSeries(),

		// timeNow total
		NewIndicatorFull("forklift_wrapper_caching_time_total", timeNow, float64(report.TotalForkliftTime.Milliseconds()),
			map[string]string{},
			commonLabels,
		).ToTimeSeries(),

		// timeNow tasks
		NewIndicatorFull("forklift_wrapper_caching_time_task", timeNow, float64(report.TotalDownloadTime.Milliseconds()),
			map[string]string{
				"task": "download",
			},
			commonLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_time_task", timeNow, float64(report.TotalDecompressTime.Milliseconds()),
			map[string]string{
				"task": "decompress",
			},
			commonLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_time_task", timeNow, float64(report.TotalUnpackTime.Milliseconds()),
			map[string]string{
				"task": "unpack",
			},
			commonLabels,
		).ToTimeSeries(),
		NewIndicatorFull("forklift_wrapper_caching_time_task", timeNow, float64(report.TotalRustcTime.Milliseconds()),
			map[string]string{
				"task": "rustc",
			},
			commonLabels,
		).ToTimeSeries(),
	}

	return timeSeries
}
