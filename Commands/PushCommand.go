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
				var files = FileManager.Find(obj.path, obj.item.Hash)
				if len(files) > 0 {

					log.Println(fmt.Sprintf("Packing %d entries from `%s` for %s-%s", len(files), obj.folder, obj.item.Name, obj.item.Hash))
					var reader = FileManager.Tar(files)
					var compressed = compressor.Compress(&reader)
					var name = fmt.Sprintf("%s-%s-%s", obj.item.Name, obj.item.Hash, obj.folder)

					store.Upload(name, &compressed, nil)

					log.Println(fmt.Sprintf("Uploaded %d entries from `%s` for %s-%s", len(files), obj.folder, obj.item.Name, obj.item.Hash))
				} else {
					log.Println(fmt.Sprintf("No entries from `%s` for %s-%s", obj.folder, obj.item.Name, obj.item.Hash))
				}
			})
	},
}
