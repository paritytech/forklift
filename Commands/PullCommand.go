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
	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull [flags] [project_dir]",
	Short: "Download cache artifacts",
	Run: func(cmd *cobra.Command, args []string) {

		log.Traceln(params)

		if len(args) > 0 {
			err := os.Chdir(args[0])
			if err != nil {
				log.Fatalln(err)
				return
			}
		}

		var cacheItems = FileManager.ParseCacheRequest()

		store, _ := Storages.GetStorageDriver(storage, &params)
		compressor, _ := Compressors.GetCompressor(compression, &params)
		var folders = []string{"build", "deps", ".fingerprint"}

		var cpuCount = runtime.NumCPU()

		var queue = make(chan struct {
			item   FileManager.CacheItem
			path   string
			folder string
		})

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
				var name = fmt.Sprintf("%s-%s-%s-%s", obj.item.Name, obj.item.Hash, obj.folder, compressor.GetKey())
				var meta, existsInStore = store.GetMetadata(name)
				var match = false

				if !existsInStore {
					log.Debugf("%s does not exist in storage\n", name)
					return
				}

				if meta != nil {
					var files = FileManager.FindOpt(obj.path, obj.item.Hash)

					var shaLocal, shaRemote string

					if len(files) <= 0 {
						shaLocal = ""
					} else {
						var _, sha = FileManager.Tar(files)
						shaRemote = *meta["Sha-1-Content"]
						shaLocal = fmt.Sprintf("%x", sha.Sum(nil))
					}

					if shaRemote == shaLocal {
						match = true
						log.Tracef("%s checksum match\n", name)
					} else {
						log.Debugf("%s checksum mismatch, remote: %s local: %s, download\n", name, shaRemote, shaLocal)
					}
				} else {
					log.Debugf(name, "no metadata, download")
				}

				if match {
					return
				}

				var f = store.Download(name)
				if f != nil {
					FileManager.UnTar(obj.path, compressor.Decompress(&f))
					log.Infof("Downloaded artifacts for %s\n", name)
				}
			})
	},
}
