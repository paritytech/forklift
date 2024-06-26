package Server

import (
	"fmt"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/Lib/Config"
	"forklift/Lib/Diagnostic/Time"
	"forklift/Lib/Logging"
	"forklift/Lib/Metrics"
	"forklift/Rpc"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

func Run(args []string) {

	var logger = Logging.CreateLogger("Server", 4, nil)
	var timer = Time.NewForkliftTimer()

	timer.Start("Total time")

	if isJobInBlacklist() {
		logger.Infof("Job is blacklisted, bypassing forklift")
		BypassForklift()
		return
	}

	var flWorkDir string

	var wd, _ = os.Getwd()
	flWorkDir = wd

	var rpcServer = Rpc.NewForkliftServer()
	var forkliftRpc = Rpc.NewForkliftRpc()

	var storage, _ = Storages.GetStorageDriver(Config.AppConfig)
	var compressor, _ = Compressors.GetCompressor(Config.AppConfig)
	var uploader = Rpc.NewUploader(".", storage, compressor)

	var threadsCount = Config.AppConfig.General.ThreadsCount

	if !Config.AppConfig.General.Quiet {
		fmt.Printf(
			"   Using forklift"+
				", storage: %s, compressor: %s"+
				", uploader threads: %d\n",
			Config.AppConfig.Storage.Type,
			compressor.GetKey(),
			threadsCount)
	}

	uploader.Start(forkliftRpc.Uploads, threadsCount)

	go rpcServer.Start(forkliftRpc)

	var currentPath, err = GetCurrentExecPath()
	if err != nil {
		logger.Fatalf("Failed to get current executable path: %s", err)
		os.Exit(1)
	}

	// execute cargo
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"RUSTC_WRAPPER="+currentPath,
		"FORKLIFT_WORK_DIR="+flWorkDir,
		"FORKLIFT_SOCKET="+rpcServer.SocketAddress,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	var cargoErr = cmd.Run()

	close(forkliftRpc.Uploads)
	uploader.Wait()

	logger.Infof("Uploader finish")

	forkliftRpc.StatusReport.TotalForkliftTime += timer.Stop("Total time")

	if forkliftRpc.StatusReport.TotalCrates != 0 {
		forkliftRpc.StatusReport.AverageDownloadSpeedBps = int64(float64(forkliftRpc.StatusReport.AverageDownloadSpeedBps) / float64(forkliftRpc.StatusReport.TotalCrates))
	}

	if uploader.StatusReport.Total != 0 {
		uploader.StatusReport.AverageUploadSpeedBps = int64(float64(uploader.StatusReport.AverageUploadSpeedBps) / float64(uploader.StatusReport.Total))
	}

	if !Config.AppConfig.General.Quiet {
		fmt.Printf("%s\n", forkliftRpc.StatusReport)
		fmt.Printf("%s\n", uploader.StatusReport)
	}

	var extraLabels = Config.AppConfig.Metrics.ExtraLabels
	extraLabels["storage"] = Config.AppConfig.Storage.Type
	extraLabels["compressor"] = compressor.GetKey()

	rpcServer.Stop()

	if cargoErr != nil {
		extraLabels["job_result"] = "fail"
		if (&forkliftRpc.StatusReport).TotalCrates > 0 {
			Metrics.PushMetrics(&forkliftRpc.StatusReport, &uploader.StatusReport, extraLabels)
		}
		logger.Errorf("Cargo finished with error: %s", cargoErr)
		os.Exit(1)
	} else {
		extraLabels["job_result"] = "success"
		Metrics.PushMetrics(&forkliftRpc.StatusReport, &uploader.StatusReport, extraLabels)
		logger.Infof("Cargo finished successfully")
	}
}

func GetCurrentExecPath() (string, error) {
	currentExecPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	currentExecPath, err = filepath.EvalSymlinks(currentExecPath)
	if err != nil {
		return "", err
	}

	return currentExecPath, nil
}

func getCurrentJobName() string {
	var logger = Logging.CreateLogger("Server", 4, nil)

	if Config.AppConfig.General.JobNameVariable == "" {
		logger.Debugf("JobNameVariable is not set")
		return ""
	}

	currentJobName, ok := os.LookupEnv(Config.AppConfig.General.JobNameVariable)
	if !ok {
		logger.Debugf("JobNameVariable '%s' is not set", Config.AppConfig.General.JobNameVariable)
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

	for _, blacklistedJobRegex := range Config.AppConfig.General.JobsBlacklist {
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
