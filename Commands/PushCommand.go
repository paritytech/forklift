package Commands

import (
	"fmt"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
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

		for _, folder := range folders {
			var path = filepath.Join("target", mode, folder)

			for _, item := range cacheItems {
				var files = FileManager.Find(path, item.Hash)

				if len(files) > 0 {
					var reader = FileManager.Tar(files)
					var compressed = compressor.Compress(&reader)
					var name = fmt.Sprintf("%s-%s-%s", item.Name, item.Hash, folder)

					store.Upload(name, &compressed, nil)

					log.Println("Uploaded", len(files), "entries from '", folder, "' for", item.Name, item.Hash)
				} else {
					log.Println("No entries from '", folder, "' for", item.Name, item.Hash)
				}
			}
		}
	},
}
