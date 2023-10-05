package Commands

import (
	"archive/tar"
	"bytes"
	"crypto/sha1"
	"fmt"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager"
	"forklift/FileManager/Models"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

var forcePush bool

func init() {
	pushCmd.PersistentFlags().BoolVarP(&forcePush, "force-push", "f", false, "")

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
		//var folders = []string{"build", "deps", ".fingerprint"}

		/*var smth = FileManager.FindAll(cacheItems, "target/release")

		for _, items := range smth {
			for _, item := range *items {
				fmt.Println(item)
			}
		}*/

		var cpuCount = runtime.NumCPU()

		var queue = make(chan struct {
			item    Models.CacheItem
			path    string
			entries []Models.TargetFsEntry
		}, 20)

		go func() {
			var path = filepath.Join("target", mode)

			var dict = make(map[string]Models.CacheItem)

			for _, item := range cacheItems {
				dict[item.Hash] = item
			}

			var smth = FileManager.FindAll(cacheItems, path)

			for key, items := range smth {
				//for _, item := range *items {
				queue <- struct {
					item    Models.CacheItem
					path    string
					entries []Models.TargetFsEntry
				}{item: dict[key], path: path, entries: *items}
				//}
			}
			/*
				for _, item := range cacheItems {
					queue <- struct {
						item Models.CacheItem
						path string
					}{item: item, path: path}
				}*/

			close(queue)
		}()

		Lib.Parallel(
			queue,
			cpuCount,
			func(obj struct {
				item    Models.CacheItem
				path    string
				entries []Models.TargetFsEntry
			}) {
				log.Tracef("Processing %s %s %s\n", obj.path, obj.item.Name, obj.item.Hash)
				//var files = FileManager.Find(obj.path, obj.item.Hash, true)
				//log.Tracef("Found %d entries for %s %s\n", len(files), obj.path, obj.item.Hash)

				if len(obj.entries) > 0 {

					var reader, sha = Tar.Pack(obj.entries)
					var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))

					var name = fmt.Sprintf("%s-%s-%s", obj.item.Name, obj.item.Hash, compressor.GetKey())

					var meta, exists = store.GetMetadata(name)

					var needUpload = false

					if forcePush {
						needUpload = true
					} else if !exists {
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
						log.Infof("Uploaded %d entries for %s-%s, %x\n", len(obj.entries), obj.item.Name, obj.item.Hash, sha.Sum(nil))
					}
				} else {
					log.Tracef("No entries for %s-%s\n", obj.item.Name, obj.item.Hash)
				}
			})

		if len(extraDirs) > 0 {

			var extraDirQueue = make(chan struct {
				path string
				name string
			}, 20)

			go func() {
				for _, extraDir := range extraDirs {
					var extraDirPath = filepath.Join("target", mode, extraDir)

					var extraDirFiles, _ = os.ReadDir(extraDirPath)

					for i, file := range extraDirFiles {
						if !file.IsDir() {
							continue
						}
						log.Tracef("Enqueued %s\n", filepath.Join(extraDirPath, file.Name()))
						extraDirQueue <- struct {
							path string
							name string
						}{
							path: filepath.Join(extraDirPath, file.Name()),
							name: fmt.Sprintf("%s_%s_%s_%s_%d", extraDir, mode, "latest", compressor.GetKey(), i),
						}
					}

					close(extraDirQueue)
				}
			}()

			Lib.Parallel(
				extraDirQueue,
				cpuCount,
				func(obj struct {
					path string
					name string
				}) {
					log.Tracef("Processing %s\n", obj.name)
					var buf bytes.Buffer
					tw := tar.NewWriter(&buf)
					var sha = sha1.New()

					var reader io.Reader = &buf

					Tar.PackDirectory(tw, obj.path, sha)

					var meta, exists = store.GetMetadata(obj.name)
					var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))

					var needUpload = false

					if forcePush {
						needUpload = true
					} else if !exists {
						log.Debugf("%s does not exist in storage, uploading...\n", obj.name)
						needUpload = true
					} else if meta == nil {
						log.Debugf("no metadata for %s, uploading...\n", obj.name)
						needUpload = true
					} else if shaRemotePtr, ok := meta["sha-1-content"]; !ok {
						log.Debugf("no metadata for %s, uploading...\n", obj.name)
						needUpload = true
					} else if *shaRemotePtr != shaLocal {
						log.Debugf("%s checksum mismatch, remote: %s, local: %s, uploading...\n", obj.name, *shaRemotePtr, shaLocal)
						needUpload = true
					}

					if needUpload {
						var compressed = compressor.Compress(&reader)
						store.Upload(obj.name, &compressed, map[string]*string{"sha-1-content": &shaLocal})
					}

					log.Tracef("Uploaded %s\n", obj.name)
				})

		}
	},
}
