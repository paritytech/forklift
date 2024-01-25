package Rpc

import (
	"forklift/FileManager/Models"
	CacheUsage "forklift/Rpc/Models/CacheUsage"
	log "github.com/sirupsen/logrus"
	"sync"
)

type ForkliftRpc struct {
	Extern        map[string]bool
	Uploads       chan Models.CacheItem
	depsCheckLock sync.RWMutex
	reportLock    sync.RWMutex
	StatusReport  CacheUsage.ForkliftCacheStatusReport
}

func NewForkliftRpc() *ForkliftRpc {
	var uploads = make(chan Models.CacheItem, 20)
	var srv = ForkliftRpc{
		Extern:        make(map[string]bool),
		Uploads:       uploads,
		depsCheckLock: sync.RWMutex{},
		StatusReport:  CacheUsage.ForkliftCacheStatusReport{},
	}
	return &srv
}

func (server *ForkliftRpc) CheckExternDeps(paths *[]string, result *string) error {
	server.depsCheckLock.RLock()
	defer server.depsCheckLock.RUnlock()

	for _, path := range *paths {
		var _, b = server.Extern[path]
		if b {
			*result = path
			return nil
		}
	}
	*result = ""
	return nil
}

func (server *ForkliftRpc) RegisterExternDeps(paths *[]string, result *bool) error {
	server.depsCheckLock.Lock()
	defer server.depsCheckLock.Unlock()

	for _, path := range *paths {
		server.Extern[path] = true
	}
	*result = true
	return nil
}

func (server *ForkliftRpc) AddUpload(cacheItem Models.CacheItem, result *bool) error {
	server.Uploads <- cacheItem
	*result = true

	log.Debugf("Items in upload queue: %d", len(server.Uploads))
	return nil
}

func (server *ForkliftRpc) ReportStatus(report *CacheUsage.StatusReport, result *bool) error {
	server.reportLock.Lock()
	defer server.reportLock.Unlock()

	server.StatusReport.TotalCrates++

	switch report.Status {
	case CacheUsage.CacheHit:
		server.StatusReport.CacheHit++
	case CacheUsage.CacheMiss:
		server.StatusReport.CacheMiss++
	case CacheUsage.DependencyRebuilt:
		server.StatusReport.DependencyRebuilt++
	case CacheUsage.CacheHitWithRetry:
		server.StatusReport.CacheHitWithRetry++
	case CacheUsage.CacheFetchFailed:
		server.StatusReport.CacheFetchFailed++
	case CacheUsage.Undefined:
	}

	server.StatusReport.TotalDownloadTime += report.DownloadTime
	server.StatusReport.TotalDecompressTime += report.DecompressTime
	server.StatusReport.TotalUnpackTime += report.UnpackTime
	server.StatusReport.TotalRustcTime += report.RustcTime

	*result = true
	return nil
}
