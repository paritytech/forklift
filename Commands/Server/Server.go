package Server

import (
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/Lib"
	"forklift/Lib/Diagnostic/Time"
	"forklift/Lib/Logging"
	"forklift/Lib/Metrics"
	"forklift/Rpc"
	"os"
	"os/exec"
	"regexp"
)

func Run(args []string) {

	var logger = Logging.CreateLogger("Server", 4, nil)

	Time.Start("Total time")

	if isJobInBlacklist() {
		logger.Infof("Job is blacklisted, bypassing forklift")
		BypassForklift()
		return
	}

	var flWorkDir string

	// set RUSTC_WRAPPER env var
	if existingVar, ok := os.LookupEnv("RUSTC_WRAPPER"); ok {
		logger.Infof("RUSTC_WRAPPER is already set: %s", existingVar)
	} else {
		//var flExecPath, _ = os.Executable()
		//flExecPath, _ = filepath.EvalSymlinks(flExecPath)
		os.Setenv("RUSTC_WRAPPER", "forklift")
	}

	// set FORKLIFT_WORK_DIR env var
	if existingVar, ok := os.LookupEnv("FORKLIFT_WORK_DIR"); ok {
		logger.Infof("FORKLIFT_WORK_DIR is already set: %s", existingVar)
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
	logger.Infof("Uploader threads: %d", threadsCount)
	uploader.Start(forkliftRpc.Uploads, threadsCount)

	go rpcServer.Start(flWorkDir, forkliftRpc)

	// execute cargo
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	var err = cmd.Run()

	close(forkliftRpc.Uploads)
	uploader.Wait()

	logger.Infof("Uploader finish")

	forkliftRpc.StatusReport.TotalForkliftTime += Time.Stop("Total time")

	logger.Infof("%s", forkliftRpc.StatusReport)

	Metrics.PushMetrics(&forkliftRpc.StatusReport, map[string]string{
		"job_name": "smth 1/3",
	})

	rpcServer.Stop()

	if err != nil {
		logger.Errorf("Cargo finished with error: %s", err)
		os.Exit(1)
	} else {
		logger.Infof("Cargo finished successfully")
	}
}

func getCurrentJobName() string {
	var logger = Logging.CreateLogger("Server", 4, nil)

	if Lib.AppConfig.General.JobNameVariable == "" {
		logger.Debugf("JobNameVariable is not set")
		return ""
	}

	currentJobName, ok := os.LookupEnv(Lib.AppConfig.General.JobNameVariable)
	if !ok {
		logger.Debugf("JobNameVariable '%s' is not set", Lib.AppConfig.General.JobNameVariable)
		return ""
	}

	return currentJobName
}

// isJobInBlacklist - check if current job is in blacklist
func isJobInBlacklist() bool {
	var logger = Logging.CreateLogger("Server", 4, nil)

	var currentJobName = getCurrentJobName()

	if currentJobName == "" {
		return false
	}

	for _, blacklistedJobRegex := range Lib.AppConfig.General.JobsBlacklist {
		match, _ := regexp.MatchString(blacklistedJobRegex, currentJobName)

		if match {
			logger.Infof("Job %s is blacklisted by '%s'", currentJobName, blacklistedJobRegex)
			return true
		}
	}

	logger.Debugf("Job %s is not blacklisted", currentJobName)
	return false
}

// BypassForklift - bypass forklift
func BypassForklift() {

	var logger = Logging.CreateLogger("Server", 4, nil)

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		logger.Errorf("Finished with error: %s", err)
		os.Exit(1)
	}
}
