package Commands

import (
	"fmt"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(unpackCmd)
}

var unpackCmd = &cobra.Command{
	Use:   "unpack <artifact name>",
	Short: "Download and unpack artifact",
	Run: func(cmd *cobra.Command, args []string) {

		store, _ := Storages.GetStorageDriver(Lib.AppConfig)
		compressor, _ := Compressors.GetCompressor(Lib.AppConfig)

		var name = fmt.Sprintf("%s_%s", args[0], compressor.GetKey())
		var _, existsInStore = store.GetMetadata(name)

		if !existsInStore {
			log.Infof("%s does not exists in store", name)
			return
		}

		var f = store.Download(name)
		if f != nil {
			Tar.UnPack(".", compressor.Decompress(f))
			log.Infof("Downloaded artifacts for %s\n", name)
		}
	},
}
