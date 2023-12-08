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
	"forklift/Lib/Rustc"
	log "github.com/sirupsen/logrus"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type Uploader struct {
	sync.WaitGroup

	workDir    string
	uploads    chan Models.CacheItem
	compressor Compressors.ICompressor
	storage    Storages.IStorage
}

func NewUploader(workDir string, storage Storages.IStorage, compressor Compressors.ICompressor) *Uploader {
	var uploader = &Uploader{
		workDir:    workDir,
		compressor: compressor,
		storage:    storage,
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
	for {
		item, more := <-uploader.uploads
		if !more {
			uploader.Done()
			return
		}

		var wrapperTool = Rustc.NewWrapperToolFromCacheItem(uploader.workDir, item)
		log.Debugf("Processing %s %s %s\n", wrapperTool.CrateName, wrapperTool.CrateHash, wrapperTool.OutDir)

		var crateArtifactsFiles = []string{
			path.Join("target", "forklift", fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), "stderr")),
			path.Join("target", "forklift", fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), "stdout")),
			path.Join("target", "forklift", fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), "stdin")),
		}

		//crateArtifactsFiles = append(crateArtifactsFiles, FileManager.FindBuildFiles(wrapperTool.CrateHash)...)

		var stderrFile = wrapperTool.ReadStderrFile()
		fileScanner := bufio.NewScanner(stderrFile)
		fileScanner.Split(bufio.ScanLines)
		for fileScanner.Scan() {
			var artifact CacheStorage.RustcArtifact
			json.Unmarshal([]byte(fileScanner.Text()), &artifact)
			if artifact.Artifact != "" {
				if strings.Contains(artifact.Artifact, "tmp/") ||
					strings.Contains(artifact.Artifact, "/var/folders/") {
					log.Debugf("Temporary artifact folder `%s` detected, skip", artifact.Artifact)
					return
				}
				var relPath, _ = filepath.Rel(uploader.workDir, artifact.Artifact)
				crateArtifactsFiles = append(crateArtifactsFiles, relPath)
			}
		}

		log.Debugf("%s", crateArtifactsFiles)

		if len(crateArtifactsFiles) > 0 {
			var reader, sha = Tar.Pack(crateArtifactsFiles)

			var name = wrapperTool.GetCachePackageName()

			/*var _, exists = uploader.storage.GetMetadata(name)

			if !exists {
				log.Debugf("%s does not exist in storage, uploading...\n", name)
				needUpload = true
			}*/

			var needUpload = true

			var metaMap = wrapperTool.CreateMetadata()
			var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))
			metaMap["sha1-artifact"] = &shaLocal

			if needUpload {
				var compressed = uploader.compressor.Compress(reader)
				uploader.storage.Upload(name+"_"+uploader.compressor.GetKey(), &compressed, metaMap)

				marshal, _ := json.Marshal(metaMap)

				log.Infof("Uploaded %s, metadata: %s", wrapperTool.GetCachePackageName(), marshal)
				log.Infof("Items in upload queue: %d", len(uploader.uploads))
			}
		} else {
			log.Tracef("No entries for %s-%s\n", wrapperTool.GetCachePackageName(), wrapperTool.CrateHash)
		}
	}

}
