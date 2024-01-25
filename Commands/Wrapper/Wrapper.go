package Wrapper

import (
	"bytes"
	"errors"
	"forklift/CacheStorage"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	"forklift/Lib/Diagnostic/Time"
	"forklift/Lib/Logging"
	"forklift/Lib/Rustc"
	"forklift/Rpc"
	"forklift/Rpc/Models/CacheUsage"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

var WorkDir string

func Run(args []string) {

	var rustcArgsOnly = args[1:]

	var timer = Time.NewForkliftTimer()

	wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR")

	if !ok || wd == "" {
		log.Fatalln("No `FORKLIFT_WORK_DIR` specified!")
		return
	}

	WorkDir = wd

	var wrapperTool = Rustc.NewWrapperToolFromArgs(WorkDir, &rustcArgsOnly)

	logger := Logging.CreateLogger("Wrapper", 1, log.Fields{
		"crate": wrapperTool.CrateName,
		"hash":  wrapperTool.CrateHash,
	})
	wrapperTool.Logger = logger

	var flClient = Rpc.NewForkliftRpcClient()

	var cacheUsageReport = CacheUsage.StatusReport{
		CrateName: wrapperTool.CrateName,
	}

	// Real work starts here

	// check deps
	var deps = Rustc.GetExternDeps(&rustcArgsOnly, true)
	var rebuiltDep = flClient.CheckExternDeps(deps)
	var gotRebuildDeps = true
	if rebuiltDep == "" {
		gotRebuildDeps = false
	}

	if !wrapperTool.IsNeedProcessFromCache() {
		logger.Debugf("No need to use cache for %s", wrapperTool.CrateName)
		var rustcError = BypassRustc()
		if rustcError != nil {
			var exitError *exec.ExitError
			if errors.As(rustcError, &exitError) {
				os.Exit(exitError.ExitCode())
			}
			os.Exit(1)
		}
		return
	}

	if gotRebuildDeps {
		logger.Debugf("Got rebuilt dep: %s", rebuiltDep)
		cacheUsageReport.Status = CacheUsage.DependencyRebuilt
		//flClient.ReportStatus(wrapperTool.CrateName, CacheUsage.DependencyRebuilt)
	} else {
		logger.Debugf("No rebuilt deps")
	}

	// calc sources checksum
	//if wrapperTool.IsNeedProcessFromCache() {
	calcChecksum2(wrapperTool)
	//}

	var cacheHit = false
	// try get from cache
	if !gotRebuildDeps {
		cacheHit = TryUseCache(wrapperTool, logger, &cacheUsageReport)
	}

	if !cacheHit {
		// execute rustc
		timer.Start("rustc")
		logger.Infof("Executing rustc")
		cacheUsageReport.RustcTime += timer.Stop("rustc")
		var artifacts, rustcError = ExecuteRustc(wrapperTool)

		if rustcError != nil {
			logger.Errorf("Rustc finished with error: %s", rustcError)
			var exitError *exec.ExitError
			if errors.As(rustcError, &exitError) {
				os.Exit(exitError.ExitCode())
			}
			os.Exit(1)
		}
		logger.Debugf("Finished rustc")

		// register rebuilt artifacts
		RegisterRebuiltArtifacts(artifacts, flClient)

		//if wrapperTool.IsNeedProcessFromCache() {
		flClient.AddUpload(wrapperTool.ToCacheItem())
		//}
	}

	//if wrapperTool.IsNeedProcessFromCache() {
	flClient.ReportStatusObject(cacheUsageReport)
	//}
}

// TryUseCache - try to use cache, return false if failed
func TryUseCache(wrapperTool *Rustc.WrapperTool, logger *log.Entry, cacheUsageReport *CacheUsage.StatusReport) bool {

	var timer = Time.NewForkliftTimer()

	store, _ := Storages.GetStorageDriver(Lib.AppConfig)
	compressor, _ := Compressors.GetCompressor(Lib.AppConfig)

	var retries = 3
	for retries > 0 {
		// try download
		timer.Start("download")
		f, err := store.Download(wrapperTool.GetCachePackageName() + "_" + compressor.GetKey())
		cacheUsageReport.DownloadTime += timer.Stop("download")

		if f == nil && err == nil {
			logger.Debugf("%s does not exist in storage", wrapperTool.GetCachePackageName())
			cacheUsageReport.Status = CacheUsage.CacheMiss
			return false
		}
		if err != nil {
			logger.Warningf("download error: %s", err)
			retries--
			continue
		}

		// try decompress
		timer.Start("decompress")
		decompressed, err := compressor.Decompress(f)
		cacheUsageReport.DecompressTime += timer.Stop("decompress")
		if err != nil {
			logger.Warningf("decompression error: %s", err)
			retries--
			continue
		}

		// try unpack
		timer.Start("unpack")
		err = Tar.UnPack(WorkDir, decompressed)
		cacheUsageReport.UnpackTime += timer.Stop("unpack")
		if err != nil {
			logger.Warningf("unpack error: %s", err)
			retries--
			continue
		} else {

			io.Copy(os.Stderr, wrapperTool.ReadStderrFile())
			logger.Infof("Downloaded and unpacked artifacts for %s", wrapperTool.GetCachePackageName())

			if retries == 3 {
				cacheUsageReport.Status = CacheUsage.CacheHit
			} else {
				cacheUsageReport.Status = CacheUsage.CacheHitWithRetry
			}

			return true
		}
	}
	if retries <= 0 {
		logger.Errorf("Failed to pull artifacts for %s", wrapperTool.GetCachePackageName())
		cacheUsageReport.Status = CacheUsage.CacheFetchFailed
	}
	return false
}

func BypassRustc() error {
	var rustcArgsOnly = os.Args[2:]
	var cmd = exec.Command(os.Args[1], rustcArgsOnly...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	var rustcError = cmd.Run()

	return rustcError
}

// ExecuteRustc - execute rustc and process output
func ExecuteRustc(wrapperTool *Rustc.WrapperTool) (*[]CacheStorage.RustcArtifact, error) {

	cmd := exec.Command(os.Args[1], os.Args[2:]...)

	var (
		rustcStdout = bytes.Buffer{}
		rustcStderr = bytes.Buffer{}
		rustcStdin  = bytes.Buffer{}
	)

	cmd.Stdout = io.MultiWriter(os.Stdout, &rustcStdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &rustcStderr)

	rustcStdin2 := bytes.Buffer{}
	stdinWriter := io.MultiWriter(&rustcStdin2, &rustcStdin)
	cmd.Stdin = &rustcStdin
	io.Copy(stdinWriter, os.Stdin)

	var runErr = cmd.Run()
	if runErr != nil {
		return nil, runErr
	}

	wrapperTool.WriteIOStreamFile(&rustcStdout, "stdout")
	artifacts := wrapperTool.WriteStderrFile(&rustcStderr)
	wrapperTool.WriteIOStreamFile(&rustcStdin2, "stdin")

	return artifacts, nil
}

// RegisterRebuiltArtifacts - register rebuilt artifacts
func RegisterRebuiltArtifacts(artifacts *[]CacheStorage.RustcArtifact, flClient *Rpc.ForkliftRpcClient) {
	var artifactsPaths = make([]string, 0)
	for _, artifact := range *artifacts {
		var abs = filepath.Base(artifact.Artifact)

		artifactsPaths = append(artifactsPaths, abs)
	}
	flClient.RegisterExternDeps(&artifactsPaths)
}
