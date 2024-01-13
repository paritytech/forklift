package Server

import (
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/Lib"
	"forklift/Rpc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"regexp"
)

func Run(args []string) {

	var err = viper.Unmarshal(&Lib.AppConfig)
	if err != nil {
		log.Errorln(err)
	}

	logLevel, err := log.ParseLevel(Lib.AppConfig.General.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel
		log.Debugf("unknown log level (verbose) `%s`, using default `info`\n", Lib.AppConfig.General.LogLevel)
	}

	log.SetLevel(logLevel)

	if isJobInBlacklist() {
		log.Infof("Job is blacklisted, bypassing forklift")
		BypassForklift()
		return
	}

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

	var threadsCount = Lib.AppConfig.General.ThreadsCount
	if threadsCount <= 0 {
		threadsCount = 2
	}
	log.Infof("Uploader threads: %d", threadsCount)
	uploader.Start(forkliftRpc.Uploads, threadsCount)

	go rpcServer.Start(flWorkDir, forkliftRpc)

	// execute cargo
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Run()

	close(forkliftRpc.Uploads)
	uploader.Wait()

	log.Infof("Uploader finish")

	log.Infof("%s", forkliftRpc.StatusReport)

	rpcServer.Stop()
	<-rpcServer.Channel

	if err != nil {
		log.Errorf("Cargo finished with error: %s", err)
		os.Exit(1)
	} else {
		log.Infof("Cargo finished successfully")
	}
}

func isJobInBlacklist() bool {
	if Lib.AppConfig.General.JobNameVariable == "" {
		log.Debugf("JobNameVariable is not set")
		return false
	}

	currentJobName, ok := os.LookupEnv(Lib.AppConfig.General.JobNameVariable)
	if !ok {
		log.Debugf("JobNameVariable '%s' is not set", Lib.AppConfig.General.JobNameVariable)
		return false
	}
	log.Infof("Current job name is '%s'", currentJobName)

	for _, blacklistedJobRegex := range Lib.AppConfig.General.JobsBlacklist {
		match, _ := regexp.MatchString(blacklistedJobRegex, currentJobName)

		if match {
			log.Infof("Job %s is blacklisted by '%s'", currentJobName, blacklistedJobRegex)
			return true
		}
	}

	log.Debugf("Job %s is not blacklisted", currentJobName)
	return false
}

func BypassForklift() {

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		log.Errorf("Finished with error: %s", err)
		os.Exit(1)
	}
}
