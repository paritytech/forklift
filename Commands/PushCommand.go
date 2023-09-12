package Commands

import (
	"fmt"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager"
	"forklift/Lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	rootCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push [flags] [project_dir]",
	Short: "Upload cache artifacts. Require storage params",
	Run: func(cmd *cobra.Command, args []string) {

		log.Traceln(params)

		if len(args) > 0 {
			err := os.Chdir(args[0])
			if err != nil {
				log.Errorln(err)
				return
			}
		}

		store, _ := Storages.GetStorageDriver(storage, &params)
		compressor, _ := Compressors.GetCompressor(compression, &params)

		var cacheItems = FileManager.ParseCacheRequest()
		var folders = []string{"build", "deps", ".fingerprint"}

		var cpuCount = runtime.NumCPU()

		var queue = make(chan struct {
			item   FileManager.CacheItem
			path   string
			folder string
		}, 20)

		go func() {
			for _, folder := range folders {
				var path = filepath.Join("target", mode, folder)

				for _, item := range cacheItems {
					queue <- struct {
						item   FileManager.CacheItem
						path   string
						folder string
					}{item: item, path: path, folder: folder}
				}
			}
			close(queue)
		}()

		Lib.Parallel(
			queue,
			cpuCount,
			func(obj struct {
				item   FileManager.CacheItem
				path   string
				folder string
			}) {
				var files = FileManager.FindOpt(obj.path, obj.item.Hash)
				if len(files) > 0 {

					var reader, sha = FileManager.Tar(files)
					var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))

					var name = fmt.Sprintf("%s-%s-%s-%s", obj.item.Name, obj.item.Hash, obj.folder, compressor.GetKey())

					var meta, exists = store.GetMetadata(name)

					var needUpload = false

					if !exists {
						log.Debugf("%s does not exist in storage, uploading...\n", name)
						needUpload = true
					} else if meta == nil {
						log.Debugf("no metadata for %s, uploading...\n", name)
						needUpload = true
					} else if shaRemotePtr, ok := meta["sha-1-content"]; !ok {
						log.Debugf("no metadata for %s, uploading...\n", name)
						needUpload = true
					} else if *shaRemotePtr != shaLocal {
						log.Debugf("%s checksum mismatch, remote: %s, local: %s, uploading...\n", name, *shaRemotePtr, shaLocal)
						needUpload = true
					}

					if needUpload {
						var compressed = compressor.Compress(&reader)
						store.Upload(name, &compressed, map[string]*string{"sha-1-content": &shaLocal})
						log.Infof("Uploaded %d entries from `%s` for %s-%s, %x\n", len(files), obj.folder, obj.item.Name, obj.item.Hash, sha.Sum(nil))
					}
				} else {
					log.Debugf("No entries from `%s` for %s-%s\n", obj.folder, obj.item.Name, obj.item.Hash)
				}
			})
	},
}
