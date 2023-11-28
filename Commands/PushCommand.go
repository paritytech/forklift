package Commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"forklift/CacheStorage"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	"forklift/Lib/Rustc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
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
		if Lib.AppConfig.General.ThreadsCount > 0 {
			cpuCount = Lib.AppConfig.General.ThreadsCount
		}

		log.Debugf("ThreadsCount: %d", cpuCount)

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
				log.Debugf("Processing %s %s %s\n", wrapperTool.CrateName, wrapperTool.CrateHash, wrapperTool.OutDir)

				var crateArtifactsFiles []string = []string{
					path.Join("target", "forklift", fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), "stderr")),
					path.Join("target", "forklift", fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), "stdout")),
					path.Join("target", "forklift", fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), "stdin")),
				}

				//crateArtifactsFiles = append(crateArtifactsFiles, FileManager.FindBuildFiles(wrapperTool.CrateHash)...)

				var stderrFile = wrapperTool.ReadStderrFile()
				fileScanner := bufio.NewScanner(stderrFile)
				fileScanner.Split(bufio.ScanLines)
				for fileScanner.Scan() {
					var artifact CacheStorage.RustcArtifact
					json.Unmarshal([]byte(fileScanner.Text()), &artifact)
					if artifact.Artifact != "" {
						if strings.Contains(artifact.Artifact, "tmp/") ||
							strings.Contains(artifact.Artifact, "/var/folders/") {
							log.Debugf("Temporary artifact folder `%s` detected, skip", artifact.Artifact)
							return
						}
						var relPath, _ = filepath.Rel(WorkDir, artifact.Artifact)
						crateArtifactsFiles = append(crateArtifactsFiles, relPath)
					}
				}

				log.Debugf("%s", crateArtifactsFiles)

				if len(crateArtifactsFiles) > 0 {
					var reader, sha = Tar.Pack(crateArtifactsFiles)

					//Tar.PackDirectory()
					//var reader, sha = Tar.Pack(obj.entries)
					var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))

					var name = wrapperTool.GetCachePackageName() // fmt.Sprintf("%s-%s-%s", obj.item.Name, obj.item.Hash, compressor.GetKey())

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

					var metaMap = wrapperTool.CreateMetadata()
					metaMap["sha1-artifact"] = &shaLocal

					if needUpload {
						var compressed = compressor.Compress(reader)
						store.Upload(name+"_"+compressor.GetKey(), &compressed, metaMap)

						marshal, err := json.Marshal(metaMap)
						if err != nil {
							return
						}

						log.Infof("Uploaded %s, metadata: %s", wrapperTool.GetCachePackageName(), marshal)
					}
				} else {
					log.Tracef("No entries for %s-%s\n", wrapperTool.GetCachePackageName(), wrapperTool.CrateHash)
				}
			})
	},
}
