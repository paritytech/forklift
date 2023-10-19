package Commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"forklift/CacheStorage"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/Commands/Wrapper"
	"forklift/FileManager"
	"forklift/FileManager/Models"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path"
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

		log.Info(Lib.AppConfig)

		if len(args) > 0 {
			err := os.Chdir(args[0])
			if err != nil {
				log.Errorln(err)
				return
			}
		}

		var WorkDir, _ = os.Getwd()

		store, _ := Storages.GetStorageDriver(Lib.AppConfig)
		compressor, _ := Compressors.GetCompressor(Lib.AppConfig)

		var cacheItems = FileManager.ParseCacheRequest()

		log.Infof("cacheItems %d", len(cacheItems))

		var cpuCount = runtime.NumCPU()

		var queue = make(chan Models.CacheItem, 20)

		go func() {
			for _, item := range cacheItems {
				queue <- item
			}

			close(queue)
		}()

		Lib.Parallel(
			queue,
			cpuCount,
			func(item Models.CacheItem) {
				log.Debugf("Processing %s %s %s\n", item.Name, item.Hash, item.OutDir)
				//var files = FileManager.Find(obj.path, obj.item.Hash, true)
				//log.Tracef("Found %d entries for %s %s\n", len(files), obj.path, obj.item.Hash)

				var crateArtifactsFiles []string = []string{
					path.Join("target", Lib.AppConfig.General.Dir, "forklift", fmt.Sprintf("%s-%s", item.CachePackageName, "stderr")),
					path.Join("target", Lib.AppConfig.General.Dir, "forklift", fmt.Sprintf("%s-%s", item.CachePackageName, "stdout")),
					path.Join("target", Lib.AppConfig.General.Dir, "forklift", fmt.Sprintf("%s-%s", item.CachePackageName, "stdin")),
				}

				var stderrFile = Wrapper.ReadIOStreamFile(item.CachePackageName, "stderr")
				fileScanner := bufio.NewScanner(stderrFile)
				fileScanner.Split(bufio.ScanLines)
				for fileScanner.Scan() {
					var artifact CacheStorage.RustcArtifact
					json.Unmarshal([]byte(fileScanner.Text()), &artifact)
					if artifact.Artifact != "" {
						var relpath, _ = filepath.Rel(WorkDir, artifact.Artifact)
						crateArtifactsFiles = append(crateArtifactsFiles, relpath)
						log.Debug(crateArtifactsFiles)
					}
				}

				log.Debugf("%s", crateArtifactsFiles)
				//return

				log.Debugf("!!!!!!!!!!!!!!")

				if len(crateArtifactsFiles) > 0 {
					var reader, sha = Tar.Pack(crateArtifactsFiles)

					//Tar.PackDirectory()
					//var reader, sha = Tar.Pack(obj.entries)
					var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))

					var name = item.CachePackageName // fmt.Sprintf("%s-%s-%s", obj.item.Name, obj.item.Hash, compressor.GetKey())

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
						var compressed = compressor.Compress(reader)
						store.Upload(name, &compressed, map[string]*string{"sha-1-content": &shaLocal})
						log.Infof("Uploaded %s-%s, %x\n", item.Name, item.Hash, sha.Sum(nil))
					}
				} else {
					log.Tracef("No entries for %s-%s\n", item.Name, item.Hash)
				}
			})
	},
}
