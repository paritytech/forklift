package Commands

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull [flags] [project_dir]",
	Short: "Download cache artifacts",
	//Deprecated: "Use forklift as RUSTC_WRAPPER instead",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) > 0 {
			err := os.Chdir(args[0])
			if err != nil {
				log.Fatalln(err)
				return
			}
		}
		/*
			var WorkDir, _ = os.Getwd()

			var cacheItems = FileManager.ParseCacheRequest()

			store, _ := Storages.GetStorageDriver(Lib.AppConfig)
			compressor, _ := Compressors.GetCompressor(Lib.AppConfig)
			//var folders = []string{"build", "deps", ".fingerprint"}

			var cpuCount = runtime.NumCPU()

			var queue = make(chan *Rustc.WrapperTool, 20)
			go func() {
				for _, item := range cacheItems {
					queue <- Rustc.NewWrapperToolFromCacheItem(WorkDir, item)
				}

				close(queue)
			}()

			Lib.Parallel(
				queue,
				cpuCount,
				func(wrapperTool *Rustc.WrapperTool) {
					var name = wrapperTool.CachePackageName
					var meta, existsInStore = store.GetMetadata(name + "-" + compressor.GetKey())
					log.Traceln(name, meta)

					var needDownload = true

					if !existsInStore {
						log.Debugf("%s does not exist in storage\n", name)
						needDownload = false
						return
					}

					if !needDownload {
						return
					}

					var f = store.Download(name + "-" + compressor.GetKey())
					if f != nil {
						Tar.UnPack(".", compressor.Decompress(f))
						log.Infof("Downloaded artifacts for %s\n", name)
					}
				})*/

	},
}
