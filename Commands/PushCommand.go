package Commands

import (
	"fmt"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager"
	"forklift/Lib"
	"github.com/spf13/cobra"
	"log"
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

		log.Println(params)

		if len(args) > 0 {
			err := os.Chdir(args[0])
			if err != nil {
				log.Println(err)
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

					//log.Println(fmt.Sprintf("Packing %d entries from `%s` for %s-%s", len(files), obj.folder, obj.item.Name, obj.item.Hash))
					var reader, sha = FileManager.Tar(files)
					var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))

					var name = fmt.Sprintf("%s-%s-%s-%s", obj.item.Name, obj.item.Hash, obj.folder, compressor.GetKey())

					var meta, exists = store.GetMetadata(name)

					var needUpload = false

					if !exists {
						log.Printf("%s does not exist in storage, uploading...", name)
						needUpload = true
					} else if meta == nil {
						log.Printf("no metadata for %s, uploading...", name)
						needUpload = true
					} else if shaRemotePtr, ok := meta["Sha-1-Content"]; !ok {
						log.Printf("no metadata for %s, uploading...", name)
						needUpload = true
					} else if *shaRemotePtr != shaLocal {
						log.Println(name, *shaRemotePtr, shaLocal, "checksum mismatch, uploading...")
						needUpload = true
					}

					if needUpload {
						var compressed = compressor.Compress(&reader)
						store.Upload(name, &compressed, map[string]*string{"Sha-1-Content": &shaLocal})
						log.Println(fmt.Sprintf("Uploaded %d entries from `%s` for %s-%s, %x", len(files), obj.folder, obj.item.Name, obj.item.Hash, sha.Sum(nil)))
					}
				} else {
					//log.Println(fmt.Sprintf("No entries from `%s` for %s-%s", obj.folder, obj.item.Name, obj.item.Hash))
				}
			})
	},
}
