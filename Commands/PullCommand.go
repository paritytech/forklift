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
	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull [flags] [project_dir]",
	Short: "Download cache artifacts",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) > 0 {
			err := os.Chdir(args[0])
			if err != nil {
				log.Println(err)
				return
			}
		}

		var cacheItems = FileManager.ParseCacheRequest()

		store, _ := Storages.GetStorageDriver(storage, &params)
		compressor, _ := Compressors.GetCompressor(compression)
		var folders = []string{"build", "deps", ".fingerprint"}

		for _, folder := range folders {
			var path = filepath.Join("target", mode, folder)

			for i, item := range cacheItems {

				var name = fmt.Sprintf("%s-%s-%s", item.Name, item.Hash, folder)
				var f = store.Download(name)
				if f != nil {
					FileManager.UnTar(path, compressor.Decompress(&f))
				}

				/*f = store.Download(item.Hash + "-deps")
				if f != nil {
					FileManager.UnTar("./target/debug/deps", compressor.Decompress(&f))
				}

				f = store.Download(item.Hash + "-fp")
				if f != nil {
					FileManager.UnTar("./target/debug/.fingerprint", compressor.Decompress(&f))
				}*/

				log.Println("Downloaded artifacts for", item.Name, item.Hash, folder, i+1, "/", len(cacheItems))
			}
		}
	},
}
