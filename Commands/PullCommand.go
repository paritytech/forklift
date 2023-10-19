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
			var cacheItems = FileManager.ParseCacheRequest()

			store, _ := Storages.GetStorageDriver(Lib.AppConfig)
			compressor, _ := Compressors.GetCompressor(Lib.AppConfig)
			//var folders = []string{"build", "deps", ".fingerprint"}

			var cpuCount = runtime.NumCPU()

			var queue = make(chan struct {
				item Models.CacheItem
				path string
			})

			go func() {
				//for _, folder := range folders {

				var path = filepath.Join("target", Lib.AppConfig.General.Dir)

				for _, item := range cacheItems {
					queue <- struct {
						item Models.CacheItem
						path string
					}{item: item, path: path}
				}
				//}
				close(queue)
			}()

			Lib.Parallel(
				queue,
				cpuCount,
				func(obj struct {
					item Models.CacheItem
					path string
				}) {
					var name = fmt.Sprintf("%s-%s-%s", obj.item.Name, obj.item.Hash, compressor.GetKey())
					var meta, existsInStore = store.GetMetadata(name)
					log.Traceln(name, meta)

					var needDownload = true

					if !existsInStore {
						log.Debugf("%s does not exist in storage\n", name)
						return
					} else if meta == nil {
						log.Debugf("no metadata for %s, downloading...\n", name)
						needDownload = true
					} else if shaRemotePtr, ok := meta["sha-1-content"]; !ok {
						log.Debugf("no metadata header for %s, downloading...\n", name)
						needDownload = true
					} else {

						var files = FileManager.Find(obj.path, obj.item.Hash, true)

						if len(files) <= 0 {
							log.Debugf("%s no local files, downloading...\n", name)
							needDownload = true
						} else {
							var _, sha = Tar.Pack(files)
							var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))

							var shaRemote = *shaRemotePtr

							if shaRemote != shaLocal {
								log.Debugf("%s checksum mismatch, remote: %s local: %s, downloading...\n", name, shaRemote, shaLocal)
								needDownload = true
							} else {
								log.Tracef("%s checksum match , remote: %s local: %s\n", name, shaRemote, shaLocal)
								needDownload = false
							}
						}
					}

					if !needDownload {
						return
					}

					var f = store.Download(name)
					if f != nil {
						Tar.UnPack(".", compressor.Decompress(f))
						log.Infof("Downloaded artifacts for %s\n", name)
					}
				})

			/*
				if len(extraDirs) > 0 {
					var extraDirQueue = make(chan struct {
						key string
					}, 20)

					go func() {
						for _, extraDir := range extraDirs {

							var exit = false
							var i = 0
							for !exit {

								var key = fmt.Sprintf("%s_%s_%s_%s_%d", extraDir, mode, "latest", compressor.GetKey(), i)

								var _, exists = store.GetMetadata(key)
								if !exists {
									exit = true
									break
								}
								extraDirQueue <- struct{ key string }{key: key}
								log.Tracef("Enqueued %s\n", key)
								i++
							}

							close(extraDirQueue)
						}
					}()

					Lib.Parallel(
						extraDirQueue,
						cpuCount,
						func(obj struct {
							key string
						}) {
							var f = store.Download(obj.key)
							if f != nil {
								Tar.UnPack(".", compressor.Decompress(&f))
								log.Infof("Downloaded artifacts %s for extra dir\n", obj.key)
							}
						})
				}*/
	},
}
