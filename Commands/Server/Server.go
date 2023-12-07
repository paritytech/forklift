package Server

import (
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/Lib"
	"forklift/Rpc"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

func Run(args []string) {

	var flWorkDir string

	// set RUSTC_WRAPPER env var
	if existingVar, ok := os.LookupEnv("RUSTC_WRAPPER"); ok {
		log.Infof("RUSTC_WRAPPER is already set: %s", existingVar)
	} else {
		//var flExecPath, _ = os.Executable()
		//flExecPath, _ = filepath.EvalSymlinks(flExecPath)
		os.Setenv("RUSTC_WRAPPER", "forklift")
	}

	// set FORKLIFT_WORK_DIR env var
	if existingVar, ok := os.LookupEnv("FORKLIFT_WORK_DIR"); ok {
		log.Infof("FORKLIFT_WORK_DIR is already set: %s", existingVar)
		flWorkDir = existingVar
	} else {
		var wd, _ = os.Getwd()
		os.Setenv("FORKLIFT_WORK_DIR", wd)
		flWorkDir = wd
	}

	var rpcServer = Rpc.NewForkliftServer()
	var forkliftRpc = Rpc.NewForkliftRpc()

	var storage, _ = Storages.GetStorageDriver(Lib.AppConfig)
	var compressor, _ = Compressors.GetCompressor(Lib.AppConfig)
	var uploader = Rpc.NewUploader(".", storage, compressor)

	uploader.Start(forkliftRpc.Uploads, 2)

	go rpcServer.Start(flWorkDir, forkliftRpc)

	// execute cargo
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	_ = cmd.Run()

	close(forkliftRpc.Uploads)

	rpcServer.Stop()

	uploader.Wait()
	<-rpcServer.Channel
}
