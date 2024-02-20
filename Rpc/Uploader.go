package Rpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"forklift/CacheStorage"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager/Models"
	"forklift/FileManager/Tar"
	"forklift/Lib/Diagnostic/Time"
	"forklift/Lib/Logging"
	log "forklift/Lib/Logging/ConsoleLogger"
	"forklift/Lib/Rustc"
	"forklift/Rpc/Models/CacheUpload"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type Uploader struct {
	sync.WaitGroup

	workDir      string
	uploads      chan Models.CacheItem
	compressor   Compressors.ICompressor
	storage      Storages.IStorage
	StatusReport CacheUpload.ForkliftCacheStatusReport
}

func NewUploader(workDir string, storage Storages.IStorage, compressor Compressors.ICompressor) *Uploader {
	var uploader = &Uploader{
		workDir:      workDir,
		compressor:   compressor,
		storage:      storage,
		StatusReport: CacheUpload.ForkliftCacheStatusReport{},
	}
	return uploader
}

func (uploader *Uploader) Start(queue chan Models.CacheItem, threads int) {
	uploader.uploads = queue

	for i := 0; i < threads; i++ {
		uploader.Add(1)
		go uploader.upload()
	}
}

func (uploader *Uploader) upload() {
	var l = Logging.CreateLogger("Uploader", 4, nil)

	for {
		item, more := <-uploader.uploads
		if !more {
			uploader.Done()
			return
		}

		var wrapperTool = Rustc.NewWrapperToolFromCacheItem(uploader.workDir, item)
		var logger = l.WithFields(log.Fields{
			"crate": wrapperTool.CrateName,
			"hash":  wrapperTool.CrateHash,
		})

		wrapperTool.Logger = logger

		logger.Debugf("Processing %s %s %s", wrapperTool.CrateName, wrapperTool.CrateHash, wrapperTool.OutDir)

		var crateArtifactsFiles = []string{
			path.Join("target", "forklift", fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), "stderr")),
			//path.Join("target", "forklift", fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), "stdout")),
			//path.Join("target", "forklift", fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), "stdin")),
		}

		var stderrFile = wrapperTool.ReadStderrFile()
		fileScanner := bufio.NewScanner(stderrFile)
		fileScanner.Split(bufio.ScanLines)
		for fileScanner.Scan() {
			var artifact CacheStorage.RustcArtifact
			json.Unmarshal([]byte(fileScanner.Text()), &artifact)
			if artifact.Artifact != "" {
				if strings.Contains(artifact.Artifact, "tmp/") ||
					strings.Contains(artifact.Artifact, "/var/folders/") {
					logger.Tracef("Temporary artifact folder `%s` detected, skip", artifact.Artifact)
					return
				}
				var relPath, _ = filepath.Rel(uploader.workDir, artifact.Artifact)
				crateArtifactsFiles = append(crateArtifactsFiles, relPath)
			}
		}

		if len(crateArtifactsFiles) > 0 {
			var report = uploader.TryUpload(wrapperTool, crateArtifactsFiles, logger)
			uploader.CollectReport(&report)
		} else {
			logger.Tracef("No entries for '%s-%s'\n", wrapperTool.GetCachePackageName(), wrapperTool.CrateHash)
		}
	}

}

// TryUpload -	Upload crate artifacts to cache
func (uploader *Uploader) TryUpload(
	wrapperTool *Rustc.WrapperTool,
	crateArtifactsFiles []string,
	logger *log.Logger) CacheUpload.StatusReport {

	var timer = Time.NewForkliftTimer()

	timer.Start("uploader work time")

	var name = wrapperTool.GetCachePackageName()
	var metaMap = wrapperTool.CreateMetadata()

	var statusReport = CacheUpload.StatusReport{}
	statusReport.CrateName = wrapperTool.CrateName

	var retries = 3
	for retries > 0 {

		timer.Start("Pack time")
		var reader, sha, err = Tar.Pack(crateArtifactsFiles)
		statusReport.PackTime += timer.Stop("Pack time")
		if err != nil {
			logger.Errorf("pack error: %s", err)
			retries--
			continue
		}

		var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))
		metaMap["sha1-artifact"] = &shaLocal

		timer.Start("Compress time")
		compressed, err := uploader.compressor.Compress(reader)
		statusReport.CompressTime += timer.Stop("Compress time")
		if err != nil {
			logger.Warningf("compression error: %s", err)
			retries--
			continue
		}

		timer.Start("Upload time")
		uploadResult, err := uploader.storage.Upload(name+"_"+uploader.compressor.GetKey(), &compressed, metaMap)
		statusReport.UploadTime += timer.Stop("Upload time")
		if err != nil {
			logger.Warningf("upload error: %s", err)
			retries--
			continue
		}
		statusReport.UploadSize += uploadResult.BytesCount
		statusReport.UploadSpeedBps += uploadResult.SpeedBps

		marshal, _ := json.Marshal(metaMap)
		logger.Infof("Uploaded %s, metadata: %s", wrapperTool.GetCachePackageName(), marshal)
		statusReport.Status = CacheUpload.Uploaded
		statusReport.WorkTime += timer.Stop("uploader work time")
		return statusReport
	}

	logger.Errorf("Failed to upload artifact for '%s, %s'", wrapperTool.GetCachePackageName(), wrapperTool.CrateHash)
	statusReport.Status = CacheUpload.Failed
	statusReport.WorkTime += timer.Stop("uploader work time")

	return statusReport
}

func (uploader *Uploader) CollectReport(report *CacheUpload.StatusReport) {

	switch report.Status {
	case CacheUpload.Uploaded:
		uploader.StatusReport.Uploaded++
	case CacheUpload.Failed:
		uploader.StatusReport.Failed++
	case CacheUpload.UploadedWithRetry:
		uploader.StatusReport.UploadedWithRetry++
	default:
	}

	uploader.StatusReport.Total++

	uploader.StatusReport.TotalPackTime += report.PackTime
	uploader.StatusReport.TotalCompressTime += report.CompressTime
	uploader.StatusReport.TotalUploadTime += report.UploadTime

	uploader.StatusReport.TotalUploaderWorkTime += report.WorkTime

	uploader.StatusReport.TotalUploadSize += report.UploadSize
	uploader.StatusReport.AverageUploadSpeedBps += report.UploadSpeedBps
	//= int64(float64(uploader.StatusReport.TotalUploadSize) / uploader.StatusReport.TotalUploadTime.Seconds())

}
